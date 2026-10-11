import 'package:flutter_test/flutter_test.dart';
import 'package:quark_widgets/quark_widgets.dart';

/// `findCodeMatches` is the search behind `QuarkCodeViewer`: plain substring,
/// case-insensitive, in line order.
void main() {
  CodeSearchMatch hit(int line, int start, int end) =>
      CodeSearchMatch(line: line, start: start, end: end);

  test('matches in any case', () {
    expect(findCodeMatches(['Error: disk', 'no ERROR here'], 'error'), [
      hit(0, 0, 5),
      hit(1, 3, 8),
    ]);
  });

  test('finds every hit on a line, without overlap', () {
    expect(findCodeMatches(['abc.abc abc'], 'abc'), [
      hit(0, 0, 3),
      hit(0, 4, 7),
      hit(0, 8, 11),
    ]);
    expect(findCodeMatches(['aaaa'], 'aa'), [hit(0, 0, 2), hit(0, 2, 4)]);
  });

  test('treats the query as plain text, not a pattern', () {
    expect(findCodeMatches(['a.b', 'axb', r'(\d+)'], 'a.b'), [hit(0, 0, 3)]);
    expect(findCodeMatches([r'(\d+)'], r'(\d+)'), [hit(0, 0, 5)]);
  });

  test('keeps offsets true when lowercasing would change the length', () {
    // 'İ'.toLowerCase() is two code units.
    expect(findCodeMatches(['İİ needle'], 'NEEDLE'), [hit(0, 3, 9)]);
  });

  test('stops at the limit', () {
    final lines = List.filled(10, 'x x x');
    expect(findCodeMatches(lines, 'x', limit: 7), hasLength(7));
    expect(findCodeMatches(lines, 'x', limit: 7).last, hit(2, 0, 1));
    expect(findCodeMatches(lines, 'x'), hasLength(30));
  });

  test('an empty query finds nothing', () {
    expect(findCodeMatches(['abc', ''], ''), isEmpty);
    expect(findCodeMatches([], 'abc'), isEmpty);
  });

  test('hits compare by value', () {
    expect(hit(1, 2, 3), hit(1, 2, 3));
    expect(hit(1, 2, 3).hashCode, hit(1, 2, 3).hashCode);
    expect(hit(1, 2, 3), isNot(hit(1, 2, 4)));
  });
}
