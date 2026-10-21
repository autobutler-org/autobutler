import 'dart:math' as math;

import '../format/json_fields.dart';
import 'slide_color.dart';

/// The drop shadow a shape or picture casts: its silhouette in [color],
/// moved by [x] and [y] and softened by [blur], all in slide units.
///
/// The offset is the slide's own, not the element's: a rotated element's
/// shadow still falls the same way, as if lit from one place. [blur] is the
/// distance the edge fades over; the canvas draws it as a Gaussian blur of
/// standard deviation `blur / 2`.
///
/// In `.qslide` it is an element's `shadow` object; the offset and blur are
/// left out at 0.
///
/// ```json
/// {"color": "#0000007F", "x": -4.24, "y": 4.24, "blur": 61}
/// ```
class ElementShadow {
  /// Creates a shadow; [blur] must not be negative.
  ElementShadow({
    required this.color,
    this.x = 0,
    this.y = 0,
    this.blur = 0,
    this.extra = const {},
  }) : assert(blur >= 0, 'a shadow blur cannot be negative');

  /// The shadow's color, alpha included.
  final SlideColor color;

  /// How far right of the element the shadow falls, in slide units.
  final double x;

  /// How far below the element the shadow falls, in slide units.
  final double y;

  /// How far the shadow's edge fades, in slide units.
  final double blur;

  /// Fields a newer writer added that this version does not read.
  final JsonMap extra;

  static const _known = {'color', 'x', 'y', 'blur'};

  /// Reads a shadow from its `.qslide` object at [path]. A negative blur
  /// reads as 0.
  factory ElementShadow.fromJson(Object? value, String path) {
    final json = asObject(value, path);
    final color = optionalString(json, 'color', path);
    return ElementShadow(
      color: color == null
          ? SlideColor.black
          : SlideColor.parse(color, path: '$path.color'),
      x: optionalNumber(json, 'x', path) ?? 0,
      y: optionalNumber(json, 'y', path) ?? 0,
      blur: math.max(0, optionalNumber(json, 'blur', path) ?? 0),
      extra: unknownFields(json, _known),
    );
  }

  /// The shadow as its `.qslide` object.
  JsonMap toJson() => {
    ...extra,
    'color': color.toHex(),
    if (x != 0) 'x': jsonNumber(x),
    if (y != 0) 'y': jsonNumber(y),
    if (blur != 0) 'blur': jsonNumber(blur),
  };

  @override
  bool operator ==(Object other) =>
      other is ElementShadow &&
      other.color == color &&
      other.x == x &&
      other.y == y &&
      other.blur == blur &&
      jsonEquals(other.extra, extra);

  @override
  int get hashCode => Object.hash(color, x, y, blur, jsonHash(extra));

  @override
  String toString() => 'ElementShadow(${color.toHex()}, $x, $y, $blur)';
}
