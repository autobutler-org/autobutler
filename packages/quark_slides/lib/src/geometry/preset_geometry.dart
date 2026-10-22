// cspell:words diagStripe flowChartDelay round2SameRect rtTriangle
/// Preset outlines and frame-relative contours for a path shape.
///
/// A preset is evaluated from its guide formulas at the box it is drawn
/// in, so a geometry that depends on the box's aspect stays correct as the
/// shape is resized. A contour is already in the frame's own space and
/// scales with it. Keep the preset names here in step with `pathPresets`
/// in `pkg/util/pptxutil/import_shapes.go`.
library;

import 'dart:math' as math;
import 'dart:ui';

import '../model/shape_contour.dart';

part 'preset_formula.dart';
part 'preset_path.dart';

/// The outline of the preset [name] fitted to [box], or null when [name]
/// is not a preset this package evaluates or [box] has no area.
///
/// [adjust] replaces that preset's default adjust values, by guide name.
/// A built-in guide such as the box width is never replaced.
///
/// ```dart
/// presetShapePath('rtTriangle', const Rect.fromLTWH(0, 0, 200, 100));
/// ```
Path? presetShapePath(
  String? name,
  Rect box, [
  Map<String, double> adjust = const {},
]) {
  if (name == null || box.width <= 0 || box.height <= 0) return null;
  final preset = _presets[name];
  if (preset == null) return null;
  final guides = _evaluate(preset, box.width, box.height, adjust);
  if (guides == null) return null;
  return _tracePreset(preset.steps, guides, box);
}

/// [contour] drawn inside [box].
///
/// Coordinates and arc radii are fractions of [box]; arc angles are degrees
/// clockwise from 3 o'clock. A zero radius draws nothing and leaves the
/// current point where it is.
Path contourPath(ShapeContour contour, Rect box) {
  final cursor = _Cursor();
  for (final command in contour.commands) {
    switch (command) {
      case ShapeMove(:final x, :final y):
        cursor.move(_at(box, x, y));
      case ShapeLine(:final x, :final y):
        cursor.line(_at(box, x, y));
      case ShapeQuad(:final x1, :final y1, :final x, :final y):
        cursor.quad(_at(box, x1, y1), _at(box, x, y));
      case ShapeCubic(
          :final x1,
          :final y1,
          :final x2,
          :final y2,
          :final x,
          :final y,
        ):
        cursor.cubic(_at(box, x1, y1), _at(box, x2, y2), _at(box, x, y));
      case ShapeArc(:final wR, :final hR, :final start, :final swing):
        cursor.arc(wR * box.width, hR * box.height, start, swing);
      case ShapeClose():
        cursor.close();
    }
  }
  return cursor.path;
}

Offset _at(Rect box, double x, double y) =>
    box.topLeft + Offset(box.width * x, box.height * y);
