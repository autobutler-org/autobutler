# autobutler_icons

Custom icon font for Autobutler — spreadsheet and app-specific icons not covered by Material Icons.

## Usage

```dart
import 'package:autobutler_icons/autobutler_icons.dart';

Icon(AutobutlerIcons.insert_row_above)
Icon(AutobutlerIcons.delete_column)
```

Works exactly like `Icons.*` from `package:flutter/material.dart`.

## Adding icons

1. Drop a 24×24 SVG into `svgs/` (stroke-based, `currentColor`)
2. Run `make gen-icons` from the repo root
3. The new icon name is the SVG filename (without `.svg`), snake_cased

## Regenerating the font

```bash
make gen-icons
# or directly:
packages/autobutler_icons/scripts/generate_icons.sh
```

Requires `icon_font_generator` (installed as a dev dependency — `dart pub get` picks it up automatically).

## Icon set

| Constant | Description |
|---|---|
| `insert_row_above` | Insert row above selection |
| `insert_row_below` | Insert row below selection |
| `delete_row` | Delete selected row |
| `duplicate_row` | Duplicate selected row |
| `clear_row` | Clear contents of selected row |
| `insert_column_left` | Insert column left of selection |
| `insert_column_right` | Insert column right of selection |
| `delete_column` | Delete selected column |
| `duplicate_column` | Duplicate selected column |
| `clear_column` | Clear contents of selected column |
