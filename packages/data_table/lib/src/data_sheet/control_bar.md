# DataSheet Control Bar — Intended Features

This file documents the intended capabilities for the `DataSheetControlBar` companion widget. The control bar is optional — apps may provide their own controls and call methods on `DataSheetController` directly.

## Essential Controls

- Add Row: append an empty row via `DataSheetController.addRow()`.
- Add Column: append an empty column to every row via `DataSheetController.addColumn()`.
- Insert Row / Column: insert at a specified index (before/after).
- Delete Row / Column: remove selected or indexed row/column.

## Editing & Bulk Operations

- Duplicate Row/Column: clone an existing row/column.
- Bulk Edit: apply a value or transformation to a selection or rectangular range.
- Clear Cells: clear values (and optionally formatting) for a selection.
- Undo / Redo: stepwise undo/redo for edits and structural changes.
- Fill Down / Fill Right: copy a value/formula across a range.

## Selection & Navigation

- Select All / Clear Selection.
- Find & Replace across the sheet.
- Go To Cell by coordinates (row/col).
- Toggle multi-select or rectangular selection mode.

## Sort, Filter & Transform

- Sort Column(s) ascending/descending (single or multi-column).
- Filter Column with simple predicates or substring matching.
- Remove Duplicates by column(s).
- Apply Transformations (trim, case, parse number/date) to a column.

## Import / Export / Persistence

- Export CSV / Excel for full table or current selection.
- Import CSV / Paste from clipboard into current selection or new rows.
- Save / Load templates or schema presets.

## View & Layout

- Resize columns (drag or presets).
- Set column types/formats (text, number, date, currency, enum).
- Adjust column flex factors (expose `controller.columnFlex` editing).
- Freeze header rows / left columns.
- Toggle gridlines and header visibility.

## Advanced Data Features

- Formula entry / formula bar with evaluation and result display.
- Validation rules with highlight and error messages.
- Cell formatting (alignment, number formatting, font weight/italic).
- Conditional formatting rules.

## UX / Accessibility

- Keyboard shortcuts (add/delete, undo/redo, navigation).
- Tooltips and accessible labels for controls.
- Disabled / busy states for long-running operations.
- Localization support and theme-awareness.

## Extensibility & Integration

- Accepts a `DataSheetController` (required by default control bar).
- Expose `onAction` callbacks for telemetry or hooks (optional).
- Provide a slot or builder for developers to replace or extend UI.
- Keep control bar logic thin — delegate heavy ops to the controller/service layer.

## Minimal suggested initial subset

For a compact, useful control bar v1 implement:

- Add Row, Add Column, Delete Row, Delete Column, Undo, Redo, Export CSV.

## Notes for implementers

- Library widgets must not assume `MaterialApp` or `Directionality` — the control bar should render correctly inside whatever app theme is used.
- Keep the control bar stateless where possible; rely on `DataSheetController` for state and `notifyListeners()` for updates.
- Provide sensible defaults for button icons, labels, and spacing but allow overrides via constructor parameters.
