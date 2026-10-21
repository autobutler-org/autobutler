// cspell:words wR hR
import '../format/json_fields.dart';
import '../format/qslide_format_exception.dart';

/// One outline of a path shape, in the frame's own space.
///
/// Coordinates run from 0 at the left or top of the frame to 1 at the right
/// or bottom, so resizing the frame scales the outline. Arc radii are
/// fractions of the frame's width and height, and arc angles are degrees
/// clockwise from 3 o'clock. [fill] and [stroke] say whether this outline
/// takes the shape's fill and its outline; both default to drawing.
///
/// ```json
/// {"commands": [{"op": "move", "x": 0, "y": 1},
///   {"op": "line", "x": 0, "y": 0}, {"op": "line", "x": 1, "y": 1},
///   {"op": "close"}]}
/// ```
class ShapeContour {
  /// Creates an outline from [commands].
  const ShapeContour({
    this.commands = const [],
    this.fill = true,
    this.stroke = true,
  });

  /// The commands, in order.
  final List<ShapeCommand> commands;

  /// Whether the shape's fill is drawn inside this outline.
  final bool fill;

  /// Whether the shape's stroke is drawn along this outline.
  final bool stroke;

  /// Reads an outline from its `.qslide` object at [path].
  factory ShapeContour.fromJson(Object? value, String path) {
    final json = asObject(value, path);
    final commands = optionalList(json, 'commands', path);
    return ShapeContour(
      commands: List.unmodifiable([
        for (var i = 0; i < commands.length; i++)
          ShapeCommand.fromJson(commands[i], '$path.commands[$i]'),
      ]),
      fill: optionalBool(json, 'fill', path, true),
      stroke: optionalBool(json, 'stroke', path, true),
    );
  }

  /// The outline as its `.qslide` object.
  JsonMap toJson() => {
    'commands': [for (final command in commands) command.toJson()],
    if (!fill) 'fill': false,
    if (!stroke) 'stroke': false,
  };

  @override
  bool operator ==(Object other) =>
      other is ShapeContour &&
      other.fill == fill &&
      other.stroke == stroke &&
      listEquals(other.commands, commands);

  @override
  int get hashCode => Object.hash(fill, stroke, Object.hashAll(commands));
}

/// One command of a [ShapeContour].
sealed class ShapeCommand {
  const ShapeCommand();

  /// Reads a command from its `.qslide` object at [path].
  factory ShapeCommand.fromJson(Object? value, String path) {
    final json = asObject(value, path);
    final op = requireString(json, 'op', path);
    double n(String key) => requireNumber(json, key, path);
    return switch (op) {
      'move' => ShapeMove(n('x'), n('y')),
      'line' => ShapeLine(n('x'), n('y')),
      'quad' => ShapeQuad(n('x1'), n('y1'), n('x'), n('y')),
      'cubic' => ShapeCubic(n('x1'), n('y1'), n('x2'), n('y2'), n('x'), n('y')),
      'arc' => ShapeArc(
        wR: n('wR'),
        hR: n('hR'),
        start: n('start'),
        swing: n('swing'),
      ),
      'close' => const ShapeClose(),
      _ => throw QslideFormatException(
        'unknown path command',
        path: '$path.op',
      ),
    };
  }

  /// The command as its `.qslide` object.
  JsonMap toJson();
}

/// Starts a subpath at ([x], [y]).
final class ShapeMove extends ShapeCommand {
  /// Creates a move to ([x], [y]).
  const ShapeMove(this.x, this.y);

  /// The horizontal position, as a fraction of the frame width.
  final double x;

  /// The vertical position, as a fraction of the frame height.
  final double y;

  @override
  JsonMap toJson() => {'op': 'move', 'x': jsonNumber(x), 'y': jsonNumber(y)};

  @override
  bool operator ==(Object other) =>
      other is ShapeMove && other.x == x && other.y == y;

  @override
  int get hashCode => Object.hash(x, y);
}

/// A straight line to ([x], [y]).
final class ShapeLine extends ShapeCommand {
  /// Creates a line to ([x], [y]).
  const ShapeLine(this.x, this.y);

  /// The horizontal position, as a fraction of the frame width.
  final double x;

  /// The vertical position, as a fraction of the frame height.
  final double y;

  @override
  JsonMap toJson() => {'op': 'line', 'x': jsonNumber(x), 'y': jsonNumber(y)};

  @override
  bool operator ==(Object other) =>
      other is ShapeLine && other.x == x && other.y == y;

  @override
  int get hashCode => Object.hash(x, y);
}

/// A quadratic curve to ([x], [y]) with control point ([x1], [y1]).
final class ShapeQuad extends ShapeCommand {
  /// Creates a quadratic curve.
  const ShapeQuad(this.x1, this.y1, this.x, this.y);

  /// The control point's horizontal position.
  final double x1;

  /// The control point's vertical position.
  final double y1;

  /// The end point's horizontal position.
  final double x;

  /// The end point's vertical position.
  final double y;

  @override
  JsonMap toJson() => {
    'op': 'quad',
    'x1': jsonNumber(x1),
    'y1': jsonNumber(y1),
    'x': jsonNumber(x),
    'y': jsonNumber(y),
  };

  @override
  bool operator ==(Object other) =>
      other is ShapeQuad &&
      other.x1 == x1 &&
      other.y1 == y1 &&
      other.x == x &&
      other.y == y;

  @override
  int get hashCode => Object.hash(x1, y1, x, y);
}

/// A cubic curve to ([x], [y]) with control points ([x1], [y1]) and
/// ([x2], [y2]).
final class ShapeCubic extends ShapeCommand {
  /// Creates a cubic curve.
  const ShapeCubic(this.x1, this.y1, this.x2, this.y2, this.x, this.y);

  /// The first control point's horizontal position.
  final double x1;

  /// The first control point's vertical position.
  final double y1;

  /// The second control point's horizontal position.
  final double x2;

  /// The second control point's vertical position.
  final double y2;

  /// The end point's horizontal position.
  final double x;

  /// The end point's vertical position.
  final double y;

  @override
  JsonMap toJson() => {
    'op': 'cubic',
    'x1': jsonNumber(x1),
    'y1': jsonNumber(y1),
    'x2': jsonNumber(x2),
    'y2': jsonNumber(y2),
    'x': jsonNumber(x),
    'y': jsonNumber(y),
  };

  @override
  bool operator ==(Object other) =>
      other is ShapeCubic &&
      other.x1 == x1 &&
      other.y1 == y1 &&
      other.x2 == x2 &&
      other.y2 == y2 &&
      other.x == x &&
      other.y == y;

  @override
  int get hashCode => Object.hash(x1, y1, x2, y2, x, y);
}

/// An elliptical arc. [wR] and [hR] are the radii as fractions of the
/// frame's width and height. [start] and [swing] are degrees clockwise
/// from 3 o'clock; the arc begins at the current point.
final class ShapeArc extends ShapeCommand {
  /// Creates an arc.
  const ShapeArc({
    required this.wR,
    required this.hR,
    required this.start,
    required this.swing,
  });

  /// The horizontal radius, as a fraction of the frame width.
  final double wR;

  /// The vertical radius, as a fraction of the frame height.
  final double hR;

  /// The start angle, in degrees clockwise from 3 o'clock.
  final double start;

  /// How far the arc sweeps, in degrees clockwise.
  final double swing;

  @override
  JsonMap toJson() => {
    'op': 'arc',
    'wR': jsonNumber(wR),
    'hR': jsonNumber(hR),
    'start': jsonNumber(start),
    'swing': jsonNumber(swing),
  };

  @override
  bool operator ==(Object other) =>
      other is ShapeArc &&
      other.wR == wR &&
      other.hR == hR &&
      other.start == start &&
      other.swing == swing;

  @override
  int get hashCode => Object.hash(wR, hR, start, swing);
}

/// Closes the current subpath back to its start.
final class ShapeClose extends ShapeCommand {
  /// Creates a close command.
  const ShapeClose();

  @override
  JsonMap toJson() => const {'op': 'close'};

  @override
  bool operator ==(Object other) => other is ShapeClose;

  @override
  int get hashCode => 0;
}
