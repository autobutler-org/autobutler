import 'dart:typed_data';

import '../quark_model_mesh.dart';
import '../quark_model_parser.dart';
import 'mesh_builder.dart';
import 'text_cursor.dart';

const int _headerBytes = 84;
const int _facetBytes = 50;

/// Reads an STL file, ASCII or binary.
///
/// A file whose length is exactly what its binary triangle count says is
/// binary, even when its header starts with `solid` as some exporters write.
/// Otherwise a file starting with `solid` is ASCII, and anything else is read
/// as binary.
QuarkModelMesh readStl(Uint8List bytes, QuarkModelLimits limits) {
  final count = bytes.length >= _headerBytes
      ? ByteData.sublistView(bytes).getUint32(80, Endian.little)
      : -1;
  final exactBinary =
      count >= 0 && _headerBytes + count * _facetBytes == bytes.length;
  final start = TextCursor(bytes)..skipWhitespace();
  if (!exactBinary && start.word('solid')) {
    return _readAscii(bytes, limits);
  }
  if (count < 0 || _headerBytes + count * _facetBytes > bytes.length) {
    throw QuarkModelParseException(
      QuarkModelParseIssue.malformed,
      'binary STL of ${bytes.length} bytes claims $count triangles',
    );
  }
  return _readBinary(bytes, count, limits);
}

QuarkModelMesh _readBinary(
  Uint8List bytes,
  int count,
  QuarkModelLimits limits,
) {
  if (count > limits.maxTriangles) {
    throw QuarkModelParseException(
      QuarkModelParseIssue.tooManyTriangles,
      '$count triangles',
    );
  }
  final data = ByteData.sublistView(bytes);
  final builder = MeshBuilder(limits);
  final corners = List<int>.filled(3, 0);
  for (var t = 0; t < count; t++) {
    builder.tick();
    // 12 bytes of normal, recomputed from the corners instead, then three
    // corners of 12 bytes and a 2-byte attribute.
    var at = _headerBytes + t * _facetBytes + 12;
    for (var c = 0; c < 3; c++, at += 12) {
      corners[c] = builder.weld(
        data.getFloat32(at, Endian.little),
        data.getFloat32(at + 4, Endian.little),
        data.getFloat32(at + 8, Endian.little),
      );
    }
    builder.triangle(corners[0], corners[1], corners[2]);
  }
  return builder.build(isZUp: true);
}

/// `solid name`, then `facet normal … outer loop vertex x y z … endloop
/// endfacet` per triangle, then `endsolid`. Only the `vertex` lines matter:
/// a loop's corners become a fan, and every other word is skipped.
QuarkModelMesh _readAscii(Uint8List bytes, QuarkModelLimits limits) {
  final cursor = TextCursor(bytes);
  final builder = MeshBuilder(limits);
  final corners = <int>[];
  while (true) {
    cursor.skipWhitespace();
    if (cursor.atEnd) break;
    builder.tick();
    if (cursor.word('vertex')) {
      corners.add(
        builder.weld(
          cursor.readDouble(),
          cursor.readDouble(),
          cursor.readDouble(),
        ),
      );
    } else if (cursor.word('endloop')) {
      builder.polygon(corners);
      corners.clear();
    } else {
      cursor.skipWord();
    }
  }
  return builder.build(isZUp: true);
}
