import 'package:quark_widgets/quark_widgets.dart' show QuarkModelLimits;

/// Tuning values for the 3D model viewer (#2988).
class ModelViewerConfig {
  ModelViewerConfig._();

  /// The largest model file the viewer downloads. A bigger one is never
  /// read; the viewer falls back to the file's details.
  static const int maxBytes = 64 * 1024 * 1024;

  /// The most triangles a model may have. Every frame projects, sorts and
  /// repaints them all, and a million took a few hundred milliseconds a
  /// frame in a debug-mode benchmark, so this is what keeps orbiting usable.
  static const int maxTriangles = 500000;

  /// The longest reading a file may take before the viewer gives up.
  static const Duration maxParseTime = Duration(seconds: 10);

  /// [maxBytes], [maxTriangles] and [maxParseTime] as the parser takes them.
  static const QuarkModelLimits limits = QuarkModelLimits(
    maxBytes: maxBytes,
    maxTriangles: maxTriangles,
    maxDuration: maxParseTime,
  );
}
