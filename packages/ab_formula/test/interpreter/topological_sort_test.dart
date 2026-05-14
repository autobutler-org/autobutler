import 'package:ab_formula/evaluation/evaluation.dart';
import 'package:test/test.dart';

void main() {
  group('kahnSort (topological sort)', () {
    test('empty node set returns empty order', () {
      final result = kahnSort({}, {});
      expect(result.order, isEmpty);
      expect(result.remaining, isEmpty);
    });

    test('single node with no dependencies is ordered', () {
      final result = kahnSort({(0, 0)}, {(0, 0): {}});
      expect(result.order, [(0, 0)]);
      expect(result.remaining, isEmpty);
    });

    test('linear chain is ordered leaves-first', () {
      // (0,2) depends on (0,1) depends on (0,0)
      final deps = {
        (0, 0): <(int, int)>{},
        (0, 1): {(0, 0)},
        (0, 2): {(0, 1)},
      };
      final result = kahnSort({(0, 0), (0, 1), (0, 2)}, deps);
      expect(result.order, [(0, 0), (0, 1), (0, 2)]);
      expect(result.remaining, isEmpty);
    });

    test('diamond dependency resolves both branches before the root', () {
      // (0,3) depends on (0,1) and (0,2); both depend on (0,0).
      final deps = {
        (0, 0): <(int, int)>{},
        (0, 1): {(0, 0)},
        (0, 2): {(0, 0)},
        (0, 3): {(0, 1), (0, 2)},
      };
      final result = kahnSort({(0, 0), (0, 1), (0, 2), (0, 3)}, deps);

      expect(result.remaining, isEmpty);
      // (0,0) must come before (0,1) and (0,2); both before (0,3).
      final order = result.order;
      expect(order.indexOf((0, 0)), lessThan(order.indexOf((0, 1))));
      expect(order.indexOf((0, 0)), lessThan(order.indexOf((0, 2))));
      expect(order.indexOf((0, 1)), lessThan(order.indexOf((0, 3))));
      expect(order.indexOf((0, 2)), lessThan(order.indexOf((0, 3))));
    });

    test('excluded nodes are not present in the output', () {
      final deps = {
        (0, 0): <(int, int)>{},
        (0, 1): {(0, 0)},
      };
      final result = kahnSort({(0, 0), (0, 1)}, deps, excludedNodes: {(0, 0)});
      expect(result.order, [(0, 1)]);
      expect(result.remaining, isEmpty);
    });

    test('cycle in input surfaces in remaining (no excludedNodes)', () {
      // (0,0) ↔ (0,1) — pure cycle with no excluded help.
      final deps = {
        (0, 0): {(0, 1)},
        (0, 1): {(0, 0)},
      };
      final result = kahnSort({(0, 0), (0, 1)}, deps);
      expect(result.order, isEmpty);
      expect(result.remaining, {(0, 0), (0, 1)});
    });

    test('edges to literal (non-formula) cells do not inflate in-degree', () {
      // (0,1) depends on (0,0), but (0,0) is not in the formula node set.
      final deps = {
        (0, 1): {(0, 0)},
      };
      final result = kahnSort({(0, 1)}, deps);
      expect(result.order, [(0, 1)]);
      expect(result.remaining, isEmpty);
    });
  });
}
