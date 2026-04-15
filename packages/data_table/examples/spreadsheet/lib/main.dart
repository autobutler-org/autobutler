import 'package:data_table/data_table.dart';
import 'package:data_table/widgets/spreadsheet.dart';
import 'package:flutter/widgets.dart';

void main() {
  final table = DataTable([
    DataRow([DataCell('Alice'), DataCell('30'), DataCell('New York')]),
    DataRow([DataCell('Bob'), DataCell('25'), DataCell('Los Angeles')]),
    DataRow([DataCell('Charlie'), DataCell('35'), DataCell('Chicago')]),
  ]);
  runApp(Spreadsheet(table: table));
}
