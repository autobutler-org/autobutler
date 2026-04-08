class SortState {
  final String columnId;
  final bool descending;
  const SortState({required this.columnId, this.descending = false});
}

class TableState<TRow> {
  final List<TRow> data;
  final List<dynamic> columns;
  final SortState? sort;
  final String? globalFilter;
  final int page;
  final int pageSize;

  const TableState({
    required this.data,
    required this.columns,
    this.sort,
    this.globalFilter,
    this.page = 0,
    this.pageSize = 0,
  });
}

class TableResult<TRow> {
  final List<TRow> rows;
  final int totalRows;
  final int pageCount;

  const TableResult({
    required this.rows,
    required this.totalRows,
    required this.pageCount,
  });
}
