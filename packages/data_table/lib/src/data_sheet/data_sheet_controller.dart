import 'package:flutter/material.dart' show ChangeNotifier, ValueNotifier;

import '../../data_table.dart';

class DataSheetController extends ChangeNotifier {
  final DataTable table;
  final List<ValueNotifier<List<DataCell>>> _rows;
  List<int> columnFlex;

  DataSheetController._(this.table, this._rows, this.columnFlex);

  factory DataSheetController.fromTable(DataTable table,
      {List<int>? columnFlex}) {
    final rows = table.rows
        .map((r) => ValueNotifier<List<DataCell>>(List<DataCell>.from(r.cells)))
        .toList();
    final colCount = table.rows.isNotEmpty ? table.rows.first.cells.length : 0;
    final flex = columnFlex ?? List<int>.filled(colCount, 1);
    return DataSheetController._(table, rows, flex);
  }

  int get rowCount => _rows.length;

  int get colCount => _rows.isNotEmpty ? _rows[0].value.length : 0;

  ValueNotifier<List<DataCell>> rowNotifier(int row) => _rows[row];

  DataCell cellAt(int row, int col) => _rows[row].value[col];

  void updateCell(int row, int col, DataCell newCell) {
    table.rows[row].cells[col] = newCell;
    final updated = List<DataCell>.from(_rows[row].value);
    updated[col] = newCell;
    _rows[row].value = updated;
    _rows[row].notifyListeners();
    notifyListeners();
  }

  /// Replace the per-column flex factors and notify listeners so
  /// layouts rebuild using the new values.
  void setColumnFlex(List<int> flex) {
    columnFlex = List<int>.from(flex);
    notifyListeners();
  }

  /// Update a single column's flex factor.
  void updateColumnFlexAt(int index, int flex) {
    if (index < 0 || index >= columnFlex.length) return;
    columnFlex[index] = flex;
    notifyListeners();
  }

  /// Add an empty row at the end of the table.
  void addRow() {
    final cols = colCount > 0 ? colCount : 1;
    final newCells = List<DataCell>.generate(cols, (_) => DataCell(''));
    final newRow = DataRow(newCells);
    table.rows.add(newRow);
    _rows.add(ValueNotifier<List<DataCell>>(List<DataCell>.from(newCells)));
    notifyListeners();
  }

  /// Add an empty column to every row.
  void addColumn() {
    if (_rows.isEmpty) {
      final newCell = DataCell('');
      table.rows.add(DataRow([newCell]));
      _rows.add(ValueNotifier<List<DataCell>>([newCell]));
      notifyListeners();
      return;
    }
    for (var i = 0; i < _rows.length; i++) {
      table.rows[i].cells.add(DataCell(''));
      final updated = List<DataCell>.from(_rows[i].value);
      updated.add(DataCell(''));
      _rows[i].value = updated;
      _rows[i].notifyListeners();
    }
    notifyListeners();
  }

  @override
  void dispose() {
    for (final r in _rows) {
      r.dispose();
    }
    super.dispose();
  }
}
