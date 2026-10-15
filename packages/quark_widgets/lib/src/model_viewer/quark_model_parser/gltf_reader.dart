import 'dart:convert';
import 'dart:typed_data';

import '../quark_model_mesh.dart';
import '../quark_model_parser.dart';
import 'mesh_builder.dart';

const int _glbMagic = 0x46546C67; // 'glTF'
const int _jsonChunk = 0x4E4F534A; // 'JSON'
const int _binChunk = 0x004E4942; // 'BIN\0'

const int _triangles = 4;
const int _triangleStrip = 5;
const int _triangleFan = 6;

/// Extensions whose geometry cannot be read without a decoder this reader
/// does not have. A file that requires one is refused rather than drawn
/// wrong.
const Set<String> _undecodable = {
  'KHR_draco_mesh_compression',
  'EXT_meshopt_compression',
  'KHR_meshopt_compression',
};

/// The sibling files a glTF's buffers name, by file name, in buffer order.
/// Embedded buffers (a GLB's binary chunk, a `data:` URI) are not listed,
/// and neither is a URI that is not a plain file name beside the model —
/// [readGltf] refuses those if a mesh reads from them.
List<String> gltfSiblingFiles(Uint8List bytes) {
  final (json, _) = _split(bytes);
  return [
    for (final buffer in _array(json, 'buffers'))
      if (_object(buffer, 'buffer')['uri'] case final String uri)
        if (_siblingName(uri) case final String name) name,
  ];
}

/// Reads a glTF 2.0 file, JSON (`.gltf`) or binary (`.glb`), told apart by
/// the binary header's magic.
///
/// Every triangle primitive (indexed or not; lists, strips and fans) of
/// every mesh the default scene reaches is drawn, with its nodes'
/// transforms applied, as one mesh. Buffers come from the binary chunk, a
/// `data:` URI, or [siblings] keyed by file name; nothing is ever fetched.
/// Materials, textures, skins, morph targets and animation are ignored.
QuarkModelMesh readGltf(
  Uint8List bytes,
  Map<String, Uint8List> siblings,
  QuarkModelLimits limits,
) {
  final (json, bin) = _split(bytes);
  for (final name in _array(json, 'extensionsRequired')) {
    if (_undecodable.contains(name)) _bad('requires $name');
  }
  final reader = _GltfReader(json, bin, siblings, MeshBuilder(limits));
  reader.readScene();
  return reader.builder.build(isZUp: false);
}

/// The JSON document of a `.gltf` or `.glb`, and the GLB's binary chunk.
(Map<String, Object?>, Uint8List?) _split(Uint8List bytes) {
  final data = ByteData.sublistView(bytes);
  Uint8List text = bytes;
  Uint8List? bin;
  if (bytes.length >= 12 && data.getUint32(0, Endian.little) == _glbMagic) {
    if (data.getUint32(4, Endian.little) != 2) _bad('GLB version');
    final end = data.getUint32(8, Endian.little);
    if (end > bytes.length) _bad('GLB of ${bytes.length} bytes claims $end');
    final chunks = <int, Uint8List>{};
    for (var at = 12; at + 8 <= end;) {
      final length = data.getUint32(at, Endian.little);
      final type = data.getUint32(at + 4, Endian.little);
      at += 8;
      if (at + length > end) _bad('GLB chunk runs past the end');
      chunks.putIfAbsent(
        type,
        () => Uint8List.sublistView(bytes, at, at + length),
      );
      at += length;
    }
    text = chunks[_jsonChunk] ?? _bad('GLB has no JSON chunk');
    bin = chunks[_binChunk];
  }
  final Object? decoded;
  try {
    decoded = utf8.decoder.fuse(json.decoder).convert(text);
  } on FormatException catch (e) {
    _bad('JSON: ${e.message}');
  }
  final document = _object(decoded, 'document');
  final version = _object(document['asset'], 'asset')['version'];
  if (version is! String || !version.startsWith('2.')) {
    _bad('glTF version $version');
  }
  return (document, bin);
}

/// Walks a glTF document's scene into a [MeshBuilder].
class _GltfReader {
  _GltfReader(this.json, this.bin, this.siblings, this.builder)
    : _buffers = List.filled(_array(json, 'buffers').length, null);

  final Map<String, Object?> json;
  final Uint8List? bin;
  final Map<String, Uint8List> siblings;
  final MeshBuilder builder;
  final List<Uint8List?> _buffers;

  /// Draws every mesh the default scene reaches. A file without scenes
  /// draws its root nodes, and one without nodes draws every mesh as is.
  void readScene() {
    final nodes = _array(json, 'nodes');
    final scenes = _array(json, 'scenes');
    final List<Object?> roots;
    if (scenes.isNotEmpty) {
      final scene = _object(_at(scenes, _int(json, 'scene', 0)), 'scene');
      roots = _array(scene, 'nodes');
    } else if (nodes.isNotEmpty) {
      final children = {
        for (final node in nodes) ..._array(_object(node, 'node'), 'children'),
      };
      roots = [
        for (var i = 0; i < nodes.length; i++)
          if (!children.contains(i)) i,
      ];
    } else {
      for (final mesh in _array(json, 'meshes')) {
        _mesh(_object(mesh, 'mesh'), _identity());
      }
      return;
    }
    // Nodes form trees, so none is reached twice; one that is would loop
    // forever or draw twice.
    final seen = List.filled(nodes.length, false);
    final stack = <(Object?, Float64List)>[
      for (final root in roots) (root, _identity()),
    ];
    while (stack.isNotEmpty) {
      builder.tick();
      final (index, parent) = stack.removeLast();
      if (index is! int || index < 0 || index >= nodes.length) {
        _bad('node $index of ${nodes.length}');
      }
      if (seen[index]) _bad('node $index reached twice');
      seen[index] = true;
      final node = _object(nodes[index], 'node');
      final matrix = _multiply(parent, _localMatrix(node));
      if (node['mesh'] != null) {
        _mesh(
          _object(_at(_array(json, 'meshes'), _int(node, 'mesh')), 'mesh'),
          matrix,
        );
      }
      for (final child in _array(node, 'children')) {
        stack.add((child, matrix));
      }
    }
  }

  void _mesh(Map<String, Object?> mesh, Float64List matrix) {
    // A mirroring transform turns every triangle inside out; swapping two
    // corners keeps the normals pointing outward.
    final flip = _determinant(matrix) < 0;
    for (final primitive in _array(mesh, 'primitives')) {
      _primitive(_object(primitive, 'primitive'), matrix, flip);
    }
  }

  void _primitive(Map<String, Object?> primitive, Float64List m, bool flip) {
    final mode = _int(primitive, 'mode', _triangles);
    // Points and lines have no surface to draw.
    if (mode != _triangles && mode != _triangleStrip && mode != _triangleFan) {
      return;
    }
    final attributes = _object(primitive['attributes'], 'attributes');
    if (attributes['POSITION'] == null) return;
    final positions = _accessor(_int(attributes, 'POSITION'), 'VEC3');
    final base = builder.vertexCount;
    for (var i = 0; i < positions.count; i++) {
      builder.tick();
      final x = positions.read(i, 0);
      final y = positions.read(i, 1);
      final z = positions.read(i, 2);
      builder.add(
        m[0] * x + m[4] * y + m[8] * z + m[12],
        m[1] * x + m[5] * y + m[9] * z + m[13],
        m[2] * x + m[6] * y + m[10] * z + m[14],
      );
    }
    final indices = primitive['indices'] == null
        ? null
        : _accessor(_int(primitive, 'indices'), 'SCALAR', indices: true);
    final count = indices?.count ?? positions.count;
    int corner(int i) {
      final index = indices == null ? i : indices.readIndex(i);
      if (index >= positions.count) {
        _bad('index $index of ${positions.count} vertices');
      }
      return base + index;
    }

    void triangle(int a, int b, int c) {
      builder.tick();
      final (ca, cb, cc) = (corner(a), corner(b), corner(c));
      flip ? builder.triangle(ca, cc, cb) : builder.triangle(ca, cb, cc);
    }

    switch (mode) {
      case _triangles:
        for (var i = 0; i + 2 < count; i += 3) {
          triangle(i, i + 1, i + 2);
        }
      case _triangleStrip:
        for (var i = 0; i + 2 < count; i++) {
          i.isEven ? triangle(i, i + 1, i + 2) : triangle(i, i + 2, i + 1);
        }
      case _triangleFan:
        for (var i = 0; i + 2 < count; i++) {
          triangle(i + 1, i + 2, 0);
        }
    }
  }

  _Accessor _accessor(int index, String type, {bool indices = false}) {
    final accessor = _object(_at(_array(json, 'accessors'), index), 'accessor');
    if (accessor['type'] != type) _bad('accessor $index is not $type');
    if (accessor['sparse'] != null) _bad('sparse accessor $index');
    final componentType = _int(accessor, 'componentType');
    final componentSize = switch (componentType) {
      5120 || 5121 => 1,
      5122 || 5123 => 2,
      5125 || 5126 => 4,
      _ => _bad('component type $componentType'),
    };
    if (indices && !const {5121, 5123, 5125}.contains(componentType)) {
      _bad('index component type $componentType');
    }
    final components = type == 'VEC3' ? 3 : 1;
    final count = _int(accessor, 'count');
    if (count < 0) _bad('accessor $index count $count');
    final result = _Accessor(
      componentType: componentType,
      componentSize: componentSize,
      components: components,
      count: count,
      normalized: accessor['normalized'] == true,
    );
    if (accessor['bufferView'] == null) return result;
    final view = _object(
      _at(_array(json, 'bufferViews'), _int(accessor, 'bufferView')),
      'bufferView',
    );
    final buffer = _buffer(_int(view, 'buffer'));
    final viewStart = _int(view, 'byteOffset', 0);
    final viewLength = _int(view, 'byteLength');
    if (viewStart < 0 ||
        viewLength < 0 ||
        viewStart + viewLength > buffer.length) {
      _bad('bufferView past its buffer of ${buffer.length} bytes');
    }
    final elementSize = componentSize * components;
    final stride = _int(view, 'byteStride', elementSize);
    final offset = _int(accessor, 'byteOffset', 0);
    if (stride < elementSize ||
        offset < 0 ||
        (count > 0 &&
            offset + stride * (count - 1) + elementSize > viewLength)) {
      _bad('accessor $index past its bufferView');
    }
    return result
      ..data = ByteData.sublistView(buffer, viewStart, viewStart + viewLength)
      ..offset = offset
      ..stride = stride;
  }

  Uint8List _buffer(int index) {
    final buffers = _array(json, 'buffers');
    final cached = _buffers[_checkIndex(buffers, index)];
    if (cached != null) return cached;
    final buffer = _object(buffers[index], 'buffer');
    final byteLength = _int(buffer, 'byteLength');
    final uri = buffer['uri'];
    final Uint8List bytes;
    if (uri == null) {
      // Only the first buffer of a GLB may leave out its URI: it is the
      // binary chunk.
      bytes = index == 0
          ? bin ?? _bad('no binary chunk')
          : _bad('buffer $index has no uri');
    } else if (uri is! String) {
      _bad('buffer $index uri');
    } else if (uri.startsWith('data:')) {
      try {
        bytes = UriData.parse(uri).contentAsBytes();
      } on FormatException catch (e) {
        _bad('buffer $index data URI: ${e.message}');
      }
    } else {
      final name =
          _siblingName(uri) ?? _bad('buffer $index is not a sibling: $uri');
      bytes = siblings[name] ?? _bad('buffer $index: $name was not read');
    }
    if (bytes.length < byteLength) {
      _bad('buffer $index has ${bytes.length} of $byteLength bytes');
    }
    return _buffers[index] = bytes;
  }
}

/// One accessor's view of its buffer: [count] elements of [components]
/// numbers each. Without [data] every number is zero, as glTF says of an
/// accessor with no buffer view.
class _Accessor {
  _Accessor({
    required this.componentType,
    required this.componentSize,
    required this.components,
    required this.count,
    required this.normalized,
  });

  final int componentType;
  final int componentSize;
  final int components;
  final int count;
  final bool normalized;
  ByteData? data;
  int offset = 0;
  int stride = 0;

  /// Component [c] of element [i], scaled to -1..1 or 0..1 when
  /// [normalized].
  double read(int i, int c) {
    final data = this.data;
    if (data == null) return 0;
    final at = offset + i * stride + c * componentSize;
    return switch (componentType) {
      5126 => data.getFloat32(at, Endian.little),
      5120 when normalized => (data.getInt8(at) / 127).clamp(-1.0, 1.0),
      5121 when normalized => data.getUint8(at) / 255,
      5122 when normalized => (data.getInt16(at, Endian.little) / 32767).clamp(
        -1.0,
        1.0,
      ),
      5123 when normalized => data.getUint16(at, Endian.little) / 65535,
      5120 => data.getInt8(at).toDouble(),
      5121 => data.getUint8(at).toDouble(),
      5122 => data.getInt16(at, Endian.little).toDouble(),
      5123 => data.getUint16(at, Endian.little).toDouble(),
      _ => data.getUint32(at, Endian.little).toDouble(),
    };
  }

  /// Element [i] of an index accessor.
  int readIndex(int i) {
    final data = this.data;
    if (data == null) return 0;
    final at = offset + i * stride;
    return switch (componentType) {
      5121 => data.getUint8(at),
      5123 => data.getUint16(at, Endian.little),
      _ => data.getUint32(at, Endian.little),
    };
  }
}

/// The file name [uri] names beside the model, or null for a URI that is
/// not one: a `data:` URI, a scheme, an absolute path, a subdirectory or a
/// parent. Percent escapes are decoded, so `a%20b.bin` is `a b.bin`.
String? _siblingName(String uri) {
  if (uri.startsWith('data:')) return null;
  final String name;
  try {
    name = Uri.decodeComponent(uri);
  } on ArgumentError {
    return null;
  }
  if (name.isEmpty ||
      name == '.' ||
      name == '..' ||
      name.contains('/') ||
      name.contains(r'\') ||
      name.contains(':')) {
    return null;
  }
  return name;
}

/// A node's own transform: its `matrix`, or its translation, rotation and
/// scale composed as T * R * S. Column-major, as glTF writes it.
Float64List _localMatrix(Map<String, Object?> node) {
  final matrix = _numbers(node, 'matrix', 16);
  if (matrix != null) return matrix;
  final t = _numbers(node, 'translation', 3) ?? Float64List.fromList([0, 0, 0]);
  final r = _numbers(node, 'rotation', 4) ?? Float64List.fromList([0, 0, 0, 1]);
  final s = _numbers(node, 'scale', 3) ?? Float64List.fromList([1, 1, 1]);
  final (x, y, z, w) = (r[0], r[1], r[2], r[3]);
  return Float64List.fromList([
    (1 - 2 * (y * y + z * z)) * s[0],
    (2 * (x * y + z * w)) * s[0],
    (2 * (x * z - y * w)) * s[0],
    0,
    (2 * (x * y - z * w)) * s[1],
    (1 - 2 * (x * x + z * z)) * s[1],
    (2 * (y * z + x * w)) * s[1],
    0,
    (2 * (x * z + y * w)) * s[2],
    (2 * (y * z - x * w)) * s[2],
    (1 - 2 * (x * x + y * y)) * s[2],
    0,
    t[0],
    t[1],
    t[2],
    1,
  ]);
}

Float64List _identity() =>
    Float64List.fromList([1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1]);

/// [a] * [b], both column-major.
Float64List _multiply(Float64List a, Float64List b) {
  final out = Float64List(16);
  for (var col = 0; col < 4; col++) {
    for (var row = 0; row < 4; row++) {
      var sum = 0.0;
      for (var k = 0; k < 4; k++) {
        sum += a[k * 4 + row] * b[col * 4 + k];
      }
      out[col * 4 + row] = sum;
    }
  }
  return out;
}

/// The determinant of [m]'s upper 3x3: negative when it mirrors.
double _determinant(Float64List m) =>
    m[0] * (m[5] * m[10] - m[9] * m[6]) -
    m[4] * (m[1] * m[10] - m[9] * m[2]) +
    m[8] * (m[1] * m[6] - m[5] * m[2]);

Never _bad(String detail) =>
    throw QuarkModelParseException(QuarkModelParseIssue.malformed, detail);

Map<String, Object?> _object(Object? value, String what) =>
    value is Map<String, Object?> ? value : _bad('$what is not an object');

List<Object?> _array(Map<String, Object?> object, String key) =>
    switch (object[key]) {
      null => const [],
      final List<Object?> list => list,
      _ => _bad('$key is not an array'),
    };

int _int(Map<String, Object?> object, String key, [int? fallback]) =>
    switch (object[key]) {
      final int value => value,
      null when fallback != null => fallback,
      final value => _bad('$key is $value'),
    };

Float64List? _numbers(Map<String, Object?> object, String key, int length) {
  final list = _array(object, key);
  if (list.isEmpty) return null;
  if (list.length != length) _bad('$key has ${list.length} numbers');
  return Float64List.fromList([
    for (final value in list)
      value is num ? value.toDouble() : _bad('$key holds $value'),
  ]);
}

int _checkIndex(List<Object?> list, int index) =>
    index >= 0 && index < list.length
    ? index
    : _bad('$index of ${list.length}');

Object? _at(List<Object?> list, int index) => list[_checkIndex(list, index)];
