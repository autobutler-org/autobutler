import 'package:flutter_test/flutter_test.dart';
import 'package:quark/utils/svg_viewport.dart';

void main() {
  test('leaves a viewBox alone, whatever the width and height say', () {
    const markup = '<svg viewBox="0 0 24 24" width="100%"><rect/></svg>';
    expect(svgWithViewport(markup), markup);
  });

  test('leaves absolute lengths alone', () {
    for (final markup in [
      '<svg width="10" height="20"/>',
      "<svg width='10px' height='2.5em'/>",
    ]) {
      expect(svgWithViewport(markup), markup);
    }
  });

  test('gives a root with no size the browser default', () {
    expect(
      svgWithViewport('<?xml version="1.0"?><svg xmlns="x"><rect/></svg>'),
      '<?xml version="1.0"?>'
      '<svg height="150" width="300" xmlns="x"><rect/></svg>',
    );
  });

  test('replaces only the length the decoder cannot read', () {
    expect(
      svgWithViewport('<svg width="100%" height="40"><rect/></svg>'),
      '<svg width="300" height="40"><rect/></svg>',
    );
    expect(
      svgWithViewport('<svg width="40" height="10cm"/>'),
      '<svg width="40" height="150"/>',
    );
    expect(
      svgWithViewport('<svg width="40"/>'),
      '<svg height="150" width="40"/>',
    );
  });

  test('ignores a nested svg and a rect that carries its own width', () {
    expect(
      svgWithViewport('<svg><svg width="50%"/><rect width="50%"/></svg>'),
      '<svg height="150" width="300"><svg width="50%"/><rect width="50%"/></svg>',
    );
  });

  test('returns markup with no svg tag unchanged', () {
    expect(svgWithViewport('not an svg'), 'not an svg');
  });
}
