import 'package:data_table_example_formulas/evaluation/evaluation.dart';
import 'package:flutter_test/flutter_test.dart';

DependencyGraph _graph(List<((int, int), (int, int))> edges) {
  final g = DependencyGraph();
  for (final e in edges) {
    g.addEdge(e.$1, e.$2);
  }
  return g;
}

void main() {
  group('findCycleMembers (Tarjan SCC)', () {
    test('empty graph returns no cycle members', () {
      expect(findCycleMembers(DependencyGraph()), isEmpty);
    });

    test('graph with no edges returns no cycle members', () {
      final g = DependencyGraph();
      g.ensureNode((0, 0));
      g.ensureNode((0, 1));
      expect(findCycleMembers(g), isEmpty);
    });

    test('simple acyclic chain returns no cycle members', () {
      // (0,0) → (0,1) → (0,2)
      final g = _graph([((0, 0), (0, 1)), ((0, 1), (0, 2))]);
      expect(findCycleMembers(g), isEmpty);
    });

    test('self-reference is detected as a cycle', () {
      final g = _graph([((0, 0), (0, 0))]);
      expect(findCycleMembers(g), {(0, 0)});
    });

    test('two-node mutual reference is detected', () {
      final g = _graph([((0, 0), (0, 1)), ((0, 1), (0, 0))]);
      expect(findCycleMembers(g), {(0, 0), (0, 1)});
    });

    test('three-node cycle is fully detected', () {
      final g = _graph([
        ((0, 0), (0, 1)),
        ((0, 1), (0, 2)),
        ((0, 2), (0, 0)),
      ]);
      expect(findCycleMembers(g), {(0, 0), (0, 1), (0, 2)});
    });

    test('cycle members do not include acyclic nodes in the same graph', () {
      // (0,2) → (0,0) ↔ (0,1): only (0,0) and (0,1) are in a cycle.
      final g = _graph([
        ((0, 0), (0, 1)),
        ((0, 1), (0, 0)),
        ((0, 2), (0, 0)), // acyclic node depending on cycle
      ]);
      expect(findCycleMembers(g), {(0, 0), (0, 1)});
    });

    test('two independent cycles are both detected', () {
      final g = _graph([
        ((0, 0), (0, 0)), // self-loop
        ((1, 0), (1, 1)),
        ((1, 1), (1, 0)), // mutual
      ]);
      expect(findCycleMembers(g), {(0, 0), (1, 0), (1, 1)});
    });
  });
}
