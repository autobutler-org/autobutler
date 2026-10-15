import 'dart:convert';
import 'dart:typed_data';

import '../quark_model_mesh.dart';
import '../quark_model_parser.dart';
import 'mesh_builder.dart';
import 'text_cursor.dart';
import 'zip_reader.dart';

/// Where a package keeps its model when `_rels/.rels` does not say.
const String _defaultModel = '3D/3dmodel.model';

/// The relationship type that names a package's model part ends with this.
const String _modelRelationship = '/3dmodel';

/// How deeply components may nest before the file is taken as a loop.
const int _maxDepth = 32;

/// Reads a 3MF package: a zip whose model part is XML listing each
/// object's `<vertices>` and `<triangles>`, objects built from other
/// objects' `<components>`, and the `<build>` items that place them.
///
/// Every build item is drawn, with its transform and its components'
/// transforms applied, as one mesh. Parts the production extension names
/// with `p:path` are read from the same archive; nothing outside it is ever
/// fetched. Colors, materials, textures and slicer settings are ignored.
///
/// Only the parts the model reaches are decompressed, and together they are
/// held to `QuarkModelLimits.maxBytes`, so a zip bomb stops at the cap. A
/// DTD is refused, so no entity is ever expanded.
QuarkModelMesh readThreeMf(Uint8List bytes, QuarkModelLimits limits) {
  final reader = _ThreeMfReader(ZipReader(bytes), MeshBuilder(limits));
  final root = reader.part(reader.startPart());
  for (final item in root.items) {
    reader.place(root, item, _identity(), 0);
  }
  return reader.builder.build(isZUp: true);
}

/// One reference to an object: a build item or a component.
class _Reference {
  _Reference(this.objectId, this.transform, this.path);

  final int objectId;
  final Float64List transform;

  /// The part the object is in, or null for the referring part itself.
  final String? path;
}

/// An object as read: its own mesh, its components, or both.
class _Object {
  Float32List positions = Float32List(0);
  int vertexCount = 0;
  Uint32List indices = Uint32List(0);
  int indexLength = 0;
  final List<_Reference> components = [];

  void vertex(double x, double y, double z) {
    if (vertexCount * 3 + 3 > positions.length) {
      positions = Float32List(positions.length * 2 + 96)..setAll(0, positions);
    }
    positions[vertexCount * 3] = x;
    positions[vertexCount * 3 + 1] = y;
    positions[vertexCount * 3 + 2] = z;
    vertexCount++;
  }

  void triangle(int a, int b, int c) {
    if (indexLength + 3 > indices.length) {
      indices = Uint32List(indices.length * 2 + 96)..setAll(0, indices);
    }
    indices[indexLength++] = a;
    indices[indexLength++] = b;
    indices[indexLength++] = c;
  }
}

/// One model part: its objects by id, and its build items.
class _Part {
  final Map<int, _Object> objects = {};
  final List<_Reference> items = [];
}

class _ThreeMfReader {
  _ThreeMfReader(this.zip, this.builder);

  final ZipReader zip;
  final MeshBuilder builder;
  final Map<String, _Part> _parts = {};
  int _inflated = 0;
  int _vertices = 0;
  int _triangles = 0;

  QuarkModelLimits get limits => builder.limits;

  /// The model part's name, from the package relationships.
  String startPart() {
    final rels = _read('_rels/.rels');
    if (rels == null) return _defaultModel;
    final xml = _XmlScanner(rels, builder);
    while (xml.next()) {
      if (xml.isEnd || !xml.named('Relationship')) continue;
      if (xml.string('TargetMode') == 'External') continue;
      final type = xml.string('Type') ?? '';
      final target = xml.string('Target');
      if (type.endsWith(_modelRelationship) && target != null) {
        return _partName(target);
      }
    }
    return _defaultModel;
  }

  /// The part called [name], read the first time it is asked for.
  _Part part(String name) {
    final key = name.toLowerCase();
    final cached = _parts[key];
    if (cached != null) return cached;
    final bytes = _read(name) ?? _bad('no part $name');
    return _parts[key] = _parse(bytes);
  }

  /// Draws [reference]'s object from [from] with [parent] applied.
  void place(_Part from, _Reference reference, Float64List parent, int depth) {
    builder.tick();
    if (depth > _maxDepth) _bad('components nest past $_maxDepth');
    final path = reference.path;
    final part = path == null ? from : this.part(_partName(path));
    final object =
        part.objects[reference.objectId] ??
        _bad('no object ${reference.objectId}');
    final m = _compose(parent, reference.transform);
    if (object.indexLength > 0) _mesh(object, m);
    for (final component in object.components) {
      place(part, component, m, depth + 1);
    }
  }

  void _mesh(_Object object, Float64List m) {
    final base = builder.vertexCount;
    final p = object.positions;
    for (var v = 0; v < object.vertexCount; v++) {
      builder.tick();
      final (x, y, z) = (p[v * 3], p[v * 3 + 1], p[v * 3 + 2]);
      builder.add(
        m[0] * x + m[3] * y + m[6] * z + m[9],
        m[1] * x + m[4] * y + m[7] * z + m[10],
        m[2] * x + m[5] * y + m[8] * z + m[11],
      );
    }
    // A mirroring transform turns every triangle inside out; swapping two
    // corners keeps the normals pointing outward.
    final flip = _determinant(m) < 0;
    final indices = object.indices;
    for (var i = 0; i < object.indexLength; i += 3) {
      builder.tick();
      final a = base + indices[i];
      final b = base + indices[i + 1];
      final c = base + indices[i + 2];
      flip ? builder.triangle(a, c, b) : builder.triangle(a, b, c);
    }
  }

  _Part _parse(Uint8List bytes) {
    final part = _Part();
    final xml = _XmlScanner(bytes, builder);
    if (!xml.next() || xml.isEnd || !xml.named('model')) _bad('no <model>');
    _Object? object;
    var inBuild = false;
    while (xml.next()) {
      if (xml.isEnd) {
        if (xml.named('object')) {
          _check(object);
          object = null;
        } else if (xml.named('build')) {
          inBuild = false;
        }
        continue;
      }
      if (object != null) {
        if (xml.named('vertex')) {
          if (++_vertices > builder.maxVertices) _tooMany('vertices');
          object.vertex(xml.number('x'), xml.number('y'), xml.number('z'));
        } else if (xml.named('triangle')) {
          if (++_triangles > limits.maxTriangles) _tooMany('triangles');
          object.triangle(xml.index('v1'), xml.index('v2'), xml.index('v3'));
        } else if (xml.named('component')) {
          object.components.add(_reference(xml));
        }
      } else if (xml.named('object')) {
        final id = xml.index('id');
        if (part.objects.containsKey(id)) _bad('object $id twice');
        object = part.objects[id] = _Object();
        if (xml.isEmpty) object = null;
      } else if (xml.named('build')) {
        inBuild = !xml.isEmpty;
      } else if (inBuild && xml.named('item')) {
        part.items.add(_reference(xml));
      }
    }
    return part;
  }

  /// Throws when one of [object]'s triangles names a vertex it lacks.
  void _check(_Object? object) {
    if (object == null) return;
    for (var i = 0; i < object.indexLength; i++) {
      if (object.indices[i] >= object.vertexCount) {
        _bad('vertex ${object.indices[i]} of ${object.vertexCount}');
      }
    }
  }

  _Reference _reference(_XmlScanner xml) => _Reference(
    xml.index('objectid'),
    xml.transform('transform'),
    xml.string('path'),
  );

  /// The entry called [name], decompressed within what the size cap has
  /// left, or null when the archive has none.
  Uint8List? _read(String name) {
    final bytes = zip.read(
      name,
      maxLength: limits.maxBytes - _inflated,
      tick: builder.tick,
    );
    _inflated += bytes?.length ?? 0;
    return bytes;
  }

  Never _tooMany(String what) => throw QuarkModelParseException(
    QuarkModelParseIssue.tooManyTriangles,
    'too many $what',
  );
}

/// An OPC part name as a zip entry name: percent escapes decoded and the
/// leading slash dropped.
String _partName(String target) {
  var name = target;
  try {
    name = Uri.decodeFull(target);
  } on ArgumentError {
    // Not escaped after all; use it as written.
  }
  return name.startsWith('/') ? name.substring(1) : name;
}

/// Walks the start and end tags of an XML document straight from its bytes,
/// so a model of a million vertices costs no string per number. Text,
/// comments, processing instructions and CDATA are skipped; a DTD is
/// refused. Names compare by their local part, without the namespace
/// prefix.
class _XmlScanner {
  _XmlScanner(this.bytes, this.builder) : _cursor = TextCursor(bytes);

  final Uint8List bytes;
  final MeshBuilder builder;
  final TextCursor _cursor;
  int _at = 0;

  /// Whether the tag is an end tag.
  bool isEnd = false;

  /// Whether the start tag closes itself (`<a/>`).
  bool isEmpty = false;

  int _nameStart = 0;
  int _nameEnd = 0;

  /// Each attribute as four offsets: local name start and end, value start
  /// and end.
  final List<int> _attributes = [];

  static const int _lt = 0x3c, _gt = 0x3e, _slash = 0x2f, _eq = 0x3d;
  static const int _quote = 0x22, _apostrophe = 0x27, _colon = 0x3a;

  /// Moves to the next tag. False at the end of the document.
  bool next() {
    while (true) {
      builder.tick();
      while (_at < bytes.length && bytes[_at] != _lt) {
        _at++;
      }
      if (_at >= bytes.length) return false;
      _at++;
      if (_startsWith('?')) {
        _skipPast('?>');
      } else if (_startsWith('!--')) {
        _skipPast('-->');
      } else if (_startsWith('![CDATA[')) {
        _skipPast(']]>');
      } else if (_startsWith('!')) {
        _bad('DTDs are not allowed');
      } else {
        _tag();
        return true;
      }
    }
  }

  void _tag() {
    isEnd = _at < bytes.length && bytes[_at] == _slash;
    if (isEnd) _at++;
    isEmpty = false;
    final (nameStart, nameEnd) = _name();
    _nameStart = nameStart;
    _nameEnd = nameEnd;
    _attributes.clear();
    while (true) {
      _skipWhitespace();
      if (_at >= bytes.length) _bad('unclosed tag');
      final byte = bytes[_at];
      if (byte == _gt) {
        _at++;
        return;
      }
      if (byte == _slash) {
        if (_at + 1 >= bytes.length || bytes[_at + 1] != _gt) _bad('stray /');
        isEmpty = true;
        _at += 2;
        return;
      }
      final (start, end) = _name();
      _skipWhitespace();
      if (_at >= bytes.length || bytes[_at] != _eq) _bad('attribute without =');
      _at++;
      _skipWhitespace();
      if (_at >= bytes.length) _bad('unclosed tag');
      final quote = bytes[_at];
      if (quote != _quote && quote != _apostrophe) _bad('unquoted attribute');
      final valueStart = ++_at;
      while (_at < bytes.length && bytes[_at] != quote) {
        _at++;
      }
      if (_at >= bytes.length) _bad('unclosed attribute');
      _attributes.addAll([start, end, valueStart, _at++]);
    }
  }

  /// Reads a name at the cursor and returns its local part's offsets.
  (int, int) _name() {
    var start = _at;
    while (_at < bytes.length) {
      final byte = bytes[_at];
      if (byte <= 0x20 || byte == _gt || byte == _slash || byte == _eq) break;
      if (byte == _colon) start = _at + 1;
      _at++;
    }
    if (start == _at) _bad('empty name at $_at');
    return (start, _at);
  }

  /// Whether the tag's local name is [name].
  bool named(String name) => _equals(_nameStart, _nameEnd, name);

  /// The index into `_attributes` of the attribute called [name], or -1.
  int _find(String name) {
    for (var i = 0; i < _attributes.length; i += 4) {
      if (_equals(_attributes[i], _attributes[i + 1], name)) return i;
    }
    return -1;
  }

  /// The attribute [name] as a number; missing or not a number is
  /// malformed.
  double number(String name) {
    final at = _find(name);
    if (at < 0) _bad('no $name');
    _cursor.position = _attributes[at + 2];
    final value = _cursor.readDouble();
    _end(_attributes[at + 3], name);
    return value;
  }

  /// The attribute [name] as a non-negative whole number.
  int index(String name) {
    final at = _find(name);
    if (at < 0) _bad('no $name');
    _cursor.position = _attributes[at + 2];
    final value = _cursor.readInt();
    _end(_attributes[at + 3], name);
    if (value < 0) _bad('$name is $value');
    return value;
  }

  /// The attribute [name] as twelve numbers, a 3MF transform, or the
  /// identity when it is missing.
  Float64List transform(String name) {
    final at = _find(name);
    if (at < 0) return _identity();
    _cursor.position = _attributes[at + 2];
    final m = Float64List(12);
    for (var i = 0; i < 12; i++) {
      _cursor.skipWhitespace();
      m[i] = _cursor.readDouble();
    }
    _cursor.skipWhitespace();
    _end(_attributes[at + 3], name);
    return m;
  }

  /// The attribute [name] as text, with entity references decoded, or
  /// null when it is missing.
  String? string(String name) {
    final at = _find(name);
    if (at < 0) return null;
    final raw = utf8.decode(
      Uint8List.sublistView(bytes, _attributes[at + 2], _attributes[at + 3]),
      allowMalformed: true,
    );
    return raw.replaceAllMapped(RegExp('&(#x[0-9a-fA-F]+|#[0-9]+|[a-z]+);'), (
      match,
    ) {
      final entity = match[1]!;
      if (entity.startsWith('#x')) {
        return String.fromCharCode(int.parse(entity.substring(2), radix: 16));
      }
      if (entity.startsWith('#')) {
        return String.fromCharCode(int.parse(entity.substring(1)));
      }
      return switch (entity) {
        'amp' => '&',
        'lt' => '<',
        'gt' => '>',
        'quot' => '"',
        'apos' => "'",
        _ => _bad('entity &$entity;'),
      };
    });
  }

  void _end(int end, String name) {
    _cursor.skipSpaces();
    if (_cursor.position != end) _bad('$name is not a number');
  }

  bool _equals(int start, int end, String name) {
    if (end - start != name.length) return false;
    for (var i = 0; i < name.length; i++) {
      if (bytes[start + i] != name.codeUnitAt(i)) return false;
    }
    return true;
  }

  bool _startsWith(String text) => _equals(
    _at,
    _at + text.length > bytes.length ? -1 : _at + text.length,
    text,
  );

  void _skipPast(String terminator) {
    while (_at < bytes.length) {
      if (_startsWith(terminator)) {
        _at += terminator.length;
        return;
      }
      _at++;
    }
    _bad('unclosed $terminator');
  }

  void _skipWhitespace() {
    while (_at < bytes.length && bytes[_at] <= 0x20) {
      _at++;
    }
  }
}

Float64List _identity() =>
    Float64List.fromList([1, 0, 0, 0, 1, 0, 0, 0, 1, 0, 0, 0]);

/// [outer] after [inner], both 3MF transforms: the 3x3 by columns, then the
/// translation.
Float64List _compose(Float64List outer, Float64List inner) {
  final out = Float64List(12);
  for (var column = 0; column < 4; column++) {
    final x = inner[column * 3];
    final y = inner[column * 3 + 1];
    final z = inner[column * 3 + 2];
    for (var k = 0; k < 3; k++) {
      out[column * 3 + k] =
          outer[k] * x +
          outer[3 + k] * y +
          outer[6 + k] * z +
          (column == 3 ? outer[9 + k] : 0);
    }
  }
  return out;
}

/// The determinant of [m]'s 3x3: negative when it mirrors.
double _determinant(Float64List m) =>
    m[0] * (m[4] * m[8] - m[7] * m[5]) -
    m[3] * (m[1] * m[8] - m[7] * m[2]) +
    m[6] * (m[1] * m[5] - m[4] * m[2]);

Never _bad(String detail) => throw QuarkModelParseException(
  QuarkModelParseIssue.malformed,
  '3mf: $detail',
);
