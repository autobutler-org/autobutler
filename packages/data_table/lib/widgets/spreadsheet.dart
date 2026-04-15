import 'package:flutter/material.dart' hide DataTable, DataRow, DataCell;
import 'package:flutter/services.dart';

import '../data_table.dart';

class Spreadsheet extends StatefulWidget {
  const Spreadsheet({super.key});

  @override
  State<Spreadsheet> createState() => _SpreadsheetState();
}

class _SpreadsheetState extends State<Spreadsheet> {
  late DataTable table;
  final activeCellController = TextEditingController();
  final keyboardFocus = FocusNode();
  var activeRow = -1;
  var activeCol = -1;
  var highlightedRow = -1;
  var highlightedCol = -1;

  @override
  void initState() {
    super.initState();
    table = DataTable([
      DataRow([DataCell('Alice'), DataCell('30'), DataCell('New York')]),
      DataRow([DataCell('Bob'), DataCell('25'), DataCell('Los Angeles')]),
      DataRow([DataCell('Charlie'), DataCell('35'), DataCell('Chicago')]),
    ]);
  }

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      home: Scaffold(
        appBar: AppBar(title: const Text('data_table spreadsheet example')),
        body: Center(
            child: Focus(
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
                        _activateCell(
                            table.rows[highlightedRow].cells[highlightedCol],
                            highlightedRow,
                            highlightedCol);
                      } else if (activeRow >= 0 && activeCol >= 0) {
                        final previousRow = activeRow;
                        final previousCol = activeCol;
                        _storeCellValue(activeCellController.text,
                            highlightRow: previousRow,
                            highlightCol: previousCol);
                        keyboardFocus.requestFocus();
                      }
                      return KeyEventResult.handled;
                    case LogicalKeyboardKey.tab:
                      final isShiftPressed =
                          HardwareKeyboard.instance.isShiftPressed;
                      if (activeRow >= 0 && activeCol >= 0) {
                        final previousRow = activeRow;
                        final previousCol = activeCol;
                        _storeCellValue(activeCellController.text,
                            highlightRow: previousRow,
                            highlightCol: previousCol);
                        keyboardFocus.requestFocus();
                      }
                      if (isShiftPressed) {
                        _moveLeft();
                      } else {
                        _moveRight();
                      }
                      return KeyEventResult
                          .handled; // prevent browser default tabbing
                    default:
                      // If a cell is highlighted and the user starts typing, activate the cell and focus the text field
                      if (highlightedRow >= 0 && highlightedCol >= 0) {
                        _activateCell(
                            table.rows[highlightedRow].cells[highlightedCol],
                            highlightedRow,
                            highlightedCol);
                      }
                      return KeyEventResult.ignored;
                  }
                },
                child: FractionallySizedBox(
                    widthFactor: 0.90,
                    heightFactor: 0.90,
                    child: Table(
                      children: table.rows
                          .asMap()
                          .map((r, row) {
                            return MapEntry(
                                r,
                                TableRow(
                                  children: row.cells
                                      .asMap()
                                      .map((c, cell) {
                                        final isActiveCell =
                                            (r == activeRow && c == activeCol);
                                        final isHighlightedCell =
                                            (r == highlightedRow &&
                                                c == highlightedCol);
                                        const borderWidth = 1.0;
                                        final widget = MouseRegion(
                                            cursor: isActiveCell
                                                ? SystemMouseCursors.text
                                                : SystemMouseCursors.cell,
                                            child: Container(
                                                height:
                                                    40, // pinned height to avoid layout shifts
                                                decoration: BoxDecoration(
                                                  color: isActiveCell
                                                      ? Colors.grey.shade300
                                                      : null,
                                                  border: Border.all(
                                                    color: (isActiveCell ||
                                                            isHighlightedCell)
                                                        ? Theme.of(context)
                                                            .colorScheme
                                                            .primary
                                                        : Colors.grey.shade400,
                                                    width: borderWidth *
                                                        ((isActiveCell ||
                                                                isHighlightedCell)
                                                            ? 2
                                                            : 1),
                                                  ),
                                                  borderRadius:
                                                      BorderRadius.zero,
                                                ),
                                                child: isActiveCell
                                                    ? TextField(
                                                        autofocus: true,
                                                        controller:
                                                            activeCellController,
                                                        decoration:
                                                            const InputDecoration(
                                                          // contentPadding: EdgeInsets.zero,
                                                          contentPadding:
                                                              EdgeInsets
                                                                  .symmetric(
                                                                      horizontal:
                                                                          8,
                                                                      vertical:
                                                                          8),
                                                          isDense: true,
                                                          border:
                                                              InputBorder.none,
                                                        ),
                                                        textAlignVertical:
                                                            TextAlignVertical
                                                                .center,
                                                        onSubmitted:
                                                            _storeCellValue,
                                                        onEditingComplete: () {
                                                          _storeCellValue(
                                                              activeCellController
                                                                  .text);
                                                        },
                                                        onTapOutside: (_) {
                                                          _storeCellValue(
                                                              activeCellController
                                                                  .text);
                                                        },
                                                      )
                                                    : GestureDetector(
                                                        onTap: () {
                                                          if (activeRow >= 0 &&
                                                              activeCol >= 0) {
                                                            _storeCellValue(
                                                                activeCellController
                                                                    .text);
                                                          } else {
                                                            setState(() {
                                                              if (highlightedRow ==
                                                                      r &&
                                                                  highlightedCol ==
                                                                      c) {
                                                                _activateCell(
                                                                    cell, r, c);
                                                              } else {
                                                                highlightedRow =
                                                                    r;
                                                                highlightedCol =
                                                                    c;
                                                                keyboardFocus
                                                                    .requestFocus();
                                                              }
                                                            });
                                                          }
                                                        },
                                                        child:
                                                            Text(cell.data))));
                                        return MapEntry(c, widget);
                                      })
                                      .values
                                      .toList(),
                                ));
                          })
                          .values
                          .toList(),
                    )))),
      ),
    );
  }

  @override
  void dispose() {
    activeCellController.dispose();
    super.dispose();
  }

  void _storeCellValue(
    String value, {
    int highlightRow = -1,
    int highlightCol = -1,
  }) {
    setState(() {
      table.rows[activeRow].cells[activeCol] = DataCell(value);
      activeRow = -1;
      activeCol = -1;
      highlightedRow = highlightRow;
      highlightedCol = highlightCol;
    });
  }

  void _activateCell(DataCell cell, int row, int col) {
    setState(() {
      activeCellController.text = cell.data;
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
    if (highlightedRow < table.rows.length - 1) {
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
    if (highlightedCol < table.rows[0].cells.length - 1) {
      setState(() {
        highlightedCol++;
      });
    }
  }
}
