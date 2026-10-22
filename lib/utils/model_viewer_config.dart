import 'package:quark_widgets/quark_widgets.dart' show QuarkModelLimits;

/// Tuning values for the 3D model viewer (#2988).
class ModelViewerConfig {
  ModelViewerConfig._();

  /// The largest model file the viewer downloads. A bigger one is never
  /// read; the viewer falls back to the file's details.
  static const int maxBytes = 64 * 1024 * 1024;

  /// The most triangles a model may have (#3231). Drawing one frame costs
  /// about a quarter of a microsecond a triangle in a debug-mode benchmark,
  /// so a model at the cap takes about half a second to redraw in full;
  /// the viewer draws its [previewTriangles] preview while it moves and
  /// the full model only once it rests. Memory is the other bound: the
  /// mesh and the renderer's shaded buffers take about 85 bytes a
  /// triangle, some 170 MB here, on top of the file itself, and the
  /// wireframe's edge buffer adds 48 more.
  static const int maxTriangles = 2000000;

  /// The most triangles drawn while a model moves. A model with more gets
  /// a coarser preview of at most this many, which redraws in about 40 ms
  /// in the same benchmark whatever the model's size.
  static const int previewTriangles = 150000;

  /// The longest reading a file may take before the viewer gives up.
  static const Duration maxParseTime = Duration(seconds: 10);

  /// [maxBytes], [maxTriangles] and [maxParseTime] as the parser takes them.
  static const QuarkModelLimits limits = QuarkModelLimits(
    maxBytes: maxBytes,
    maxTriangles: maxTriangles,
    maxDuration: maxParseTime,
  );
}
