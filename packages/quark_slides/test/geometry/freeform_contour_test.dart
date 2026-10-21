import 'dart:ui';

import 'package:flutter_test/flutter_test.dart';
import 'package:quark_slides/quark_slides.dart';

/// The contours below are what `pkg/util/pptxutil` writes for an imported
/// freeform (#3631), so these tests pin the importer's output to the
/// editor's drawing of it.
void main() {
  ShapeContour read(Map<String, Object?> json) =>
      ShapeContour.fromJson(json, r'$');

  Offset end(Path path) {
    final metric = path.computeMetrics().last;
    return metric.getTangentForOffset(metric.length)!.position;
  }

  // The five-point outline from the issue, a 84254 by 48898 path.
  final outline = read({
    'commands': [
      {'op': 'move', 'x': 0, 'y': 0},
      {'op': 'line', 'x': 0.59723, 'y': 0},
      {'op': 'line', 'x': 1, 'y': 0.693996},
      {'op': 'line', 'x': 1, 'y': 1},
      {'op': 'line', 'x': 0.579177, 'y': 1},
      {'op': 'close'},
    ],
  });

  test('an imported freeform draws its outline, not its box', () {
    const box = Rect.fromLTWH(10, 20, 200, 150);
    final path = contourPath(outline, box);
    expect(path.getBounds(), box);
    // The cut top-right corner and the open bottom-left are left out.
    expect(path.contains(const Offset(205, 25)), isFalse);
    expect(path.contains(const Offset(20, 160)), isFalse);
    expect(path.contains(const Offset(150, 100)), isTrue);
  });

  test('an imported freeform scales with its frame', () {
    const wide = Rect.fromLTWH(0, 0, 400, 100);
    final path = contourPath(outline, wide);
    expect(path.getBounds(), wide);
    // The diagonal edge moves with the frame: 0.597 of the width along
    // the top, 0.694 of the height down the right.
    expect(path.contains(const Offset(236, 2)), isTrue);
    expect(path.contains(const Offset(250, 2)), isFalse);
    expect(path.contains(const Offset(398, 72)), isTrue);
    expect(path.contains(const Offset(398, 64)), isFalse);
  });

  test('a stretched arc ends where the path said it would', () {
    // A 100 by 100 path in a 200 by 150 frame: from (75, 25), an arc of
    // radius 10 from 45 degrees through -90 ends at (75, 10.858) in the
    // path, (150, 16.287) in the frame. The importer carried the angles
    // into the frame's aspect.
    final contour = read({
      'commands': [
        {'op': 'move', 'x': 0.25, 'y': 0.25},
        {'op': 'line', 'x': 0.75, 'y': 0.25},
        {
          'op': 'arc',
          'wR': 0.1,
          'hR': 0.1,
          'start': 36.869898,
          'swing': -73.739795,
        },
      ],
    });
    final at = end(contourPath(contour, const Rect.fromLTWH(0, 0, 200, 150)));
    expect(at.dx, closeTo(150, 0.01));
    expect(at.dy, closeTo(16.287, 0.01));
  });

  test('an unfilled freeform path keeps its stroke', () {
    final shape = ShapeElement(
      id: 'f',
      frame: ElementFrame(x: 0, y: 0, width: 200, height: 150),
      kind: ShapeKind.path,
      contours: [
        read({
          'commands': [
            {'op': 'move', 'x': 0, 'y': 0.5},
            {'op': 'quad', 'x1': 0, 'y1': 0, 'x': 0.5, 'y': 0},
          ],
          'fill': false,
        }),
        read({
          'commands': [
            {'op': 'move', 'x': 0, 'y': 0},
            {'op': 'line', 'x': 1, 'y': 1},
          ],
          'stroke': false,
        }),
      ],
    );
    final pieces = shapePieces(shape, const Rect.fromLTWH(0, 0, 200, 150));
    expect(
      [for (final p in pieces) (p.fill, p.stroke)],
      [(false, true), (true, false)],
    );
  });
}
