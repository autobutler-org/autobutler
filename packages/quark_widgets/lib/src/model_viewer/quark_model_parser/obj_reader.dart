import 'dart:typed_data';

import '../quark_model_mesh.dart';
import '../quark_model_parser.dart';
import 'mesh_builder.dart';
import 'text_cursor.dart';

/// Reads a Wavefront OBJ file: `v x y z` lines and `f` lines, whose corners
/// may be written `7`, `7/2`, `7//4` or `7/2/4` and may count back from the
/// latest vertex (`-1`). Every other line (normals, texture coordinates,
/// groups, materials, lines, points) is skipped.
QuarkModelMesh readObj(Uint8List bytes, QuarkModelLimits limits) {
  final cursor = TextCursor(bytes);
  final builder = MeshBuilder(limits);
  final corners = <int>[];
  while (!cursor.atEnd) {
    builder.tick();
    if (cursor.word('v')) {
      builder.add(
        cursor.readDouble(),
        cursor.readDouble(),
        cursor.readDouble(),
      );
    } else if (cursor.word('f')) {
      corners.clear();
      // A `#` starts a comment that runs to the end of the line.
      while (!cursor.atLineEnd && !cursor.at(0x23)) {
        final index = cursor.readInt();
        // OBJ counts from 1; a negative index counts back from the latest
        // vertex, and 0 names nothing.
        final resolved = index > 0 ? index - 1 : builder.vertexCount + index;
        if (index == 0 || resolved < 0) {
          throw QuarkModelParseException(
            QuarkModelParseIssue.malformed,
            'face index $index',
          );
        }
        corners.add(resolved);
        // The texture and normal indices after the slashes.
        cursor.skipRestOfWord();
      }
      builder.polygon(corners);
    }
    cursor.nextLine();
  }
  return builder.build(isZUp: false);
}
