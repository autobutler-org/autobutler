part of 'preset_geometry.dart';

/// A preset path in [box], or null when a command names an unknown guide.
Path? _tracePreset(List<_Step> steps, Map<String, double> guides, Rect box) {
  final cursor = _Cursor();
  double? coord(String name) => double.tryParse(name) ?? guides[name];
  for (final step in steps) {
    switch (step) {
      case _Move(:final x, :final y):
        final px = coord(x), py = coord(y);
        if (px == null || py == null) return null;
        cursor.move(box.topLeft + Offset(px, py));
      case _Line(:final x, :final y):
        final px = coord(x), py = coord(y);
        if (px == null || py == null) return null;
        cursor.line(box.topLeft + Offset(px, py));
      case _Arc(:final wR, :final hR, :final start, :final swing):
        final rx = coord(wR),
            ry = coord(hR),
            st = coord(start),
            sw = coord(swing);
        if (rx == null || ry == null || st == null || sw == null) return null;
        // Arc angles are stored in 60,000ths of a degree.
        cursor.arc(rx, ry, st / 60000, sw / 60000);
      case _Close():
        cursor.close();
    }
  }
  return cursor.path;
}

/// Walks commands while remembering the current point, which [Path] does
/// not report. Angles are degrees clockwise from 3 o'clock.
class _Cursor {
  final path = Path();
  var _current = Offset.zero;
  var _subpath = Offset.zero;

  void move(Offset point) {
    path.moveTo(point.dx, point.dy);
    _current = _subpath = point;
  }

  void line(Offset point) {
    path.lineTo(point.dx, point.dy);
    _current = point;
  }

  void quad(Offset control, Offset point) {
    path.quadraticBezierTo(control.dx, control.dy, point.dx, point.dy);
    _current = point;
  }

  void cubic(Offset c1, Offset c2, Offset point) {
    path.cubicTo(c1.dx, c1.dy, c2.dx, c2.dy, point.dx, point.dy);
    _current = point;
  }

  void close() {
    path.close();
    _current = _subpath;
  }

  /// An elliptical arc of radii [wR] and [hR] beginning at the current
  /// point. A zero radius or a zero swing leaves the point where it is.
  /// The sweep is held to one full turn.
  void arc(double wR, double hR, double startDeg, double swingDeg) {
    if (wR <= 0 || hR <= 0 || swingDeg == 0) return;
    if (!wR.isFinite || !hR.isFinite) return;
    final st = startDeg * math.pi / 180;
    final sw = swingDeg.clamp(-360.0, 360.0) * math.pi / 180;
    final turns = (sw / (2 * math.pi)).truncateToDouble();
    final sw0 = sw - turns * 2 * math.pi;
    final a0 = _ellipseAngle(st, wR, hR);
    final a1 = _ellipseAngle(st + sw0, wR, hR);
    var sweep = a1 - a0;
    if (sw0 > 0 && sweep <= 0) sweep += 2 * math.pi;
    if (sw0 < 0 && sweep >= 0) sweep -= 2 * math.pi;
    sweep += turns * 2 * math.pi;
    final cx = _current.dx - wR * math.cos(a0);
    final cy = _current.dy - hR * math.sin(a0);
    path.arcTo(
      Rect.fromCenter(center: Offset(cx, cy), width: wR * 2, height: hR * 2),
      a0,
      sweep,
      false,
    );
    _current = Offset(
      cx + wR * math.cos(a0 + sweep),
      cy + hR * math.sin(a0 + sweep),
    );
  }
}

/// The ellipse parameter for a ray at [radians] clockwise from 3 o'clock.
///
/// A circle's parameter is the ray angle. On a wider or taller ellipse the
/// ray and the parameter differ, and the arc has to be drawn by the
/// parameter so it passes through the point the ray names.
double _ellipseAngle(double radians, double wR, double hR) =>
    math.atan2(wR * math.sin(radians), hR * math.cos(radians));
