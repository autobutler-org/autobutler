import 'data_row.dart';

class DataTable {
  List<DataRow> rows = [];

  DataTable(this.rows);
  DataTable.unnamed(this.rows);
}
