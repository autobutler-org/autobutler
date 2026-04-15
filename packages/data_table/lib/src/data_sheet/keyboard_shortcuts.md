# DataSheet Keyboard Shortcuts

Legend: ✅ implemented · 🔜 planned · ❌ out of scope for this package

Shortcuts are handled inside the `DataSheet` widget itself via its `Focus` +
`onKeyEvent` handler. They work regardless of whether a `DataSheetControlBar`
is present.

The entire key binding map is customizable — developers can supply their own
`DataSheetControlScheme` and even swap it at runtime. See
[Control Scheme Customization](#control-scheme-customization) below.

---

## Navigation

| Keys          | Action                                        | Status |
| ------------- | --------------------------------------------- | ------ |
| Arrow keys    | Move highlight one cell                       | ✅     |
| Tab           | Move highlight right; wraps at row end        | ✅     |
| Shift+Tab     | Move highlight left                           | ✅     |
| Enter         | Confirm edit / activate highlighted cell      | ✅     |
| Ctrl/Cmd+Home | Jump to first cell (row 0, col 0)             | ✅     |
| Ctrl/Cmd+End  | Jump to last cell                             | ✅     |
| Page Down     | Move highlight down by visible-page height    | 🔜     |
| Page Up       | Move highlight up by visible-page height      | 🔜     |
| Home          | Move highlight to first column in current row | ✅     |
| End           | Move highlight to last column in current row  | ✅     |

---

## Editing

| Keys                           | Action                                     | Status |
| ------------------------------ | ------------------------------------------ | ------ |
| Any printable key              | Open cell for editing (start typing)       | ✅     |
| F2                             | Enter edit mode for the highlighted cell   | ✅     |
| Escape                         | Cancel active edit, restore previous value | ✅     |
| Delete / Backspace             | Clear the highlighted cell's value         | ✅     |
| Ctrl/Cmd+Z                     | Undo last action                           | ✅     |
| Ctrl/Cmd+Y or Ctrl/Cmd+Shift+Z | Redo                                       | ✅     |

---

## Clipboard

| Keys       | Action                                          | Status |
| ---------- | ----------------------------------------------- | ------ |
| Ctrl/Cmd+C | Copy highlighted cell value to system clipboard | ✅     |
| Ctrl/Cmd+X | Cut — copy then clear highlighted cell          | ✅     |
| Ctrl/Cmd+V | Paste clipboard text into highlighted cell      | ✅     |

---

## Selection

| Keys               | Action                                          | Status |
| ------------------ | ----------------------------------------------- | ------ |
| Shift+Arrow        | Extend selection by one cell in arrow direction | 🔜     |
| Ctrl/Cmd+A         | Select all cells                                | 🔜     |
| Ctrl/Cmd+Shift+End | Extend selection to last used cell              | 🔜     |

---

## Data Operations

| Keys       | Action                                                            | Status |
| ---------- | ----------------------------------------------------------------- | ------ |
| Ctrl/Cmd+D | Fill down — copy value of cell above into highlighted cell        | ✅     |
| Ctrl/Cmd+R | Fill right — copy value of cell to the left into highlighted cell | ✅     |
| Ctrl/Cmd+F | Open Find & Replace dialog                                        | 🔜     |
| Ctrl/Cmd+G | Open Go To Cell dialog                                            | 🔜     |

---

## Row / Column Structural Operations

| Keys                 | Action                                | Status |
| -------------------- | ------------------------------------- | ------ |
| Ctrl/Cmd+Plus        | Insert row above highlighted cell     | ✅     |
| Ctrl/Cmd+Minus       | Delete row of highlighted cell        | ✅     |
| Ctrl/Cmd+Shift+Plus  | Insert column before highlighted cell | ✅     |
| Ctrl/Cmd+Shift+Minus | Delete column of highlighted cell     | ✅     |

---

## Implementation Notes

- All shortcuts are dispatched through `DataSheetControlScheme` — the active
  scheme is resolved each key event via `widget.controlScheme ?? DataSheetControlScheme.defaults()`.
- `KeyboardShortcut.matches` checks `HardwareKeyboard.instance.isControlPressed || isMetaPressed`
  for the `ctrl` flag, so the same scheme works on Windows/Linux and macOS.
- Clipboard operations use the `flutter/services.dart` `Clipboard` API — no
  extra packages required. `_pasteCell` is async; the cell update is scheduled
  after the future resolves and guarded by a `mounted` check.
- `_priorCellValue` is captured in `_activateCell` so that pressing Escape can
  restore the original value without touching undo history.
- Structural shortcuts delegate to `DataSheetController` methods that push an
  undo snapshot, so Ctrl+Z recovers inserted/deleted rows and columns.
- Page Up / Page Down require access to the scroll position and visible row
  count; they remain 🔜 until a `ScrollController` is wired into the view.
- Ctrl+F and Ctrl+G open dialogs; those are owned by the control bar rather than
  the sheet itself and remain 🔜 at the sheet level.

---

## Control Scheme Customization

### Motivations

- Different users have muscle memory for different editors (Excel, Google Sheets,
  Vim, etc.).
- Apps may need to reserve certain key combos for their own use.
- Accessibility requirements may mandate different bindings.

### Proposed API

`DataSheet` should accept an optional `controlScheme` parameter:

```dart
DataSheet(
  controller: myController,
  table: myTable,
  controlScheme: DataSheetControlScheme.excel(), // built-in preset
)
```

If omitted, `DataSheet` falls back to `DataSheetControlScheme.defaults()`.

### `DataSheetControlScheme`

A `DataSheetControlScheme` is a plain data class (no Flutter dependency) that
maps each logical **action** to one or more **key triggers**. Developers can:

1. Use a built-in preset (`defaults`, `excel`, `googleSheets`, `vim`).
2. Start from a preset and override individual bindings:

   ```dart
   final scheme = DataSheetControlScheme.excel().copyWith(
     undo: [KeyboardShortcut.ctrl(LogicalKeyboardKey.keyZ)],
     fillDown: [KeyboardShortcut.ctrl(LogicalKeyboardKey.keyD)],
   );
   ```

3. Build one entirely from scratch via the default constructor.

Swapping the scheme at runtime is supported — `DataSheet` reads it on each key
event, so passing a new scheme to a live widget via `setState` takes effect
immediately with no special handling.

### `KeyboardShortcut`

A lightweight value type describing a single trigger:

```dart
class KeyboardShortcut {
  final LogicalKeyboardKey key;
  final bool ctrl;   // also matches Cmd on macOS
  final bool shift;
  final bool alt;

  const KeyboardShortcut(this.key,
      {this.ctrl = false, this.shift = false, this.alt = false});

  factory KeyboardShortcut.ctrl(LogicalKeyboardKey key) =>
      KeyboardShortcut(key, ctrl: true);

  factory KeyboardShortcut.ctrlShift(LogicalKeyboardKey key) =>
      KeyboardShortcut(key, ctrl: true, shift: true);
}
```

### Named Actions

Each field of `DataSheetControlScheme` maps to one of the actions in this
document. The full set of action names (each takes `List<KeyboardShortcut>`):

| Field              | Default trigger       |
| ------------------ | --------------------- |
| `moveUp`           | Arrow Up              |
| `moveDown`         | Arrow Down            |
| `moveLeft`         | Arrow Left            |
| `moveRight`        | Arrow Right           |
| `moveNextCell`     | Tab                   |
| `movePreviousCell` | Shift+Tab             |
| `confirmEdit`      | Enter                 |
| `enterEditMode`    | F2                    |
| `cancelEdit`       | Escape                |
| `clearCell`        | Delete or Backspace   |
| `undo`             | Ctrl+Z                |
| `redo`             | Ctrl+Y / Ctrl+Shift+Z |
| `copy`             | Ctrl+C                |
| `cut`              | Ctrl+X                |
| `paste`            | Ctrl+V                |
| `selectAll`        | Ctrl+A                |
| `fillDown`         | Ctrl+D                |
| `fillRight`        | Ctrl+R                |
| `findReplace`      | Ctrl+F                |
| `goToCell`         | Ctrl+G                |
| `jumpToFirst`      | Ctrl+Home             |
| `jumpToLast`       | Ctrl+End              |
| `jumpRowStart`     | Home                  |
| `jumpRowEnd`       | End                   |
| `pageUp`           | Page Up               |
| `pageDown`         | Page Down             |
| `insertRow`        | Ctrl+Plus             |
| `deleteRow`        | Ctrl+Minus            |
| `insertColumn`     | Ctrl+Shift+Plus       |
| `deleteColumn`     | Ctrl+Shift+Minus      |

### Saving and Loading Schemes

Serialization is left to the caller. `DataSheetControlScheme` should implement
`toJson()` / `fromJson()` so apps can:

- Persist a user's preferred scheme to `SharedPreferences`, a file, or a
  database.
- Ship pre-built scheme files (JSON) and load them at startup.
- Let users import/export schemes through their own settings UI.

The package does not bundle any particular persistence mechanism — keeping the
model serializable is enough.
