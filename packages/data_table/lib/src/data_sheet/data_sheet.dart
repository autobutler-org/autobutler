import 'package:flutter/material.dart' hide DataTable, DataRow, DataCell;
import 'package:flutter/services.dart';

import '../../data_table.dart';
import 'cell/cell.dart';
import 'cell/editable_cell.dart';
import 'cell/heading/heading_cells.dart';
import 'cell/heading/util.dart';
import 'data_sheet_control_scheme.dart';
import 'data_sheet_controller.dart';

class DataSheet extends StatelessWidget {
  final DataTable table;

  /// Callback that is called before a cell value is changed.
  /// If it returns false, the change is rejected and the cell
  /// value remains unchanged.
  final bool Function(String, int, int)? beforeCellValueChanged;

  /// Callback that is called after a cell value is changed.
  /// The boolean parameter indicates whether the change had been
  /// accepted or rejected.
  final Function(String, int, int, bool)? afterCellValueChanged;

  /// Optional external controller. If omitted, a controller is created
  /// from the provided `table` and owned by the internal view.
  final DataSheetController? controller;

  /// Optional per-column flex factors. If provided and no external
  /// controller is supplied, these are forwarded to the internal controller.
  final List<int>? columnFlex;

  /// Optional key binding scheme. Defaults to
  /// [DataSheetControlScheme.defaults] when `null`.
  ///
  /// Provide a customized scheme to remap or disable individual shortcuts.
  /// The scheme can be swapped at runtime by passing a new value from a
  /// parent widget.
  final DataSheetControlScheme? controlScheme;

  /// Whether to show the column-letter header row and row-number gutter.
  /// Defaults to `true`.
  final bool showHeadings;

  const DataSheet(
      {super.key,
      required this.table,
      this.beforeCellValueChanged,
      this.afterCellValueChanged,
      this.controller,
      this.columnFlex,
      this.controlScheme,
      this.showHeadings = true});

  DataSheet.unnamed({super.key})
      : table = DataTable([]),
        beforeCellValueChanged = null,
        afterCellValueChanged = null,
        controller = null,
        columnFlex = null,
        controlScheme = null,
        showHeadings = true;

  @override
  Widget build(BuildContext context) {
    return _DataSheetView(
      table: table,
      controller: controller,
      columnFlex: columnFlex,
      beforeCellValueChanged: beforeCellValueChanged,
      afterCellValueChanged: afterCellValueChanged,
      controlScheme: controlScheme,
      showHeadings: showHeadings,
    );
  }
}

class _DataSheetView extends StatefulWidget {
  final DataTable table;
  final DataSheetController? controller;
  final List<int>? columnFlex;
  final bool Function(String, int, int)? beforeCellValueChanged;
  final Function(String, int, int, bool)? afterCellValueChanged;
  final DataSheetControlScheme? controlScheme;
  final bool showHeadings;

  const _DataSheetView({
    required this.table,
    this.controller,
    this.columnFlex,
    this.beforeCellValueChanged,
    this.afterCellValueChanged,
    this.controlScheme,
    this.showHeadings = true,
  });

  @override
  State<_DataSheetView> createState() => _DataSheetViewState();
}

class _DataSheetViewState extends State<_DataSheetView> {
  late final TextEditingController activeCellController;
  late final FocusNode keyboardFocus;
  late final DataSheetController controller;
  late final bool _ownsController;
  String _priorCellValue = '';
  String? _internalClipboard;

  int get activeRow => controller.selection.activeRow;
  int get activeCol => controller.selection.activeCol;
  int get highlightedRow => controller.selection.highlightedRow;
  int get highlightedCol => controller.selection.highlightedCol;

  @override
  void initState() {
    super.initState();
    activeCellController = TextEditingController();
    keyboardFocus = FocusNode();
    if (widget.controller != null) {
      controller = widget.controller!;
      _ownsController = false;
    } else {
      controller = DataSheetController.fromTable(widget.table,
          columnFlex: widget.columnFlex);
      _ownsController = true;
    }
    controller.addListener(_onControllerChanged);
  }

  @override
  void dispose() {
    activeCellController.dispose();
    keyboardFocus.dispose();
    controller.removeListener(_onControllerChanged);
    if (_ownsController) controller.dispose();
    super.dispose();
  }

  void _onControllerChanged() {
    if (mounted) setState(() {});
  }

  @override
  Widget build(BuildContext context) {
    return Focus(
        focusNode: keyboardFocus,
        onKeyEvent: (node, event) {
          if (event is! KeyDownEvent) return KeyEventResult.ignored;
          final scheme =
              widget.controlScheme ?? DataSheetControlScheme.defaults();
          bool m(List<KeyboardShortcut> triggers) =>
              triggers.any((t) => t.matches(event));

          // Modifier shortcuts are checked first (most specific) to prevent
          // them from falling through to plain-key handlers.
          if (m(scheme.undo)) {
            controller.undo();
            return KeyEventResult.handled;
          }
          if (m(scheme.redo)) {
            controller.redo();
            return KeyEventResult.handled;
          }
          if (m(scheme.copy)) {
            _copyCell();
            return KeyEventResult.handled;
          }
          if (m(scheme.cut)) {
            _cutCell();
            return KeyEventResult.handled;
          }
          if (m(scheme.paste)) {
            _pasteCell();
            return KeyEventResult.handled;
          }
          if (m(scheme.fillDown)) {
            _fillDown();
            return KeyEventResult.handled;
          }
          if (m(scheme.fillRight)) {
            _fillRight();
            return KeyEventResult.handled;
          }
          if (m(scheme.jumpToFirst)) {
            _jumpToFirst();
            return KeyEventResult.handled;
          }
          if (m(scheme.jumpToLast)) {
            _jumpToLast();
            return KeyEventResult.handled;
          }
          if (m(scheme.insertRow)) {
            _insertRow();
            return KeyEventResult.handled;
          }
          if (m(scheme.deleteRow)) {
            _deleteRow();
            return KeyEventResult.handled;
          }
          if (m(scheme.insertColumn)) {
            _insertColumn();
            return KeyEventResult.handled;
          }
          if (m(scheme.deleteColumn)) {
            _deleteColumn();
            return KeyEventResult.handled;
          }

          // Plain / shift shortcuts.
          if (m(scheme.moveUp)) {
            _moveUp();
            return KeyEventResult.handled;
          }
          if (m(scheme.moveDown)) {
            _moveDown();
            return KeyEventResult.handled;
          }
          if (m(scheme.moveLeft)) {
            _moveLeft();
            return KeyEventResult.handled;
          }
          if (m(scheme.moveRight)) {
            _moveRight();
            return KeyEventResult.handled;
          }
          if (m(scheme.movePreviousCell)) {
            if (activeRow >= 0 && activeCol >= 0) {
              final r = activeRow;
              final c = activeCol;
              _storeCellValue(activeCellController.text,
                  highlightRow: r, highlightCol: c);
              keyboardFocus.requestFocus();
            }
            _moveLeft();
            return KeyEventResult.handled;
          }
          if (m(scheme.moveNextCell)) {
            if (activeRow >= 0 && activeCol >= 0) {
              final r = activeRow;
              final c = activeCol;
              _storeCellValue(activeCellController.text,
                  highlightRow: r, highlightCol: c);
              keyboardFocus.requestFocus();
            }
            _moveRight();
            return KeyEventResult.handled;
          }
          if (m(scheme.confirmEdit)) {
            if (highlightedRow >= 0 && highlightedCol >= 0) {
              _activateCell(controller.cellAt(highlightedRow, highlightedCol),
                  highlightedRow, highlightedCol);
            } else if (activeRow >= 0 && activeCol >= 0) {
              final r = activeRow;
              final c = activeCol;
              _storeCellValue(activeCellController.text,
                  highlightRow: r, highlightCol: c);
              keyboardFocus.requestFocus();
            }
            return KeyEventResult.handled;
          }
          if (m(scheme.enterEditMode)) {
            if (highlightedRow >= 0 && highlightedCol >= 0) {
              _activateCell(controller.cellAt(highlightedRow, highlightedCol),
                  highlightedRow, highlightedCol);
            }
            return KeyEventResult.handled;
          }
          if (m(scheme.cancelEdit)) {
            if (activeRow >= 0 && activeCol >= 0) {
              _storeCellValue(_priorCellValue,
                  highlightRow: activeRow, highlightCol: activeCol);
              keyboardFocus.requestFocus();
            } else {
              controller.selection.clear();
            }
            return KeyEventResult.handled;
          }
          if (m(scheme.clearCell)) {
            if (highlightedRow >= 0 && highlightedCol >= 0) {
              controller.clearCell(highlightedRow, highlightedCol);
            }
            return KeyEventResult.handled;
          }
          if (m(scheme.jumpRowStart)) {
            if (highlightedRow >= 0) {
              controller.selection.setHighlighted(highlightedRow, 0);
            }
            return KeyEventResult.handled;
          }
          if (m(scheme.jumpRowEnd)) {
            if (highlightedRow >= 0) {
              controller.selection
                  .setHighlighted(highlightedRow, controller.colCount - 1);
            }
            return KeyEventResult.handled;
          }

          // Any other key while a cell is highlighted: start editing it.
          // Skip modifier-only keypresses (Ctrl, Cmd, Shift, Alt) so that
          // pressing a modifier alone does not inadvertently activate a cell.
          final modifierKeys = {
            LogicalKeyboardKey.control,
            LogicalKeyboardKey.controlLeft,
            LogicalKeyboardKey.controlRight,
            LogicalKeyboardKey.meta,
            LogicalKeyboardKey.metaLeft,
            LogicalKeyboardKey.metaRight,
            LogicalKeyboardKey.shift,
            LogicalKeyboardKey.shiftLeft,
            LogicalKeyboardKey.shiftRight,
            LogicalKeyboardKey.alt,
            LogicalKeyboardKey.altLeft,
            LogicalKeyboardKey.altRight,
          };
          if (modifierKeys.contains(event.logicalKey)) {
            return KeyEventResult.ignored;
          }
          if (highlightedRow >= 0 && highlightedCol >= 0) {
            _activateCell(controller.cellAt(highlightedRow, highlightedCol),
                highlightedRow, highlightedCol);
          }
          return KeyEventResult.ignored;
        },
        child: Column(
          children: [
            // ── Column header row ────────────────────────────────────────
            if (widget.showHeadings)
              ListenableBuilder(
                listenable: controller,
                builder: (context, _) {
                  return Row(
                    children: [
                      // Corner cell above row-number gutter
                      HeaderCornerCell(),
                      ...List.generate(controller.colCount, (c) {
                        final flex = (c < controller.columnFlex.length)
                            ? controller.columnFlex[c]
                            : 1;
                        return Expanded(
                          flex: flex,
                          child: ColumnHeaderCell(label: columnLabel(c)),
                        );
                      }),
                    ],
                  );
                },
              ),
            // ── Data rows ────────────────────────────────────────────────
            Expanded(
              child: ListView.builder(
                itemCount: controller.rowCount,
                itemBuilder: (context, r) {
                  return ValueListenableBuilder<List<DataCell>>(
                    valueListenable: controller.rowNotifier(r),
                    builder: (context, rowCells, _) {
                      return Row(
                        children: [
                          // Row number gutter
                          if (widget.showHeadings) RowNumberCell(number: r + 1),
                          ...List.generate(rowCells.length, (c) {
                            final isActiveCell =
                                (r == activeRow && c == activeCol);
                            final isHighlightedCell =
                                (r == highlightedRow && c == highlightedCol);

                            final cellChild = isActiveCell
                                ? EditableCell(
                                    key: ValueKey('r${r}c$c'),
                                    controller: activeCellController,
                                    onSubmitted: _storeCellValue,
                                    onEditingComplete: () {
                                      _storeCellValue(
                                          activeCellController.text);
                                    },
                                    onTapOutside: (_) {
                                      _storeCellValue(
                                        activeCellController.text,
                                        highlightRow: activeRow,
                                        highlightCol: activeCol,
                                      );
                                      keyboardFocus.requestFocus();
                                    },
                                  )
                                : Text(rowCells[c].value);

                            final flex = (c < controller.columnFlex.length)
                                ? controller.columnFlex[c]
                                : 1;
                            final cell = Cell(
                              key: ValueKey('r${r}c$c'),
                              isActive: isActiveCell,
                              isHighlighted: isHighlightedCell,
                              cursor: isActiveCell
                                  ? SystemMouseCursors.text
                                  : SystemMouseCursors.cell,
                              child: cellChild,
                            );
                            return Expanded(
                              flex: flex,
                              child: isActiveCell
                                  ? cell
                                  : GestureDetector(
                                      behavior: HitTestBehavior.opaque,
                                      onTap: () {
                                        if (activeRow >= 0 && activeCol >= 0) {
                                          _storeCellValue(
                                              activeCellController.text);
                                        } else {
                                          if (highlightedRow == r &&
                                              highlightedCol == c) {
                                            _activateCell(rowCells[c], r, c);
                                          } else {
                                            controller.selection
                                                .setHighlighted(r, c);
                                            keyboardFocus.requestFocus();
                                          }
                                        }
                                      },
                                      child: cell,
                                    ),
                            );
                          }),
                        ],
                      );
                    },
                  );
                },
              ),
            ),
          ],
        ));
  }

  void _storeCellValue(
    String value, {
    int highlightRow = -1,
    int highlightCol = -1,
  }) {
    final changedRow = activeRow;
    final changedCol = activeCol;
    if (changedRow < 0 || changedCol < 0) return;
    final isChangeAccepted =
        widget.beforeCellValueChanged?.call(value, changedRow, changedCol) ??
            true;
    if (isChangeAccepted) {
      controller.updateCell(changedRow, changedCol, DataCell(value));
    }
    activeCellController.clear();
    if (highlightRow >= 0) {
      controller.selection.goTo(highlightRow, highlightCol);
    } else {
      controller.selection.clear();
    }
    widget.afterCellValueChanged
        ?.call(value, changedRow, changedCol, isChangeAccepted);
  }

  void _activateCell(DataCell cell, int row, int col) {
    _priorCellValue = cell.value.toString();
    activeCellController.text = _priorCellValue;
    keyboardFocus.unfocus();
    controller.selection.setActive(row, col);
  }

  // ---------------------------------------------------------------------------
  // Shortcut action helpers
  // ---------------------------------------------------------------------------

  void _copyCell() {
    final row = controller.selection.contextRow;
    final col = controller.selection.contextCol;
    if (row < 0 || col < 0) return;
    _internalClipboard = controller.cellAt(row, col).value;
  }

  void _cutCell() {
    final row = controller.selection.contextRow;
    final col = controller.selection.contextCol;
    if (row < 0 || col < 0) return;
    _internalClipboard = controller.cellAt(row, col).value;
    controller.clearCell(row, col);
  }

  void _pasteCell() {
    final row = controller.selection.contextRow;
    final col = controller.selection.contextCol;
    if (row < 0 || col < 0) return;
    if (_internalClipboard == null) return;
    controller.updateCell(row, col, DataCell(_internalClipboard!));
  }

  void _fillDown() {
    final row = controller.selection.contextRow;
    final col = controller.selection.contextCol;
    if (row < 0 || col < 0) return;
    controller.fillDown(row, col);
  }

  void _fillRight() {
    final row = controller.selection.contextRow;
    final col = controller.selection.contextCol;
    if (row < 0 || col < 0) return;
    controller.fillRight(row, col);
  }

  void _jumpToFirst() {
    if (controller.rowCount == 0 || controller.colCount == 0) return;
    controller.selection.goTo(0, 0);
  }

  void _jumpToLast() {
    if (controller.rowCount == 0 || controller.colCount == 0) return;
    controller.selection.goTo(controller.rowCount - 1, controller.colCount - 1);
  }

  void _insertRow() {
    final row = controller.selection.contextRow;
    if (row < 0) return;
    controller.insertRowAt(row);
  }

  void _deleteRow() {
    final row = controller.selection.contextRow;
    if (row < 0) return;
    controller.deleteRowAt(row);
  }

  void _insertColumn() {
    final col = controller.selection.contextCol;
    if (col < 0) return;
    controller.insertColumnAt(col);
  }

  void _deleteColumn() {
    final col = controller.selection.contextCol;
    if (col < 0) return;
    controller.deleteColumnAt(col);
  }

  void _moveUp() {
    if (highlightedRow > 0) {
      controller.selection.setHighlighted(highlightedRow - 1, highlightedCol);
    }
  }

  void _moveDown() {
    if (highlightedRow < controller.rowCount - 1) {
      controller.selection.setHighlighted(highlightedRow + 1, highlightedCol);
    }
  }

  void _moveLeft() {
    if (highlightedCol > 0) {
      controller.selection.setHighlighted(highlightedRow, highlightedCol - 1);
    }
  }

  void _moveRight() {
    if (highlightedCol < controller.colCount - 1) {
      controller.selection.setHighlighted(highlightedRow, highlightedCol + 1);
    }
  }

  // ── Header helpers ────────────────────────────────────────────────────────
  // Column label conversion is in heading/column_label.dart (columnLabel())
  // and heading widgets are in heading/heading_cells.dart.
}
