import 'package:flutter/foundation.dart' show ChangeNotifier, ValueNotifier;

import '../../data_table.dart';
import 'data_sheet_selection.dart';

// ---------------------------------------------------------------------------
// Internal snapshot used for undo / redo.
// ---------------------------------------------------------------------------

class _TableSnapshot {
  final List<List<String>> cells; // [row][col] as strings
  final List<int> columnFlex;

  _TableSnapshot(this.cells, this.columnFlex);

  factory _TableSnapshot.capture(DataSheetController c) {
    return _TableSnapshot(
      List.generate(
        c._rows.length,
        (r) => c._rows[r].value.map((cell) => cell.value.toString()).toList(),
      ),
      List<int>.from(c.columnFlex),
    );
  }

  /// Restore this snapshot into [c]. Callers must call [c.notifyListeners]
  /// afterward if needed.
  void restore(DataSheetController c) {
    for (final row in c._rows) {
      row.dispose();
    }
    c._rows.clear();
    c.table.rows.clear();
    for (final rowData in cells) {
      final rowCells = rowData.map((v) => DataCell(v)).toList();
      c.table.rows.add(DataRow(List<DataCell>.from(rowCells)));
      c._rows.add(ValueNotifier<List<DataCell>>(List<DataCell>.from(rowCells)));
    }
    c.columnFlex = List<int>.from(columnFlex);
  }
}

// ---------------------------------------------------------------------------
// Controller
// ---------------------------------------------------------------------------

class DataSheetController extends ChangeNotifier {
  final DataTable table;
  final List<ValueNotifier<List<DataCell>>> _rows;
  List<int> columnFlex;

  /// Selection state shared between the sheet view and the control bar.
  final DataSheetSelectionModel selection = DataSheetSelectionModel();

  final List<_TableSnapshot> _undoStack = [];
  final List<_TableSnapshot> _redoStack = [];

  static const int _maxUndoDepth = 100;

  DataSheetController._(this.table, this._rows, this.columnFlex) {
    selection.addListener(_onSelectionChanged);
  }

  void _onSelectionChanged() => notifyListeners();

  factory DataSheetController.fromTable(DataTable table,
      {List<int>? columnFlex}) {
    final rows = table.rows
        .map((r) => ValueNotifier<List<DataCell>>(List<DataCell>.from(r.cells)))
        .toList();
    final colCount = table.rows.isNotEmpty ? table.rows.first.cells.length : 0;
    final flex = columnFlex != null
        ? List<int>.from(columnFlex)
        : List<int>.filled(colCount, 1, growable: true);
    return DataSheetController._(table, rows, flex);
  }

  // -------------------------------------------------------------------------
  // Read-only accessors
  // -------------------------------------------------------------------------

  int get rowCount => _rows.length;

  int get colCount => _rows.isNotEmpty ? _rows[0].value.length : 0;

  ValueNotifier<List<DataCell>> rowNotifier(int row) => _rows[row];

  DataCell cellAt(int row, int col) => _rows[row].value[col];

  // -------------------------------------------------------------------------
  // Undo / Redo
  // -------------------------------------------------------------------------

  bool get canUndo => _undoStack.isNotEmpty;
  bool get canRedo => _redoStack.isNotEmpty;

  void _pushSnapshot() {
    _undoStack.add(_TableSnapshot.capture(this));
    _redoStack.clear();
    if (_undoStack.length > _maxUndoDepth) _undoStack.removeAt(0);
  }

  void undo() {
    if (_undoStack.isEmpty) return;
    _redoStack.add(_TableSnapshot.capture(this));
    _undoStack.removeLast().restore(this);
    notifyListeners();
  }

  void redo() {
    if (_redoStack.isEmpty) return;
    _undoStack.add(_TableSnapshot.capture(this));
    _redoStack.removeLast().restore(this);
    notifyListeners();
  }

  // -------------------------------------------------------------------------
  // Cell mutation
  // -------------------------------------------------------------------------

  void updateCell(int row, int col, DataCell newCell) {
    table.rows[row].cells[col] = newCell;
    final updated = List<DataCell>.from(_rows[row].value);
    updated[col] = newCell;
    _rows[row].value = updated;
    _rows[row].notifyListeners();
    notifyListeners();
  }

  /// Clear the value of a single cell.
  void clearCell(int row, int col) {
    if (row < 0 || row >= rowCount || col < 0 || col >= colCount) return;
    _pushSnapshot();
    updateCell(row, col, DataCell(''));
  }

  /// Clear all cell values in [rowIndex].
  void clearRow(int rowIndex) {
    if (rowIndex < 0 || rowIndex >= rowCount) return;
    _pushSnapshot();
    final empty = List<DataCell>.generate(colCount, (_) => DataCell(''));
    for (var c = 0; c < colCount; c++) {
      table.rows[rowIndex].cells[c] = empty[c];
    }
    _rows[rowIndex].value = empty;
    notifyListeners();
  }

  /// Clear all cell values in [colIndex].
  void clearColumn(int colIndex) {
    if (colIndex < 0 || colIndex >= colCount) return;
    _pushSnapshot();
    for (var r = 0; r < rowCount; r++) {
      table.rows[r].cells[colIndex] = DataCell('');
      final updated = List<DataCell>.from(_rows[r].value);
      updated[colIndex] = DataCell('');
      _rows[r].value = updated;
    }
    notifyListeners();
  }

  // -------------------------------------------------------------------------
  // Row operations
  // -------------------------------------------------------------------------

  /// Append an empty row at the end.
  void addRow() {
    _pushSnapshot();
    final cols = colCount > 0 ? colCount : 1;
    final cells = List<DataCell>.generate(cols, (_) => DataCell(''));
    table.rows.add(DataRow(List<DataCell>.from(cells)));
    _rows.add(ValueNotifier<List<DataCell>>(List<DataCell>.from(cells)));
    notifyListeners();
  }

  /// Insert an empty row at [index] (0-based). Appends if [index] >= rowCount.
  void insertRowAt(int index, {List<DataCell>? cells}) {
    _pushSnapshot();
    final cols = colCount > 0 ? colCount : 1;
    final newCells =
        cells ?? List<DataCell>.generate(cols, (_) => DataCell(''));
    final clamped = index.clamp(0, _rows.length);
    table.rows.insert(clamped, DataRow(List<DataCell>.from(newCells)));
    _rows.insert(
        clamped, ValueNotifier<List<DataCell>>(List<DataCell>.from(newCells)));
    notifyListeners();
  }

  /// Delete the row at [index].
  void deleteRowAt(int index) {
    if (index < 0 || index >= rowCount) return;
    _pushSnapshot();
    table.rows.removeAt(index);
    _rows[index].dispose();
    _rows.removeAt(index);
    notifyListeners();
  }

  /// Duplicate the row at [index], inserting the copy immediately after.
  void duplicateRow(int index) {
    if (index < 0 || index >= rowCount) return;
    _pushSnapshot();
    final sourceCells =
        _rows[index].value.map((c) => DataCell(c.value.toString())).toList();
    table.rows.insert(index + 1, DataRow(List<DataCell>.from(sourceCells)));
    _rows.insert(index + 1,
        ValueNotifier<List<DataCell>>(List<DataCell>.from(sourceCells)));
    notifyListeners();
  }

  // -------------------------------------------------------------------------
  // Column operations
  // -------------------------------------------------------------------------

  /// Append an empty column to every row.
  void addColumn() {
    _pushSnapshot();
    if (_rows.isEmpty) {
      table.rows.add(DataRow([DataCell('')]));
      _rows.add(ValueNotifier<List<DataCell>>([DataCell('')]));
      notifyListeners();
      return;
    }
    for (var i = 0; i < _rows.length; i++) {
      table.rows[i].cells.add(DataCell(''));
      final updated = List<DataCell>.from(_rows[i].value)..add(DataCell(''));
      _rows[i].value = updated;
    }
    columnFlex.add(1);
    notifyListeners();
  }

  /// Insert an empty column at [index]. Appends if [index] >= colCount.
  void insertColumnAt(int index, {String defaultValue = ''}) {
    if (_rows.isEmpty) return;
    _pushSnapshot();
    final clamped = index.clamp(0, colCount);
    for (var i = 0; i < _rows.length; i++) {
      final newCell = DataCell(defaultValue);
      table.rows[i].cells.insert(clamped, newCell);
      final updated = List<DataCell>.from(_rows[i].value)
        ..insert(clamped, DataCell(defaultValue));
      _rows[i].value = updated;
    }
    if (clamped < columnFlex.length) {
      columnFlex.insert(clamped, 1);
    } else {
      columnFlex.add(1);
    }
    notifyListeners();
  }

  /// Delete the column at [index].
  void deleteColumnAt(int index) {
    if (index < 0 || index >= colCount) return;
    _pushSnapshot();
    for (var i = 0; i < _rows.length; i++) {
      table.rows[i].cells.removeAt(index);
      final updated = List<DataCell>.from(_rows[i].value)..removeAt(index);
      _rows[i].value = updated;
    }
    if (index < columnFlex.length) columnFlex.removeAt(index);
    notifyListeners();
  }

  /// Duplicate the column at [index], inserting the copy immediately after.
  void duplicateColumn(int index) {
    if (index < 0 || index >= colCount) return;
    _pushSnapshot();
    for (var i = 0; i < _rows.length; i++) {
      final newCell = DataCell(_rows[i].value[index].value.toString());
      table.rows[i].cells.insert(index + 1, newCell);
      final updated = List<DataCell>.from(_rows[i].value)
        ..insert(index + 1, DataCell(_rows[i].value[index].value.toString()));
      _rows[i].value = updated;
    }
    final flex = index < columnFlex.length ? columnFlex[index] : 1;
    if (index < columnFlex.length) {
      columnFlex.insert(index + 1, flex);
    } else {
      columnFlex.add(flex);
    }
    notifyListeners();
  }

  // -------------------------------------------------------------------------
  // Column flex configuration
  // -------------------------------------------------------------------------

  void setColumnFlex(List<int> flex) {
    columnFlex = List<int>.from(flex);
    notifyListeners();
  }

  void updateColumnFlexAt(int index, int flex) {
    if (index < 0 || index >= columnFlex.length) return;
    columnFlex[index] = flex;
    notifyListeners();
  }

  // -------------------------------------------------------------------------
  // Fill operations
  // -------------------------------------------------------------------------

  /// Copy the value of cell `(fromRow, col)` to all rows below it in the same
  /// column.
  void fillDown(int fromRow, int col) {
    if (fromRow < 0 || fromRow >= rowCount || col < 0 || col >= colCount) {
      return;
    }
    if (fromRow == rowCount - 1) return;
    _pushSnapshot();
    final sourceValue = _rows[fromRow].value[col].value.toString();
    for (var r = fromRow + 1; r < rowCount; r++) {
      table.rows[r].cells[col] = DataCell(sourceValue);
      final updated = List<DataCell>.from(_rows[r].value);
      updated[col] = DataCell(sourceValue);
      _rows[r].value = updated;
    }
    notifyListeners();
  }

  /// Copy the value of cell `(row, fromCol)` to all columns to the right of it
  /// in the same row.
  void fillRight(int row, int fromCol) {
    if (row < 0 || row >= rowCount || fromCol < 0 || fromCol >= colCount) {
      return;
    }
    if (fromCol == colCount - 1) return;
    _pushSnapshot();
    final sourceValue = _rows[row].value[fromCol].value.toString();
    final updated = List<DataCell>.from(_rows[row].value);
    for (var c = fromCol + 1; c < colCount; c++) {
      table.rows[row].cells[c] = DataCell(sourceValue);
      updated[c] = DataCell(sourceValue);
    }
    _rows[row].value = updated;
    notifyListeners();
  }

  // -------------------------------------------------------------------------
  // Sort / deduplication
  // -------------------------------------------------------------------------

  /// Sort all rows by the values in [col] (0-based). Numeric values are sorted
  /// numerically; anything else is sorted lexicographically.
  void sortByColumn(int col, {bool ascending = true}) {
    if (col < 0 || col >= colCount || rowCount == 0) return;
    _pushSnapshot();
    table.rows.sort((a, b) {
      final av = a.cells[col].value.toString();
      final bv = b.cells[col].value.toString();
      final n1 = num.tryParse(av);
      final n2 = num.tryParse(bv);
      final cmp =
          (n1 != null && n2 != null) ? n1.compareTo(n2) : av.compareTo(bv);
      return ascending ? cmp : -cmp;
    });
    for (var i = 0; i < _rows.length; i++) {
      _rows[i].value = List<DataCell>.from(table.rows[i].cells);
    }
    notifyListeners();
  }

  /// Remove duplicate rows (all cell values must match). First occurrence is
  /// kept.
  void removeDuplicateRows() {
    if (rowCount == 0) return;
    _pushSnapshot();
    final seen = <String>{};
    final toRemove = <int>[];
    for (var i = 0; i < _rows.length; i++) {
      final key = _rows[i].value.map((c) => c.value.toString()).join('\x00');
      if (!seen.add(key)) toRemove.add(i);
    }
    for (final i in toRemove.reversed) {
      table.rows.removeAt(i);
      _rows[i].dispose();
      _rows.removeAt(i);
    }
    notifyListeners();
  }

  // -------------------------------------------------------------------------
  // Find / Replace
  // -------------------------------------------------------------------------

  /// Return all `(row, col)` pairs whose cell value contains [query].
  List<({int row, int col})> findCells(String query,
      {bool caseSensitive = false}) {
    final results = <({int row, int col})>[];
    final q = caseSensitive ? query : query.toLowerCase();
    for (var r = 0; r < _rows.length; r++) {
      for (var c = 0; c < _rows[r].value.length; c++) {
        final v = caseSensitive
            ? _rows[r].value[c].value.toString()
            : _rows[r].value[c].value.toString().toLowerCase();
        if (v.contains(q)) results.add((row: r, col: c));
      }
    }
    return results;
  }

  /// Replace all occurrences of [from] with [to] in every cell. Returns the
  /// number of cells that were changed.
  int replaceCells(String from, String to, {bool caseSensitive = false}) {
    if (from.isEmpty) return 0;
    _pushSnapshot();
    var count = 0;
    for (var r = 0; r < _rows.length; r++) {
      final updated = List<DataCell>.from(_rows[r].value);
      var rowChanged = false;
      for (var c = 0; c < updated.length; c++) {
        final v = updated[c].value.toString();
        final newV = caseSensitive
            ? v.replaceAll(from, to)
            : v.replaceAllMapped(
                RegExp(RegExp.escape(from), caseSensitive: false), (_) => to);
        if (newV != v) {
          updated[c] = DataCell(newV);
          table.rows[r].cells[c] = DataCell(newV);
          count++;
          rowChanged = true;
        }
      }
      if (rowChanged) _rows[r].value = updated;
    }
    if (count > 0) notifyListeners();
    return count;
  }

  // -------------------------------------------------------------------------
  // CSV export / import
  // -------------------------------------------------------------------------

  /// Return the table as a RFC-4180-compliant CSV string.
  String exportCsv() {
    return _rows.map((row) {
      return row.value.map((cell) {
        final v = cell.value.toString();
        if (v.contains(',') || v.contains('"') || v.contains('\n')) {
          return '"${v.replaceAll('"', '""')}"';
        }
        return v;
      }).join(',');
    }).join('\n');
  }

  /// Replace the entire table with rows parsed from [csv].
  ///
  /// Follows RFC-4180:
  /// - Fields may be enclosed in double-quotes.
  /// - A double-quote inside a quoted field is escaped as "".
  /// - Quoted fields may span multiple lines.
  /// - Both LF and CRLF are accepted as record separators.
  /// - Completely empty or whitespace-only *unquoted* lines are skipped.
  void loadFromCsv(String csv) {
    _pushSnapshot();
    for (final r in _rows) {
      r.dispose();
    }
    _rows.clear();
    table.rows.clear();

    final parsedRows = _parseCsv(csv);
    for (final rowValues in parsedRows) {
      if (rowValues.isEmpty) continue;
      final cells = rowValues.map(DataCell.new).toList();
      table.rows.add(DataRow(List<DataCell>.from(cells)));
      _rows.add(ValueNotifier<List<DataCell>>(List<DataCell>.from(cells)));
    }
    final newColCount = _rows.isNotEmpty ? _rows[0].value.length : 0;
    columnFlex = List<int>.filled(newColCount, 1);
    notifyListeners();
  }

  /// Full RFC-4180 CSV parser.
  ///
  /// Handles:
  ///  - Quoted fields (may contain commas, newlines, escaped quotes)
  ///  - Both LF and CRLF record separators
  ///  - Trailing commas (empty last field)
  ///  - Blank / whitespace-only lines are skipped
  static List<List<String>> _parseCsv(String csv) {
    final rows = <List<String>>[];
    var fields = <String>[];
    var i = 0;
    final n = csv.length;

    while (i <= n) {
      if (i == n) {
        // End of input: flush the current row.
        // If the last character was a comma, there is an implicit trailing
        // empty field (e.g. "a,b," has three fields, the last being empty).
        if (n > 0 && csv[n - 1] == ',') fields.add('');
        // Skip only a truly empty row (no fields accumulated at all).
        if (fields.isNotEmpty) rows.add(fields);
        break;
      }

      final ch = csv[i];

      if (ch == '"') {
        // Quoted field — may contain commas and newlines.
        i++;
        final buf = StringBuffer();
        while (i < n) {
          final c = csv[i];
          if (c == '"') {
            if (i + 1 < n && csv[i + 1] == '"') {
              // Escaped double-quote.
              buf.write('"');
              i += 2;
            } else {
              // Closing quote.
              i++;
              break;
            }
          } else {
            buf.write(c);
            i++;
          }
        }
        fields.add(buf.toString());
        // Consume comma or record separator after the closing quote.
        if (i < n && csv[i] == ',') {
          i++;
        } else if (i < n && csv[i] == '\r' && i + 1 < n && csv[i + 1] == '\n') {
          rows.add(fields);
          fields = [];
          i += 2;
        } else if (i < n && csv[i] == '\n') {
          rows.add(fields);
          fields = [];
          i++;
        }
      } else if (ch == '\r' && i + 1 < n && csv[i + 1] == '\n') {
        // CRLF record separator.
        // Only add a trailing empty field if the last char before CRLF was a
        // comma (i.e. the row ended with a delimiter, meaning a trailing empty
        // field was intended).
        if (i > 0 && csv[i - 1] == ',') fields.add('');
        final row = fields;
        fields = [];
        // Skip completely blank lines: no fields, or a single field that is
        // empty (a bare newline). Whitespace-only cells are kept.
        final isBlankLine = row.isEmpty || (row.length == 1 && row[0].isEmpty);
        if (!isBlankLine) rows.add(row);
        i += 2;
      } else if (ch == '\n') {
        // LF record separator.
        if (i > 0 && csv[i - 1] == ',') fields.add('');
        final row = fields;
        fields = [];
        final isBlankLine = row.isEmpty || (row.length == 1 && row[0].isEmpty);
        if (!isBlankLine) rows.add(row);
        i++;
      } else if (ch == ',') {
        // Field separator — the current (unquoted) field ended.
        fields.add('');
        i++;
      } else {
        // Unquoted field: read up to the next comma, LF, CRLF, or EOF.
        final buf = StringBuffer();
        while (i < n &&
            csv[i] != ',' &&
            csv[i] != '\n' &&
            !(csv[i] == '\r' && i + 1 < n && csv[i + 1] == '\n')) {
          buf.write(csv[i++]);
        }
        fields.add(buf.toString());
        // If we stopped at a comma, consume it and continue.
        if (i < n && csv[i] == ',') i++;
      }
    }

    return rows;
  }

  // -------------------------------------------------------------------------
  // Dispose
  // -------------------------------------------------------------------------

  @override
  void dispose() {
    selection.removeListener(_onSelectionChanged);
    selection.dispose();
    for (final r in _rows) {
      r.dispose();
    }
    super.dispose();
  }
}
