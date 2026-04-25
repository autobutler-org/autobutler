import 'dependency_graph.dart';

/// Runs Tarjan's strongly-connected-components algorithm on [graph] and
/// returns the set of nodes that participate in a cycle.
///
/// A node with a self-edge is considered a cycle of size 1.  All members of
/// any SCC of size > 1 are also included.
Set<(int, int)> findCycleMembers(DependencyGraph graph) {
  final cycleMembers = <(int, int)>{};
  final index = <(int, int), int>{};
  final lowlink = <(int, int), int>{};
  final onStack = <(int, int), bool>{};
  final stack = <(int, int)>[];
  var indexCounter = 0;

  void strongConnect((int, int) v) {
    index[v] = indexCounter;
    lowlink[v] = indexCounter;
    indexCounter++;
    stack.add(v);
    onStack[v] = true;

    for (final w in graph.dependsOn[v] ?? const <(int, int)>{}) {
      if (!index.containsKey(w)) {
        strongConnect(w);
        final wLow = lowlink[w]!;
        if (wLow < lowlink[v]!) lowlink[v] = wLow;
      } else if (onStack[w] == true) {
        final wIdx = index[w]!;
        if (wIdx < lowlink[v]!) lowlink[v] = wIdx;
      }
    }

    if (lowlink[v] == index[v]) {
      final scc = <(int, int)>[];
      (int, int) w;
      do {
        w = stack.removeLast();
        onStack[w] = false;
        scc.add(w);
      } while (w != v);

      // A trivial SCC of size 1 is only a cycle if the node has a self-edge.
      final isCycle = scc.length > 1 ||
          (graph.dependsOn[scc[0]]?.contains(scc[0]) ?? false);
      if (isCycle) {
        cycleMembers.addAll(scc);
      }
    }
  }

  for (final cell in graph.dependsOn.keys) {
    if (!index.containsKey(cell)) {
      strongConnect(cell);
    }
  }

  return cycleMembers;
}
