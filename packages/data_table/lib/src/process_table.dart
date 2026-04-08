import 'table_state.dart';

TableResult<TRow> processTable<TRow>(TableState<TRow> state) {
  var rows = List<TRow>.from(state.data);

  // Filtering: simple global string match on `.toString()` of each cell
  if (state.globalFilter != null && state.globalFilter!.isNotEmpty) {
    final q = state.globalFilter!.toLowerCase();
    rows = rows.where((r) {
      return state.columns.any((col) {
        try {
          final v = (col.accessor as dynamic)(r);
          return v != null && v.toString().toLowerCase().contains(q);
        } catch (_) {
          return false;
        }
      });
    }).toList();
  }

  // Sorting: single-column
  if (state.sort != null) {
    final sort = state.sort!;
    final col = state.columns.firstWhere(
      (c) => c.id == sort.columnId,
      orElse: () => null,
    );
    if (col != null) {
      rows.sort((a, b) {
        final av = (col.accessor as dynamic)(a);
        final bv = (col.accessor as dynamic)(b);
        final cmp = _compareValues(av, bv, col.sortFn);
        return sort.descending ? -cmp : cmp;
      });
    }
  }

  final total = rows.length;
  int pageCount = 1;
  if (state.pageSize > 0) {
    pageCount = (total / state.pageSize).ceil();
    final start = state.page * state.pageSize;
    final end = start + state.pageSize;
    if (start < rows.length) {
      rows = rows.sublist(start, end.clamp(0, rows.length));
    } else {
      rows = <TRow>[];
    }
  }

  return TableResult(rows: rows, totalRows: total, pageCount: pageCount);
}

int _compareValues(
  dynamic a,
  dynamic b,
  int Function(dynamic a, dynamic b)? custom,
) {
  if (custom != null) return custom(a, b);
  try {
    if (a == null && b == null) return 0;
    if (a == null) return -1;
    if (b == null) return 1;
    if (a is Comparable && b is Comparable) return a.compareTo(b);
    return a.toString().compareTo(b.toString());
  } catch (_) {
    return 0;
  }
}
