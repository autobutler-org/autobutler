class ColumnDef<TRow> {
  final String id;
  final String header;
  final dynamic Function(TRow row) accessor;
  final int Function(dynamic a, dynamic b)? sortFn;

  const ColumnDef({
    required this.id,
    required this.header,
    required this.accessor,
    this.sortFn,
  });
}
