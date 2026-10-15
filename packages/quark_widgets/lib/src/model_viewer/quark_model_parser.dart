import 'dart:typed_data';

import 'quark_model_mesh.dart';
import 'quark_model_parser/gltf_reader.dart';
import 'quark_model_parser/obj_reader.dart';
import 'quark_model_parser/ply_reader.dart';
import 'quark_model_parser/stl_reader.dart';

/// Why `parseQuarkModel` gave up on a file.
enum QuarkModelParseIssue {
  /// The file is larger than `QuarkModelLimits.maxBytes`.
  tooLarge,

  /// The file has more triangles, or more vertices, than the limits allow.
  tooManyTriangles,

  /// Reading took longer than `QuarkModelLimits.maxDuration`.
  tooSlow,

  /// The bytes are not a valid file of the format they were read as.
  malformed,

  /// The file is valid but has no triangles: a point cloud, or an empty
  /// solid.
  noSurfaces,
}

/// Thrown by `parseQuarkModel` when it cannot or will not read a file.
class QuarkModelParseException implements Exception {
  /// Creates the exception for [issue], with [detail] for logs.
  const QuarkModelParseException(this.issue, [this.detail = '']);

  /// What went wrong, for the caller to describe.
  final QuarkModelParseIssue issue;

  /// A diagnostic line for logs; never shown to a user.
  final String detail;

  @override
  String toString() => 'QuarkModelParseException(${issue.name}: $detail)';
}

/// The hard caps `parseQuarkModel` holds a file to. A file over any of them
/// throws a `QuarkModelParseException` rather than costing unbounded memory
/// or time.
class QuarkModelLimits {
  /// Creates limits; the defaults suit a phone.
  const QuarkModelLimits({
    this.maxBytes = 64 * 1024 * 1024,
    this.maxTriangles = 1000000,
    this.maxDuration = const Duration(seconds: 10),
  });

  /// The largest file read, in bytes.
  final int maxBytes;

  /// The most triangles, after polygons are split, a mesh may have. A mesh
  /// may have at most three times as many vertices.
  final int maxTriangles;

  /// The longest reading may take.
  final Duration maxDuration;
}

/// The files beside a [format] file that [parseQuarkModel] needs as its
/// `resources`, by file name: the `.bin` buffers a `.gltf` names. Empty for
/// every other format, and for a glTF whose buffers are all embedded.
///
/// Only plain file names in the same folder are listed. A buffer URI with a
/// scheme, a path or a parent is never fetched, so a model cannot make its
/// viewer read anything but its own siblings. Throws a
/// [QuarkModelParseException] when [bytes] are not a glTF document.
List<String> quarkModelResources(Uint8List bytes, QuarkModelFormat format) =>
    format == QuarkModelFormat.gltf ? gltfSiblingFiles(bytes) : const [];

/// Reads [bytes] as a [format] file into a mesh, within [limits].
///
/// [resources] are the sibling files a glTF's buffers name, keyed by the
/// names [quarkModelResources] lists; together with [bytes] they are held to
/// `QuarkModelLimits.maxBytes`.
///
/// STL is told apart as ASCII or binary from its contents, and glTF as JSON
/// or GLB; PLY's header says which encoding it uses. Polygons are split into triangle fans, and STL's
/// unshared corners are merged where they are identical, so the vertex count
/// means what it does in a modeling tool.
///
/// Pure Dart and synchronous: run it with `compute` to keep it off the UI
/// isolate. Throws a [QuarkModelParseException] for anything it cannot draw.
QuarkModelMesh parseQuarkModel(
  Uint8List bytes,
  QuarkModelFormat format, {
  QuarkModelLimits limits = const QuarkModelLimits(),
  Map<String, Uint8List> resources = const {},
}) {
  final total = resources.values.fold(bytes.length, (sum, r) => sum + r.length);
  if (total > limits.maxBytes) {
    throw QuarkModelParseException(
      QuarkModelParseIssue.tooLarge,
      '$total bytes',
    );
  }
  try {
    return switch (format) {
      QuarkModelFormat.stl => readStl(bytes, limits),
      QuarkModelFormat.obj => readObj(bytes, limits),
      QuarkModelFormat.ply => readPly(bytes, limits),
      QuarkModelFormat.gltf => readGltf(bytes, resources, limits),
    };
  } on RangeError catch (e) {
    // A count or offset that runs past the end of the bytes.
    throw QuarkModelParseException(QuarkModelParseIssue.malformed, '$e');
  }
}
