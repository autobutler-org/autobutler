import 'package:data_table/data_sheet.dart';
import 'package:data_table/data_table.dart';
import 'package:flutter/material.dart' hide DataTable, DataRow, DataCell;

/// One tab of a spreadsheet: the control bar above the grid it drives.
class SheetTabView extends StatelessWidget {
  final DataSheetController controller;
  final DataTable table;

  const SheetTabView({
    required this.controller,
    required this.table,
    super.key,
  });

  @override
  Widget build(BuildContext context) {
    return Column(
      children: [
        DataSheetControlBar(controller: controller),
        Expanded(
          child: DataSheet(controller: controller, table: table),
        ),
      ],
    );
  }
}
