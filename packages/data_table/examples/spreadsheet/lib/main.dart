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
          children: table.rows.map((row) {
            return TableRow(
              children: row.cells.map((cell) {
                return Padding(
                  padding: const EdgeInsets.all(8.0),
                  child: Text(cell.data),
                );
              }).toList(),
            );
          }).toList(),
        ),
      ),
    );
  }
}
