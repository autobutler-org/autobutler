import 'package:flutter/rendering.dart';

import '../model/slide_element.dart';
import '../model/stroke.dart';
import '../theme/slide_theme.dart';
import 'slide_shadow_painter.dart';
import 'slide_shape_paths.dart';

/// Paints a [ShapeElement]'s figure, fill and outline, fitted to the box it
/// is given — the element's frame, in slide units. The outline is dashed as
/// its [Stroke.dash] says; the shape's opacity is applied around the
/// painter, so a fill and an outline that overlap do not darken. Its
/// [ShapeElement.shadow], when it has one, is cast by the fill and outline
/// together and painted first (see [paintShadow]).
class SlideShapePainter extends CustomPainter {
  /// Creates a painter for [shape].
  const SlideShapePainter(this.shape, this.theme);

  /// The shape to paint.
  final ShapeElement shape;

  /// The theme a role color resolves against.
  final SlideTheme theme;

  @override
  void paint(Canvas canvas, Size size) {
    final pieces = shapePieces(shape, Offset.zero & size);
    if (shape.shadow case final shadow?) {
      paintShadow(canvas, shadow, shape.frame.rotation, theme,
          (canvas) => _paintPieces(canvas, pieces));
    }
    _paintPieces(canvas, pieces);
  }

  /// Fills and outlines [pieces] as the shape says.
  void _paintPieces(Canvas canvas, List<ShapePiece> pieces) {
    final fill = shape.fill;
    final stroke = shape.stroke;
    for (final piece in pieces) {
      if (piece.fill && fill != null) {
        canvas.drawPath(
          piece.path,
          Paint()..color = Color(fill.resolve(theme)),
        );
      }
      if (piece.stroke && stroke != null && stroke.width > 0) {
        final intervals = dashIntervals(stroke.dash, stroke.width);
        canvas.drawPath(
          intervals == null ? piece.path : dashedPath(piece.path, intervals),
          Paint()
            ..style = PaintingStyle.stroke
            ..strokeWidth = stroke.width
            ..strokeJoin = StrokeJoin.miter
            ..color = Color(stroke.color.resolve(theme)),
        );
      }
    }
  }

  @override
  bool shouldRepaint(SlideShapePainter oldDelegate) =>
      oldDelegate.shape != shape || oldDelegate.theme != theme;
}
