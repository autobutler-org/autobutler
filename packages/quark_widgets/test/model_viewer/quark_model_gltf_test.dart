import 'dart:convert';
import 'dart:typed_data';

import 'package:flutter_test/flutter_test.dart';
import 'package:quark_widgets/quark_widgets.dart';

Uint8List floats(List<double> values) =>
    Float32List.fromList(values).buffer.asUint8List();

Uint8List shorts(List<int> values) =>
    Uint16List.fromList(values).buffer.asUint8List();

/// The unit square in the XY plane: four float corners (48 bytes), then the
/// two triangles' six ushort indices (12 bytes).
final Uint8List squareBin = Uint8List.fromList([
  ...floats([0, 0, 0, 1, 0, 0, 1, 1, 0, 0, 1, 0]),
  ...shorts([0, 1, 2, 0, 2, 3]),
]);

/// A glTF document drawing [squareBin] from a buffer at [uri] (the GLB's
/// binary chunk when null), through [nodes] (node 0 is the scene's root),
/// with [primitive] in place of the indexed triangle list.
Map<String, Object?> squareGltf({
  String? uri,
  List<Map<String, Object?>>? nodes,
  Map<String, Object?>? primitive,
  Map<String, Object?>? extra,
}) => {
  'asset': {'version': '2.0'},
  'buffers': [
    {'byteLength': 60, 'uri': ?uri},
  ],
  'bufferViews': [
    {'buffer': 0, 'byteLength': 48},
    {'buffer': 0, 'byteOffset': 48, 'byteLength': 12},
  ],
  'accessors': [
    {'bufferView': 0, 'componentType': 5126, 'count': 4, 'type': 'VEC3'},
    {'bufferView': 1, 'componentType': 5123, 'count': 6, 'type': 'SCALAR'},
  ],
  'meshes': [
    {
      'primitives': [
        primitive ??
            {
              'attributes': {'POSITION': 0},
              'indices': 1,
            },
      ],
    },
  ],
  'nodes':
      nodes ??
      [
        {'mesh': 0},
      ],
  'scenes': [
    {
      'nodes': [0],
    },
  ],
  ...?extra,
};

/// [document] as a GLB, with [bin] as its binary chunk.
Uint8List glb(Map<String, Object?> document, [Uint8List? bin]) {
  final text = utf8.encode(jsonEncode(document));
  final jsonLength = (text.length + 3) & ~3;
  final binLength = bin == null ? 0 : (bin.length + 3) & ~3;
  final total = 12 + 8 + jsonLength + (bin == null ? 0 : 8 + binLength);
  final out = Uint8List(total);
  final data = ByteData.sublistView(out)
    ..setUint32(0, 0x46546C67, Endian.little)
    ..setUint32(4, 2, Endian.little)
    ..setUint32(8, total, Endian.little)
    ..setUint32(12, jsonLength, Endian.little)
    ..setUint32(16, 0x4E4F534A, Endian.little);
  out.setAll(20, text);
  out.fillRange(20 + text.length, 20 + jsonLength, 0x20);
  if (bin != null) {
    final at = 20 + jsonLength;
    data
      ..setUint32(at, binLength, Endian.little)
      ..setUint32(at + 4, 0x004E4942, Endian.little);
    out.setAll(at + 8, bin);
  }
  return out;
}

/// [document] as a `.gltf` file.
Uint8List gltf(Map<String, Object?> document) =>
    utf8.encode(jsonEncode(document));

String dataUri(Uint8List bytes) =>
    'data:application/octet-stream;base64,${base64.encode(bytes)}';

QuarkModelParseIssue? issueOf(
  Uint8List bytes, {
  QuarkModelLimits limits = const QuarkModelLimits(),
  Map<String, Uint8List> resources = const {},
}) {
  try {
    parseQuarkModel(
      bytes,
      QuarkModelFormat.gltf,
      limits: limits,
      resources: resources,
    );
    return null;
  } on QuarkModelParseException catch (e) {
    return e.issue;
  }
}

QuarkModelMesh parse(
  Uint8List bytes, [
  Map<String, Uint8List> resources = const {},
]) => parseQuarkModel(bytes, QuarkModelFormat.gltf, resources: resources);

/// `parseQuarkModel` for glTF (#3225): GLB and JSON with each kind of
/// buffer, node transforms, every triangle mode, and the files it refuses.
void main() {
  test('GLB: an indexed square from the binary chunk', () {
    final mesh = parse(glb(squareGltf(), squareBin));
    expect(mesh.triangleCount, 2);
    expect(mesh.vertexCount, 4);
    expect(mesh.isZUp, isFalse);
    expect(mesh.indices, [0, 1, 2, 0, 2, 3]);
    expect(mesh.size, (x: 1.0, y: 1.0, z: 0.0));
    expect(mesh.normals.sublist(0, 3), [0, 0, 1]);
  });

  test('JSON: a buffer embedded as a data URI', () {
    final mesh = parse(gltf(squareGltf(uri: dataUri(squareBin))));
    expect(mesh.triangleCount, 2);
  });

  test('JSON: a sibling .bin, named by its decoded file name', () {
    final bytes = gltf(squareGltf(uri: 'square%20mesh.bin'));
    expect(quarkModelResources(bytes, QuarkModelFormat.gltf), [
      'square mesh.bin',
    ]);
    final mesh = parse(bytes, {'square mesh.bin': squareBin});
    expect(mesh.triangleCount, 2);
    // Not handed the sibling, the parser has nothing to read.
    expect(issueOf(bytes), QuarkModelParseIssue.malformed);
  });

  test('a buffer outside the model\'s folder is never asked for', () {
    for (final uri in [
      'https://example.com/a.bin',
      'file:///etc/passwd',
      '/abs/a.bin',
      '../a.bin',
      'sub/a.bin',
      'sub%2Fa.bin',
      '..',
    ]) {
      final bytes = gltf(squareGltf(uri: uri));
      expect(quarkModelResources(bytes, QuarkModelFormat.gltf), isEmpty);
      expect(
        issueOf(bytes, resources: {uri: squareBin}),
        QuarkModelParseIssue.malformed,
        reason: uri,
      );
    }
  });

  test('other formats need no resources', () {
    expect(quarkModelResources(Uint8List(10), QuarkModelFormat.stl), isEmpty);
  });

  test('node transforms are flattened, parent then child', () {
    final mesh = parse(
      glb(
        squareGltf(
          nodes: [
            {
              'scale': [2, 2, 2],
              'children': [1],
            },
            {
              'mesh': 0,
              'translation': [10, 0, 0],
              // A quarter turn about X stands the square up in XZ.
              'rotation': [0.7071067811865476, 0, 0, 0.7071067811865476],
            },
          ],
        ),
        squareBin,
      ),
    );
    expect(mesh.min[0], closeTo(20, 1e-5));
    expect(mesh.max[0], closeTo(22, 1e-5));
    expect(mesh.size.y, closeTo(0, 1e-5));
    expect(mesh.size.z, closeTo(2, 1e-5));
  });

  test('a matrix places the mesh, and a mirror keeps faces outward', () {
    final mesh = parse(
      glb(
        squareGltf(
          nodes: [
            {
              'mesh': 0,
              'matrix': [-1, 0, 0, 0, 0, 1, 0, 0, 0, 0, 1, 0, 5, 0, 0, 1],
            },
          ],
        ),
        squareBin,
      ),
    );
    expect(mesh.min[0], 4);
    expect(mesh.max[0], 5);
    // Mirrored in X, the square still faces +Z.
    expect(mesh.normals.sublist(0, 3), [0, 0, 1]);
  });

  test('a mesh used by two nodes is drawn twice', () {
    final mesh = parse(
      glb(
        squareGltf(
          nodes: [
            {
              'mesh': 0,
              'children': [1],
            },
            {
              'mesh': 0,
              'translation': [0, 0, 3],
            },
          ],
        ),
        squareBin,
      ),
    );
    expect(mesh.triangleCount, 4);
    expect(mesh.size.z, 3);
  });

  test('non-indexed lists, strips and fans', () {
    for (final (mode, triangles) in [(4, 1), (5, 2), (6, 2)]) {
      final mesh = parse(
        glb(
          squareGltf(
            primitive: {
              'attributes': {'POSITION': 0},
              'mode': mode,
            },
          ),
          squareBin,
        ),
      );
      expect(mesh.triangleCount, triangles, reason: 'mode $mode');
    }
  });

  test('normalized ushort positions are scaled to 0..1', () {
    final bin = Uint8List.fromList([
      ...shorts([0, 0, 0, 65535, 0, 0, 0, 65535, 0]),
      0, 0, // padding to four bytes
    ]);
    final mesh = parse(
      glb({
        'asset': {'version': '2.0'},
        'buffers': [
          {'byteLength': bin.length},
        ],
        'bufferViews': [
          {'buffer': 0, 'byteLength': 18},
        ],
        'accessors': [
          {
            'bufferView': 0,
            'componentType': 5123,
            'normalized': true,
            'count': 3,
            'type': 'VEC3',
          },
        ],
        'meshes': [
          {
            'primitives': [
              {
                'attributes': {'POSITION': 0},
              },
            ],
          },
        ],
      }, bin),
    );
    expect(mesh.size, (x: 1.0, y: 1.0, z: 0.0));
  });

  test('points and lines have no surfaces', () {
    expect(
      issueOf(
        glb(
          squareGltf(
            primitive: {
              'attributes': {'POSITION': 0},
              'mode': 0,
            },
          ),
          squareBin,
        ),
      ),
      QuarkModelParseIssue.noSurfaces,
    );
  });

  test('the caps hold, sibling buffers included', () {
    expect(
      issueOf(
        glb(squareGltf(), squareBin),
        limits: const QuarkModelLimits(maxTriangles: 1),
      ),
      QuarkModelParseIssue.tooManyTriangles,
    );
    final bytes = gltf(squareGltf(uri: 'a.bin'));
    expect(
      issueOf(
        bytes,
        limits: QuarkModelLimits(maxBytes: bytes.length + 59),
        resources: {'a.bin': squareBin},
      ),
      QuarkModelParseIssue.tooLarge,
    );
  });

  test('broken files are malformed, never a crash', () {
    final cases = <String, Uint8List>{
      'not JSON': utf8.encode('{"asset": '),
      'glTF 1.0': gltf({
        'asset': {'version': '1.0'},
      }),
      'truncated GLB': Uint8List.sublistView(
        glb(squareGltf(), squareBin),
        0,
        30,
      ),
      'GLB without a binary chunk': glb(squareGltf()),
      'index past the vertices': glb(
        squareGltf(),
        Uint8List.fromList([
          ...squareBin.sublist(0, 48),
          ...shorts([0, 1, 9, 0, 2, 3]),
        ]),
      ),
      'accessor past its view': glb(
        squareGltf(
          extra: {
            'accessors': [
              {
                'bufferView': 0,
                'componentType': 5126,
                'count': 5,
                'type': 'VEC3',
              },
            ],
            'meshes': [
              {
                'primitives': [
                  {
                    'attributes': {'POSITION': 0},
                  },
                ],
              },
            ],
          },
        ),
        squareBin,
      ),
      'a node cycle': glb(
        squareGltf(
          nodes: [
            {
              'mesh': 0,
              'children': [1],
            },
            {
              'children': [0],
            },
          ],
        ),
        squareBin,
      ),
      'Draco': glb(
        squareGltf(
          extra: {
            'extensionsRequired': ['KHR_draco_mesh_compression'],
          },
        ),
        squareBin,
      ),
      'a string where a number goes': glb(
        squareGltf(
          nodes: [
            {'mesh': '0'},
          ],
        ),
        squareBin,
      ),
    };
    for (final MapEntry(:key, :value) in cases.entries) {
      expect(issueOf(value), QuarkModelParseIssue.malformed, reason: key);
    }
  });
}
