// ignore_for_file: avoid_print

import 'package:data_table/data_sheet.dart';
import 'package:data_table/data_table.dart';
import 'package:flutter/material.dart' hide DataTable, DataRow, DataCell;

void main() {
  final table = DataTable([
    DataRow([DataCell('Equation'), DataCell('Answer')]),
    DataRow([DataCell('1+2'), DataCell('=1+2')]),
    DataRow([DataCell('2*2'), DataCell('=2*2')]),
  ]);
  final sheetController = DataSheetController.fromTable(table);
  runApp(
    MaterialApp(
      home: Scaffold(
        appBar: AppBar(title: const Text('formulas example')),
        body: Padding(
          padding: const EdgeInsets.all(16),
          child: Column(
            children: [
              DataSheetControlBar(controller: sheetController),
              Expanded(
                child: DataSheet(
                  controller: sheetController,
                  table: table,
                ),
              ),
            ],
          ),
        ),
      ),
    ),
  );
}
