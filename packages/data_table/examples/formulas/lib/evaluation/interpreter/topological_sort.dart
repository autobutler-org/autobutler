import 'dart:collection';

/// Result of a Kahn's-algorithm topological sort.
///
/// [order] contains all nodes in a valid topological order.  [remaining]
/// contains any nodes that could not be ordered because they are part of a
/// cycle (non-empty only when [graph] contained a cycle that was not excluded
/// via [excludedNodes]).
class TopologicalOrder {
  /// Nodes in a valid topological order (leaves first).
  final List<(int, int)> order;

  /// Nodes with unresolved dependencies; non-empty if a cycle was present.
  final Set<(int, int)> remaining;

  const TopologicalOrder({required this.order, required this.remaining});
}

/// Runs Kahn's topological-sort algorithm on the subgraph induced by [nodes]
/// using [dependsOn] as the adjacency map.
///
/// [excludedNodes] are treated as already resolved: edges to them do not
/// contribute to a node's in-degree.  This is used to handle cycle members
/// that have been pre-assigned error values before the topological pass.
///
/// Only edges whose targets are in [nodes] count toward in-degree; edges to
/// non-formula (literal) cells are ignored.
TopologicalOrder kahnSort(
  Set<(int, int)> nodes,
  Map<(int, int), Set<(int, int)>> dependsOn, {
  Set<(int, int)> excludedNodes = const {},
}) {
  // Working set: nodes minus excluded.
  final activeNodes = nodes.difference(excludedNodes);

  // in-degree counts only edges from active formula cells.
  final inDegree = <(int, int), int>{};
  final dependedOnBy = <(int, int), Set<(int, int)>>{};

  for (final cell in activeNodes) {
    var degree = 0;
    for (final dep in dependsOn[cell] ?? const <(int, int)>{}) {
      if (excludedNodes.contains(dep)) continue;
      if (!activeNodes.contains(dep)) continue; // literal cells: ignore
      degree++;
      dependedOnBy.putIfAbsent(dep, () => {}).add(cell);
    }
    inDegree[cell] = degree;
  }

  final queue = Queue<(int, int)>();
  for (final entry in inDegree.entries) {
    if (entry.value == 0) queue.add(entry.key);
  }

  final order = <(int, int)>[];
  while (queue.isNotEmpty) {
    final cell = queue.removeFirst();
    order.add(cell);

    for (final dependent in dependedOnBy[cell] ?? const <(int, int)>{}) {
      final newDegree = (inDegree[dependent] ?? 1) - 1;
      inDegree[dependent] = newDegree;
      if (newDegree == 0) queue.add(dependent);
    }
  }

  final remaining = activeNodes.difference(order.toSet());
  return TopologicalOrder(order: order, remaining: remaining);
}
