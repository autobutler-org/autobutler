/// Directed dependency graph keyed by (row, col) cell coordinates.
///
/// An edge `from → to` means the cell at [from] depends on the cell at [to].
class DependencyGraph {
  /// Maps each cell to the set of cells it directly depends on.
  final Map<(int, int), Set<(int, int)>> dependsOn = {};

  /// Maps each cell to the set of cells that directly depend on it.
  final Map<(int, int), Set<(int, int)>> dependedOnBy = {};

  /// Records a dependency: [from] depends on [to].
  void addEdge((int, int) from, (int, int) to) {
    dependsOn.putIfAbsent(from, () => {}).add(to);
    dependedOnBy.putIfAbsent(to, () => {}).add(from);
    // Ensure both endpoints exist in both maps.
    dependsOn.putIfAbsent(to, () => {});
    dependedOnBy.putIfAbsent(from, () => {});
  }

  /// Ensures [cell] is present in both adjacency maps even if it has no edges.
  void ensureNode((int, int) cell) {
    dependsOn.putIfAbsent(cell, () => {});
    dependedOnBy.putIfAbsent(cell, () => {});
  }
}
