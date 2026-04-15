# DataSheet Control Bar — Intended Features

This file documents the intended capabilities for the `DataSheetControlBar` companion widget. The control bar is optional
— apps may provide their own controls and call methods on `DataSheetController` directly.

Legend: ✅ implemented · 🔜 planned · ❌ out of scope for this package

## Essential Controls

- ✅ Add Row — `DataSheetController.addRow()`
- ✅ Add Column — `DataSheetController.addColumn()`
- ✅ Insert Row Before / After — `insertRowAt(index)`
- ✅ Insert Column Before / After — `insertColumnAt(index)`
- ✅ Delete Row — `deleteRowAt(index)`
- ✅ Delete Column — `deleteColumnAt(index)`

## Editing & Bulk Operations

- ✅ Undo / Redo — snapshot-based, 100-step depth
- ✅ Duplicate Row — `duplicateRow(index)`
- ✅ Duplicate Column — `duplicateColumn(index)`
- ✅ Clear Row — `clearRow(index)` (empties all values)
- ✅ Clear Column — `clearColumn(index)`
- ✅ Clear Cell — `clearCell(row, col)`
- ✅ Fill Down — copies selected cell value to all rows below in the same column
- ✅ Fill Right — copies selected cell value to all columns to the right in the same row
- 🔜 Bulk Edit — apply a value to a rectangular range (needs multi-cell selection)

## Selection & Navigation

- ✅ Go To Cell — dialog, sets selection via `DataSheetSelectionModel.goTo(row, col)`
- 🔜 Select All / Clear Selection — needs multi-cell selection model
- 🔜 Multi-select / rectangular selection — future work
- 🔜 Find (highlight results) — `findCells()` exists on controller; UI highlight not yet wired

## Sort, Filter & Transform

- ✅ Sort by column — `sortByColumn(col, ascending)` with dialog
- ✅ Remove Duplicate Rows — `removeDuplicateRows()`
- 🔜 Filter by column value / predicate
- 🔜 Apply column transformations (trim, case, parse)

## Find & Replace

- ✅ Find — `findCells(query)` on controller
- ✅ Replace All — `replaceCells(from, to)` with dialog, case-sensitive toggle

## Import / Export / Persistence

- ✅ Export CSV — `exportCsv()` returns RFC-4180 string; shown in copy-able dialog
- ✅ Import CSV — `loadFromCsv(csv)` replaces table; dialog accepts pasted text
- 🔜 Export to Excel / spreadsheet format
- 🔜 Save / Load named templates

## View & Layout

- ✅ Column flex configuration — `setColumnFlex`, `updateColumnFlexAt`
- 🔜 Freeze rows / columns (sticky header)
- 🔜 Toggle gridlines visibility
- 🔜 Drag-to-resize columns
- 🔜 Column type / format metadata (text, number, date)

## Advanced Data Features

- 🔜 Formula bar / expression evaluation
- 🔜 Per-cell validation rules
- 🔜 Cell formatting (font weight, alignment, number format)
- 🔜 Conditional formatting rules

## UX / Accessibility

- ✅ Tooltip labels on every toolbar button
- ✅ Disabled states — buttons are null (disabled) when no row/column is selected
- ✅ Context-sensitive enabling — row/column buttons require a cell to be highlighted
- 🔜 Keyboard shortcuts for toolbar actions
- 🔜 Localization / i18n

## Extensibility & Integration

- ✅ Accepts `DataSheetController` (required)
- ✅ Selection model (`DataSheetSelectionModel`) exposed on the controller so custom bars can read it
- 🔜 `onAction` callback hooks for telemetry
- 🔜 Custom builder slot for replacing individual button groups

## Architecture Notes

- `DataSheetSelectionModel` is owned by `DataSheetController` and propagates its changes through the controller's `notifyListeners`,
  so any `ListenableBuilder(listenable: controller)` reacts to both data and selection changes.
- The control bar uses `ListenableBuilder` internally; no external state management needed.
- Library widgets do not embed `MaterialApp` or assume `Directionality` — the host app provides the material tree.
- All mutating controller methods push an undo snapshot before making changes.
