import 'package:data_table/data_table.dart';
import 'package:flutter/material.dart' hide DataTable, DataRow, DataCell;

void main() => runApp(const Spreadsheet());

class Spreadsheet extends StatefulWidget {
  const Spreadsheet({super.key});

  @override
  State<Spreadsheet> createState() => _SpreadsheetState();
}

class _SpreadsheetState extends State<Spreadsheet> {
  late DataTable table;
  final activeCellController = TextEditingController();
  var activeRow = -1;
  var activeCol = -1;

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
        body: Table(
          children: table.rows
              .asMap()
              .map((r, row) {
                return MapEntry(
                    r,
                    TableRow(
                      children: row.cells
                          .asMap()
                          .map((c, cell) {
                            final widget = Padding(
                                padding: const EdgeInsets.all(8.0),
                                child: (r == activeRow && c == activeCol)
                                    ? TextField(
                                        controller: activeCellController,
                                        decoration: const InputDecoration(
                                          // contentPadding: EdgeInsets.zero,
                                          contentPadding: EdgeInsets.symmetric(
                                              horizontal: 8, vertical: 8),
                                          isDense: true,
                                          border: OutlineInputBorder(),
                                        ),
                                        textAlignVertical:
                                            TextAlignVertical.center,
                                        onSubmitted: _setCell,
                                        autofocus: true,
                                        onEditingComplete: () {
                                          _setCell(activeCellController.text);
                                        },
                                        onTapOutside: (_) {
                                          _setCell(activeCellController.text);
                                        },
                                      )
                                    : GestureDetector(
                                        onTap: () {
                                          final tappedCellRow = r;
                                          final tappedCellCol = c;
                                          setState(() {
                                            activeCellController.text =
                                                cell.data;
                                            activeRow = tappedCellRow;
                                            activeCol = tappedCellCol;
                                          });
                                        },
                                        child: Text(cell.data)));
                            return MapEntry(c, widget);
                          })
                          .values
                          .toList(),
                    ));
              })
              .values
              .toList(),
        ),
      ),
    );
  }

  @override
  void dispose() {
    activeCellController.dispose();
    super.dispose();
  }

  void _setCell(String value) {
    setState(() {
      print('Updating cell at ($activeRow, $activeCol) with value: $value');
      table.rows[activeRow].cells[activeCol] = DataCell(value);
      activeRow = -1;
      activeCol = -1;
    });
  }
}
