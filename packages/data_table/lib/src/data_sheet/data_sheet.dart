import 'package:flutter/material.dart' hide DataTable, DataRow, DataCell;
import 'package:flutter/services.dart';

import '../../data_table.dart';
import 'cell/cell.dart';
import 'cell/editable_cell.dart';
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

  const DataSheet(
      {super.key,
      required this.table,
      this.beforeCellValueChanged,
      this.afterCellValueChanged,
      this.controller,
      this.columnFlex});

  DataSheet.unnamed({super.key})
      : table = DataTable([]),
        beforeCellValueChanged = null,
        afterCellValueChanged = null,
        controller = null,
        columnFlex = null;

  @override
  Widget build(BuildContext context) {
    return _DataSheetView(
      table: table,
      controller: controller,
      columnFlex: columnFlex,
      beforeCellValueChanged: beforeCellValueChanged,
      afterCellValueChanged: afterCellValueChanged,
    );
  }
}

class _DataSheetView extends StatefulWidget {
  final DataTable table;
  final DataSheetController? controller;
  final List<int>? columnFlex;
  final bool Function(String, int, int)? beforeCellValueChanged;
  final Function(String, int, int, bool)? afterCellValueChanged;

  const _DataSheetView({
    required this.table,
    this.controller,
    this.columnFlex,
    this.beforeCellValueChanged,
    this.afterCellValueChanged,
  });

  @override
  State<_DataSheetView> createState() => _DataSheetViewState();
}

class _DataSheetViewState extends State<_DataSheetView> {
  late final TextEditingController activeCellController;
  late final FocusNode keyboardFocus;
  late final DataSheetController controller;
  late final bool _ownsController;

  var activeRow = -1;
  var activeCol = -1;
  var highlightedRow = -1;
  var highlightedCol = -1;

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
  }

  @override
  void dispose() {
    activeCellController.dispose();
    keyboardFocus.dispose();
    if (_ownsController) controller.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Focus(
        focusNode: keyboardFocus,
        onKeyEvent: (node, event) {
          if (event is! KeyDownEvent) return KeyEventResult.ignored;
          final key = event.logicalKey;
          switch (key) {
            case LogicalKeyboardKey.arrowUp:
              _moveUp();
              return KeyEventResult.handled;
            case LogicalKeyboardKey.arrowDown:
              _moveDown();
              return KeyEventResult.handled;
            case LogicalKeyboardKey.arrowLeft:
              _moveLeft();
              return KeyEventResult.handled;
            case LogicalKeyboardKey.arrowRight:
              _moveRight();
              return KeyEventResult.handled;
            case LogicalKeyboardKey.enter:
              if (highlightedRow >= 0 && highlightedCol >= 0) {
                _activateCell(controller.cellAt(highlightedRow, highlightedCol),
                    highlightedRow, highlightedCol);
              } else if (activeRow >= 0 && activeCol >= 0) {
                final previousRow = activeRow;
                final previousCol = activeCol;
                _storeCellValue(activeCellController.text,
                    highlightRow: previousRow, highlightCol: previousCol);
                keyboardFocus.requestFocus();
              }
              return KeyEventResult.handled;
            case LogicalKeyboardKey.tab:
              final isShiftPressed = HardwareKeyboard.instance.isShiftPressed;
              if (activeRow >= 0 && activeCol >= 0) {
                final previousRow = activeRow;
                final previousCol = activeCol;
                _storeCellValue(activeCellController.text,
                    highlightRow: previousRow, highlightCol: previousCol);
                keyboardFocus.requestFocus();
              }
              if (isShiftPressed) {
                _moveLeft();
              } else {
                _moveRight();
              }
              return KeyEventResult.handled;
            default:
              if (highlightedRow >= 0 && highlightedCol >= 0) {
                _activateCell(controller.cellAt(highlightedRow, highlightedCol),
                    highlightedRow, highlightedCol);
              }
              return KeyEventResult.ignored;
          }
        },
        child: ListView.builder(
          itemCount: controller.rowCount,
          itemBuilder: (context, r) {
            return ValueListenableBuilder<List<DataCell>>(
              valueListenable: controller.rowNotifier(r),
              builder: (context, rowCells, _) {
                return Row(
                  children: List.generate(rowCells.length, (c) {
                    final isActiveCell = (r == activeRow && c == activeCol);
                    final isHighlightedCell =
                        (r == highlightedRow && c == highlightedCol);

                    final child = isActiveCell
                        ? EditableCell(
                            key: ValueKey('r${r}c$c'),
                            controller: activeCellController,
                            onSubmitted: _storeCellValue,
                            onEditingComplete: () {
                              _storeCellValue(activeCellController.text);
                            },
                            onTapOutside: (_) {
                              _storeCellValue(activeCellController.text);
                            },
                          )
                        : GestureDetector(
                            onTap: () {
                              if (activeRow >= 0 && activeCol >= 0) {
                                _storeCellValue(activeCellController.text);
                              } else {
                                setState(() {
                                  if (highlightedRow == r &&
                                      highlightedCol == c) {
                                    _activateCell(rowCells[c], r, c);
                                  } else {
                                    highlightedRow = r;
                                    highlightedCol = c;
                                    keyboardFocus.requestFocus();
                                  }
                                });
                              }
                            },
                            child: Text(rowCells[c].value),
                          );

                    final flex = (c < controller.columnFlex.length)
                        ? controller.columnFlex[c]
                        : 1;
                    return Expanded(
                      flex: flex,
                      child: Cell(
                        key: ValueKey('r${r}c$c'),
                        isActive: isActiveCell,
                        isHighlighted: isHighlightedCell,
                        cursor: isActiveCell
                            ? SystemMouseCursors.text
                            : SystemMouseCursors.cell,
                        child: child,
                      ),
                    );
                  }),
                );
              },
            );
          },
        ));
  }

  void _storeCellValue(
    String value, {
    int highlightRow = -1,
    int highlightCol = -1,
  }) {
    final changedRow = activeRow;
    final changedCol = activeCol;
    final isChangeAccepted =
        widget.beforeCellValueChanged?.call(value, changedRow, changedCol) ??
            true;
    setState(() {
      if (isChangeAccepted) {
        controller.updateCell(activeRow, activeCol, DataCell(value));
      }
      activeRow = -1;
      activeCol = -1;
      highlightedRow = highlightRow;
      highlightedCol = highlightCol;
    });
    widget.afterCellValueChanged
        ?.call(value, changedRow, changedCol, isChangeAccepted);
  }

  void _activateCell(DataCell cell, int row, int col) {
    setState(() {
      activeCellController.text = cell.value;
      activeRow = row;
      activeCol = col;
      highlightedRow = -1;
      highlightedCol = -1;
      keyboardFocus.unfocus();
    });
  }

  void _moveUp() {
    if (highlightedRow > 0) {
      setState(() {
        highlightedRow--;
      });
    }
  }

  void _moveDown() {
    if (highlightedRow < controller.rowCount - 1) {
      setState(() {
        highlightedRow++;
      });
    }
  }

  void _moveLeft() {
    if (highlightedCol > 0) {
      setState(() {
        highlightedCol--;
      });
    }
  }

  void _moveRight() {
    if (highlightedCol < controller.colCount - 1) {
      setState(() {
        highlightedCol++;
      });
    }
  }
}
