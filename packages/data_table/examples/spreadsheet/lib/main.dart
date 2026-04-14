import 'package:data_table/data_table.dart';
import 'package:flutter/material.dart' hide DataTable, DataRow, DataCell;
import 'package:flutter/services.dart';

void main() => runApp(const Spreadsheet());

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
            child: KeyboardListener(
                focusNode: keyboardFocus,
                onKeyEvent: (value) {
                  if (value is! KeyDownEvent) return;
                  print(value);
                  switch (value.logicalKey.keyLabel) {
                    case 'Arrow Up':
                      if (highlightedRow > 0) {
                        setState(() {
                          highlightedRow--;
                        });
                      }
                      break;
                    case 'Arrow Down':
                      if (highlightedRow < table.rows.length - 1) {
                        setState(() {
                          highlightedRow++;
                        });
                      }
                      break;
                    case 'Arrow Left':
                      if (highlightedCol > 0) {
                        setState(() {
                          highlightedCol--;
                        });
                      }
                      break;
                    case 'Tab':
                      final isShiftPressed =
                          HardwareKeyboard.instance.isShiftPressed;
                      if (isShiftPressed) {
                        if (highlightedCol > 0) {
                          setState(() {
                            highlightedCol--;
                          });
                        }
                      } else {
                        if (highlightedCol < table.rows[0].cells.length - 1) {
                          setState(() {
                            highlightedCol++;
                          });
                        }
                      }
                    case 'Arrow Right':
                      if (highlightedCol < table.rows[0].cells.length - 1) {
                        setState(() {
                          highlightedCol++;
                        });
                      }
                      break;
                    case 'Enter':
                      if (highlightedRow >= 0 && highlightedCol >= 0) {
                        _activateCell(
                            table.rows[highlightedRow].cells[highlightedCol],
                            highlightedRow,
                            highlightedCol);
                      }
                      break;
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

  void _storeCellValue(String value) {
    setState(() {
      table.rows[activeRow].cells[activeCol] = DataCell(value);
      activeRow = -1;
      activeCol = -1;
      highlightedRow = -1;
      highlightedCol = -1;
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
}
