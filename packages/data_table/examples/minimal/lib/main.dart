import 'package:data_table/data_table.dart';
import 'package:flutter/material.dart' hide DataTable, DataRow, DataCell;

void main() => runApp(const ExampleApp());

class ExampleApp extends StatelessWidget {
  const ExampleApp({super.key});

  @override
  Widget build(BuildContext context) {
    return const MaterialApp(home: ExamplePage());
  }
}

class ExamplePage extends StatelessWidget {
  const ExamplePage({super.key});

  @override
  Widget build(BuildContext context) {
    var table = DataTable([
      DataRow([DataCell('Alice'), DataCell('30'), DataCell('New York')]),
      DataRow([DataCell('Bob'), DataCell('25'), DataCell('Los Angeles')]),
      DataRow([DataCell('Charlie'), DataCell('35'), DataCell('Chicago')]),
    ]);
    return Scaffold(
      appBar: AppBar(title: const Text('data_table minimal example')),
      body: Table(
        children: table.rows.map((row) {
          return TableRow(
            children: row.cells.map((cell) {
              return Padding(
                padding: const EdgeInsets.all(8.0),
                child: Text(cell.value),
              );
            }).toList(),
          );
        }).toList(),
      ),
    );
  }
}
