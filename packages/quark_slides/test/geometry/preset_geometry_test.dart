// cspell:words diagStripe flowChartDelay round2SameRect rtTriangle
import 'dart:ui';

import 'package:flutter_test/flutter_test.dart';
import 'package:quark_slides/quark_slides.dart';

void main() {
  const box = Rect.fromLTWH(0, 0, 200, 100);

  void inside(Path? path, Offset point, {String? reason}) {
    expect(path, isNotNull, reason: reason);
    expect(path!.contains(point), isTrue, reason: '${reason ?? ''} $point');
  }

  void outside(Path? path, Offset point, {String? reason}) {
    expect(path, isNotNull, reason: reason);
    expect(path!.contains(point), isFalse, reason: '${reason ?? ''} $point');
  }

  group('diagStripe', () {
    test('a narrow stripe keeps the top-right and cuts the top-left', () {
      final path = presetShapePath('diagStripe', box, const {'adj': 10000});
      inside(path, const Offset(30, 5));
      outside(path, const Offset(1, 1));
    });

    test('a wide stripe covers the lower left and not the upper middle', () {
      final path = presetShapePath('diagStripe', box, const {'adj': 90000});
      outside(path, const Offset(30, 5));
      inside(path, const Offset(10, 95));
    });

    test('the default adjust matches an explicit half', () {
      final explicit = presetShapePath('diagStripe', box, const {'adj': 50000});
      final fallback = presetShapePath('diagStripe', box);
      outside(explicit, const Offset(1, 1));
      inside(explicit, const Offset(150, 10));
      outside(explicit, const Offset(150, 80));
      inside(explicit, const Offset(10, 90));
      for (final point in [
        const Offset(1, 1),
        const Offset(150, 10),
        const Offset(150, 80),
        const Offset(10, 90),
      ]) {
        expect(fallback!.contains(point), explicit!.contains(point));
      }
    });
  });

  test('a trapezoid slopes in from the top corners', () {
    final path = presetShapePath('trapezoid', box, const {'adj': 25000});
    // A rectangle would contain the top-left corner.
    outside(path, const Offset(5, 5));
    inside(path, const Offset(100, 10));
    inside(path, const Offset(1, 98));
  });

  test('a right triangle has its right angle at the bottom left', () {
    final path = presetShapePath('rtTriangle', box);
    outside(path, const Offset(190, 10));
    inside(path, const Offset(10, 90));
    inside(path, const Offset(10, 10));
  });

  group('round2SameRect', () {
    test('small top corners leave the bottom square', () {
      final path = presetShapePath('round2SameRect', box, const {
        'adj1': 4129,
        'adj2': 0,
      });
      outside(path, const Offset(0.2, 0.2));
      inside(path, const Offset(0.2, 99));
      inside(path, const Offset(100, 2));
    });

    test('a large bottom adjust rounds the bottom corner away', () {
      final path = presetShapePath('round2SameRect', box, const {
        'adj1': 4129,
        'adj2': 40000,
      });
      outside(path, const Offset(1, 99));
      inside(path, const Offset(100, 90));
    });
  });

  test('a delay shape is a rectangle with a half ellipse on the right', () {
    final path = presetShapePath('flowChartDelay', box);
    inside(path, const Offset(199, 50));
    outside(path, const Offset(199, 1));
    inside(path, const Offset(1, 1));
  });

  test('a preset follows its box when the box is not at the origin', () {
    const moved = Rect.fromLTWH(10, 20, 200, 100);
    final path = presetShapePath('rtTriangle', moved);
    inside(path, const Offset(20, 110));
    outside(path, const Offset(200, 30));
  });

  test('an unknown preset and an empty box have no outline', () {
    expect(presetShapePath('hexagon', box), isNull);
    expect(presetShapePath(null, box), isNull);
    expect(presetShapePath('rtTriangle', Rect.zero), isNull);
  });

  test('a path with no preset and no contours draws as a rectangle', () {
    final path = shapePath(ShapeKind.path, box);
    expect(path.contains(const Offset(1, 1)), isTrue);
    expect(path.contains(const Offset(190, 10)), isTrue);
    expect(
      shapePath(
        ShapeKind.path,
        box,
        preset: 'hexagon',
      ).contains(const Offset(1, 1)),
      isTrue,
    );
  });

  test('a contour triangle matches the right triangle', () {
    const contour = ShapeContour(
      commands: [
        ShapeMove(0, 1),
        ShapeLine(0, 0),
        ShapeLine(1, 1),
        ShapeClose(),
      ],
    );
    final path = shapePath(ShapeKind.path, box, contours: const [contour]);
    final preset = presetShapePath('rtTriangle', box);
    for (final point in [
      const Offset(190, 10),
      const Offset(10, 90),
      const Offset(10, 10),
    ]) {
      expect(path.contains(point), preset!.contains(point), reason: '$point');
    }
  });

  test('a contour arc matches the delay shape', () {
    const contour = ShapeContour(
      commands: [
        ShapeMove(0, 0),
        ShapeLine(0.5, 0),
        ShapeArc(wR: 0.5, hR: 0.5, start: 270, swing: 180),
        ShapeLine(0, 1),
        ShapeClose(),
      ],
    );
    final path = contourPath(contour, box);
    final preset = presetShapePath('flowChartDelay', box);
    for (final point in [
      const Offset(199, 50),
      const Offset(199, 1),
      const Offset(1, 1),
      const Offset(100, 50),
    ]) {
      expect(path.contains(point), preset!.contains(point), reason: '$point');
    }
  });

  test('an arc on a flat ellipse follows the ray, not the parameter', () {
    // Center (100, 50), radii 80 and 20, from 3 o'clock through 45 degrees.
    // (130, 61) sits on that wedge. A parameter sweep of 45 degrees stays
    // inside a much shallower ray and leaves the point out.
    const contour = ShapeContour(
      commands: [
        ShapeMove(0.5, 0.5),
        ShapeLine(0.9, 0.5),
        ShapeArc(wR: 0.4, hR: 0.2, start: 0, swing: 45),
        ShapeClose(),
      ],
    );
    final path = contourPath(contour, box);
    expect(path.contains(const Offset(130, 61)), isTrue);
    expect(path.contains(const Offset(150, 52)), isTrue);
    expect(path.contains(const Offset(100, 20)), isFalse);
  });

  test('a contour can decline the fill and keep the stroke', () {
    final frame = ElementFrame(x: 0, y: 0, width: 200, height: 100);
    final shape = ShapeElement(
      id: 'p',
      frame: frame,
      kind: ShapeKind.path,
      contours: const [
        ShapeContour(
          commands: [
            ShapeMove(0, 0),
            ShapeLine(1, 0),
            ShapeLine(1, 1),
            ShapeClose(),
          ],
          fill: false,
        ),
      ],
    );
    final pieces = shapePieces(shape, box);
    expect(pieces, hasLength(1));
    expect(pieces.single.fill, isFalse);
    expect(pieces.single.stroke, isTrue);
  });
}
