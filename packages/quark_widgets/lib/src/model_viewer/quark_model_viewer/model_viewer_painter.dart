import 'package:flutter/rendering.dart';

import '../quark_model_camera.dart';
import 'model_renderer.dart';

/// Paints a `QuarkModelViewer`'s canvas: [renderer]'s mesh as [camera] sees
/// it, shaded in [color] or, with [wireframe], as edges in [edgeColor].
class ModelViewerPainter extends CustomPainter {
  /// Creates the painter.
  ModelViewerPainter({
    required this.renderer,
    required this.camera,
    required this.color,
    required this.edgeColor,
    required this.wireframe,
  });

  /// Draws the mesh; it is reused across frames for its buffers.
  final ModelRenderer renderer;

  /// Where the mesh is seen from.
  final QuarkModelCamera camera;

  /// The color of a face square to the light.
  final Color color;

  /// The color of an edge in wireframe.
  final Color edgeColor;

  /// Whether to draw edges instead of faces.
  final bool wireframe;

  @override
  void paint(Canvas canvas, Size size) {
    canvas.clipRect(Offset.zero & size);
    renderer.paint(
      canvas,
      size,
      camera,
      color: color,
      edgeColor: edgeColor,
      wireframe: wireframe,
    );
  }

  @override
  bool shouldRepaint(ModelViewerPainter oldDelegate) =>
      oldDelegate.renderer != renderer ||
      oldDelegate.camera != camera ||
      oldDelegate.color != color ||
      oldDelegate.edgeColor != edgeColor ||
      oldDelegate.wireframe != wireframe;
}
