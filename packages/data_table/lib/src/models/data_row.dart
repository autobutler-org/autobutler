import 'data_cell.dart';

class DataRow {
  List<DataCell> cells = [];

  DataRow(this.cells);
  DataRow.unnamed(this.cells);

  List<String> toJson() => cells.map((c) => c.toJson()).toList();

  static DataRow fromJson(List<dynamic> json) =>
      DataRow(json.map((c) => DataCell.fromJson(c)).toList());
}
