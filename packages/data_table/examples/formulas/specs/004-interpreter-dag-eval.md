## Overview

Build DataSheetInterpreter that orchestrates full-sheet parsing, dependency graph construction, cycle detection (Tarjan SCC), and topological evaluation (Kahn's algorithm). Produces a Map<(row,col),FormulaValue> of computed values for formula cells.

## Acceptance Criteria

- interpretSheet API implemented:
  Map<(int,int),FormulaValue> interpretSheet(int rows,int cols,String Function(int,int) rawCellValue)
- Stage 1: Parse all formula cells into ParsedFormula map (null for literals).
- Stage 2: Build DependencyGraph from ParsedFormula.cellRefs and expanded rangeRefs.
- Stage 3: Detect cycles using Tarjan's SCC (or equivalent). Mark cycle members with ErrorValue('#REF!', 'Circular reference') and remove cycle edges to form maximal acyclic subgraph.
- Stage 4: Topologically evaluate remaining DAG using Kahn's algorithm; literal cells are leaves; cycle-marked cells have preassigned error values.
- Cells depending on cycle nodes receive the propagated ErrorValue naturally.
- Unit tests for direct self-reference, multi-hop cycles, and correct topological ordering.

## Out of Scope

- Incremental re-evaluation optimisations (future ticket).
- Parallel parsing/evaluation across isolates (left for perf follow-up).

## Notes

- DependencyGraph stores dependsOn and dependedOnBy maps keyed by (row,col) tuples.
- Evaluation must use an accessor that reads from the already-computed results map; no recursive access during topological pass.
- Keep implementation single-threaded for MVP.