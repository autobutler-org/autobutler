import 'package:quark_formula/evaluation/evaluation.dart';
import 'package:test/test.dart';

void main() {
  group('DependencyGraph', () {
    test('starts empty', () {
      final g = DependencyGraph();
      expect(g.dependsOn, isEmpty);
      expect(g.dependedOnBy, isEmpty);
    });

    test('ensureNode registers a node with empty adjacency sets', () {
      final g = DependencyGraph();
      g.ensureNode((0, 0));
      expect(g.dependsOn[(0, 0)], isEmpty);
      expect(g.dependedOnBy[(0, 0)], isEmpty);
    });

    test('ensureNode is idempotent', () {
      final g = DependencyGraph();
      g.ensureNode((0, 0));
      g.ensureNode((0, 0));
      expect(g.dependsOn.keys.where((k) => k == (0, 0)), hasLength(1));
    });

    test('addEdge records dependsOn for the source', () {
      final g = DependencyGraph();
      g.addEdge((0, 1), (0, 0));
      expect(g.dependsOn[(0, 1)], contains((0, 0)));
    });

    test('addEdge records dependedOnBy for the target', () {
      final g = DependencyGraph();
      g.addEdge((0, 1), (0, 0));
      expect(g.dependedOnBy[(0, 0)], contains((0, 1)));
    });

    test('addEdge ensures both endpoints exist in both maps', () {
      final g = DependencyGraph();
      g.addEdge((0, 1), (0, 0));
      expect(g.dependsOn.containsKey((0, 0)), isTrue);
      expect(g.dependedOnBy.containsKey((0, 1)), isTrue);
    });

    test('addEdge self-loop is recorded correctly', () {
      final g = DependencyGraph();
      g.addEdge((0, 0), (0, 0));
      expect(g.dependsOn[(0, 0)], contains((0, 0)));
      expect(g.dependedOnBy[(0, 0)], contains((0, 0)));
    });

    test('multiple edges from the same source are all recorded', () {
      final g = DependencyGraph();
      g.addEdge((0, 2), (0, 0));
      g.addEdge((0, 2), (0, 1));
      expect(g.dependsOn[(0, 2)], containsAll([(0, 0), (0, 1)]));
    });

    test('multiple edges to the same target are all recorded', () {
      final g = DependencyGraph();
      g.addEdge((0, 0), (0, 2));
      g.addEdge((0, 1), (0, 2));
      expect(g.dependedOnBy[(0, 2)], containsAll([(0, 0), (0, 1)]));
    });

    test('addEdge is idempotent for duplicate edges', () {
      final g = DependencyGraph();
      g.addEdge((0, 1), (0, 0));
      g.addEdge((0, 1), (0, 0));
      expect(g.dependsOn[(0, 1)], hasLength(1));
      expect(g.dependedOnBy[(0, 0)], hasLength(1));
    });
  });
}
