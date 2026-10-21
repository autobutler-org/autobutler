part of 'preset_geometry.dart';

/// One preset: its default adjust values, the guides computed from them,
/// and the path those guides draw.
class _Preset {
  const _Preset(this.defaults, this.guides, this.steps);

  /// Adjust guide names and the literal each one takes when unset.
  final List<(String, double)> defaults;

  /// Guides in evaluation order: a name and its formula.
  final List<(String, String)> guides;

  /// The path, naming guides for its coordinates.
  final List<_Step> steps;
}

final class _Move extends _Step {
  const _Move(this.x, this.y);
  final String x;
  final String y;
}

final class _Line extends _Step {
  const _Line(this.x, this.y);
  final String x;
  final String y;
}

final class _Arc extends _Step {
  const _Arc(this.wR, this.hR, this.start, this.swing);
  final String wR;
  final String hR;
  final String start;
  final String swing;
}

final class _Close extends _Step {
  const _Close();
}

sealed class _Step {
  const _Step();
}

/// The five presets an import draws as a path. Adding another is data here
/// and a matching name in the importer; the rest of the preset set is not
/// evaluated and still imports as a rectangle.
const _presets = {
  'diagStripe': _Preset(
    [('adj', 50000)],
    [
      ('a', 'pin 0 adj 100000'),
      ('x2', '*/ w a 100000'),
      ('x1', '*/ x2 1 2'),
      ('x3', '+/ x2 r 2'),
      ('y2', '*/ h a 100000'),
      ('y1', '*/ y2 1 2'),
      ('y3', '+/ y2 b 2'),
    ],
    [
      _Move('l', 'y2'),
      _Line('x2', 't'),
      _Line('r', 't'),
      _Line('l', 'b'),
      _Close(),
    ],
  ),
  'trapezoid': _Preset(
    [('adj', 25000)],
    [
      ('maxAdj', '*/ 50000 w ss'),
      ('a', 'pin 0 adj maxAdj'),
      ('x1', '*/ ss a 200000'),
      ('x2', '*/ ss a 100000'),
      ('x3', '+- r 0 x2'),
      ('x4', '+- r 0 x1'),
      ('il', '*/ wd3 a maxAdj'),
      ('it', '*/ hd3 a maxAdj'),
      ('ir', '+- r 0 il'),
    ],
    [
      _Move('l', 'b'),
      _Line('x2', 't'),
      _Line('x3', 't'),
      _Line('r', 'b'),
      _Close(),
    ],
  ),
  'rtTriangle': _Preset(
    [],
    [('it', '*/ h 7 12'), ('ir', '*/ w 7 12'), ('ib', '*/ h 11 12')],
    [_Move('l', 'b'), _Line('l', 't'), _Line('r', 'b'), _Close()],
  ),
  'round2SameRect': _Preset(
    [('adj1', 16667), ('adj2', 0)],
    [
      ('a1', 'pin 0 adj1 50000'),
      ('a2', 'pin 0 adj2 50000'),
      ('tx1', '*/ ss a1 100000'),
      ('tx2', '+- r 0 tx1'),
      ('bx1', '*/ ss a2 100000'),
      ('bx2', '+- r 0 bx1'),
      ('by1', '+- b 0 bx1'),
      ('d', '+- tx1 0 bx1'),
      ('tdx', '*/ tx1 29289 100000'),
      ('bdx', '*/ bx1 29289 100000'),
      ('il', '?: d tdx bdx'),
      ('ir', '+- r 0 il'),
      ('ib', '+- b 0 bdx'),
    ],
    [
      _Move('tx1', 't'),
      _Line('tx2', 't'),
      _Arc('tx1', 'tx1', '3cd4', 'cd4'),
      _Line('r', 'by1'),
      _Arc('bx1', 'bx1', '0', 'cd4'),
      _Line('bx1', 'b'),
      _Arc('bx1', 'bx1', 'cd4', 'cd4'),
      _Line('l', 'tx1'),
      _Arc('tx1', 'tx1', 'cd2', 'cd4'),
      _Close(),
    ],
  ),
  'flowChartDelay': _Preset(
    [],
    [
      ('idx', 'cos wd2 2700000'),
      ('idy', 'sin hd2 2700000'),
      ('ir', '+- hc idx 0'),
      ('it', '+- vc 0 idy'),
      ('ib', '+- vc idy 0'),
    ],
    [
      _Move('l', 't'),
      _Line('hc', 't'),
      _Arc('wd2', 'hd2', '3cd4', 'cd2'),
      _Line('l', 'b'),
      _Close(),
    ],
  ),
};

/// Built-in guides for a box [w] by [h], then the preset's defaults, then
/// [adjust] by name, then the preset's own guides. A missing name, a
/// division by zero, or a non-finite result fails the whole evaluation.
Map<String, double>? _evaluate(
  _Preset preset,
  double w,
  double h,
  Map<String, double> adjust,
) {
  final built = _builtins(w, h);
  final guides = Map<String, double>.of(built);
  for (final (name, value) in preset.defaults) {
    if (!built.containsKey(name)) guides[name] = value;
  }
  for (final entry in adjust.entries) {
    if (!built.containsKey(entry.key)) guides[entry.key] = entry.value;
  }
  for (final (name, formula) in preset.guides) {
    final value = _formula(formula, guides);
    if (value == null) return null;
    guides[name] = value;
  }
  return guides;
}

Map<String, double> _builtins(double w, double h) {
  final ss = w < h ? w : h;
  final ls = w > h ? w : h;
  final guides = <String, double>{
    'w': w,
    'h': h,
    'l': 0,
    't': 0,
    'r': w,
    'b': h,
    'hc': w / 2,
    'vc': h / 2,
    'ss': ss,
    'ls': ls,
  };
  for (final n in [2, 3, 4, 5, 6, 8, 10, 12, 32]) {
    guides['wd$n'] = w / n;
    guides['hd$n'] = h / n;
  }
  for (final n in [2, 4, 6, 8, 16, 32]) {
    guides['ssd$n'] = ss / n;
  }
  const turn = 21600000.0;
  for (final n in [2, 3, 4, 8]) {
    guides['cd$n'] = turn / n;
  }
  guides['3cd4'] = 3 * turn / 4;
  guides['3cd8'] = 3 * turn / 8;
  guides['5cd8'] = 5 * turn / 8;
  guides['7cd8'] = 7 * turn / 8;
  return guides;
}

/// One guide formula. Arguments are literals or earlier guide names; an
/// argument that is left off is 0. Angles are in 60,000ths of a degree.
double? _formula(String formula, Map<String, double> guides) {
  final parts = formula.trim().split(RegExp(r'\s+'));
  if (parts.isEmpty || parts.first.isEmpty) return null;
  double? arg(int index) {
    if (index >= parts.length) return 0;
    final token = parts[index];
    return double.tryParse(token) ?? guides[token];
  }

  final x = arg(1);
  final y = arg(2);
  final z = arg(3);
  if (x == null || y == null || z == null) return null;
  final value = switch (parts[0]) {
    'val' => x,
    '*/' => z == 0 ? null : x * y / z,
    '+-' => x + y - z,
    '+/' => z == 0 ? null : (x + y) / z,
    '?:' => x > 0 ? y : z,
    'abs' => x.abs(),
    'min' => x < y ? x : y,
    'max' => x > y ? x : y,
    'sqrt' => math.sqrt(x.abs()),
    'sin' => x * math.sin(_radians(y)),
    'cos' => x * math.cos(_radians(y)),
    'tan' => x * math.tan(_radians(y)),
    'at2' => math.atan2(y, x) * 10800000 / math.pi,
    'mod' => math.sqrt(x * x + y * y + z * z),
    'pin' => y < x ? x : (y > z ? z : y),
    'cat2' => x * math.cos(math.atan2(z, y)),
    'sat2' => x * math.sin(math.atan2(z, y)),
    _ => null,
  };
  if (value == null || value.isNaN || value.isInfinite) return null;
  return value;
}

/// [ang] is in 60,000ths of a degree.
double _radians(double ang) => ang / 60000 * math.pi / 180;
