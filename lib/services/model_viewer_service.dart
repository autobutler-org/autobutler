import 'package:flutter/foundation.dart';
import 'package:quark_widgets/quark_widgets.dart';

/// Reads 3D model files for the model viewer (#2988) off the UI isolate.
///
/// The bytes come through `CodeViewerService.readRange`, which reads at most
/// the length asked for; this only parses them.
class ModelViewerService {
  ModelViewerService._();

  /// Parses [bytes] as a [format] file within [limits] on a background
  /// isolate (inline on the web, where the time cap bounds the pause).
  /// Throws the parser's `QuarkModelParseException` for a file it will not
  /// draw.
  static Future<QuarkModelMesh> parse(
    Uint8List bytes,
    QuarkModelFormat format,
    QuarkModelLimits limits,
  ) => compute(_parse, (bytes: bytes, format: format, limits: limits));

  static QuarkModelMesh _parse(
    ({Uint8List bytes, QuarkModelFormat format, QuarkModelLimits limits}) job,
  ) => parseQuarkModel(job.bytes, job.format, limits: job.limits);
}
