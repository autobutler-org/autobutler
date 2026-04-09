import 'data_cell.dart';

class DataRow {
  List<DataCell> cells = [];

  DataRow(this.cells);
  DataRow.unnamed(this.cells);
}
