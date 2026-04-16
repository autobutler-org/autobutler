import 'package:data_table/src/data_sheet/data_sheet_controller.dart';
import 'package:data_table/src/models/data_cell.dart';
import 'package:data_table/src/models/data_row.dart';
import 'package:data_table/src/models/data_table.dart';
import 'package:flutter_test/flutter_test.dart';

DataTable _makeTable(List<List<String>> values) {
  return DataTable(
    values.map((row) => DataRow(row.map((v) => DataCell(v)).toList())).toList(),
  );
}

DataSheetController _makeController(List<List<String>> values) {
  return DataSheetController.fromTable(_makeTable(values));
}

void main() {
  group('DataSheetController', () {
    group('fromTable', () {
      test('rowCount and colCount match the table', () {
        final c = _makeController([
          ['a', 'b', 'c'],
          ['d', 'e', 'f'],
        ]);
        expect(c.rowCount, 2);
        expect(c.colCount, 3);
        c.dispose();
      });

      test('cellAt returns the correct value', () {
        final c = _makeController([
          ['x', 'y'],
        ]);
        expect(c.cellAt(0, 0).value, 'x');
        expect(c.cellAt(0, 1).value, 'y');
        c.dispose();
      });

      test('columnFlex defaults to all 1s', () {
        final c = _makeController([
          ['a', 'b', 'c'],
        ]);
        expect(c.columnFlex, [1, 1, 1]);
        c.dispose();
      });

      test('columnFlex is growable (no fixed-length crash)', () {
        final c = _makeController([
          ['a', 'b'],
        ]);
        expect(() => c.columnFlex.add(1), returnsNormally);
        c.dispose();
      });

      test('accepts custom columnFlex', () {
        final table = _makeTable([
          ['a', 'b'],
        ]);
        final c = DataSheetController.fromTable(table, columnFlex: [2, 3]);
        expect(c.columnFlex, [2, 3]);
        c.dispose();
      });
    });

    group('updateCell', () {
      test('updates cell value', () {
        final c = _makeController([
          ['old', 'b'],
        ]);
        c.updateCell(0, 0, DataCell('new'));
        expect(c.cellAt(0, 0).value, 'new');
        c.dispose();
      });

      test('notifies listeners', () {
        final c = _makeController([
          ['a'],
        ]);
        var notified = false;
        c.addListener(() => notified = true);
        c.updateCell(0, 0, DataCell('b'));
        expect(notified, true);
        c.dispose();
      });
    });

    group('clearCell', () {
      test('clears a single cell value', () {
        final c = _makeController([
          ['hello', 'world'],
        ]);
        c.clearCell(0, 0);
        expect(c.cellAt(0, 0).value, '');
        expect(c.cellAt(0, 1).value, 'world');
        c.dispose();
      });

      test('does nothing for out-of-bounds indices', () {
        final c = _makeController([
          ['a'],
        ]);
        expect(() => c.clearCell(5, 5), returnsNormally);
        c.dispose();
      });
    });

    group('clearRow', () {
      test('clears all cells in the row', () {
        final c = _makeController([
          ['a', 'b', 'c'],
          ['d', 'e', 'f'],
        ]);
        c.clearRow(0);
        expect(c.cellAt(0, 0).value, '');
        expect(c.cellAt(0, 1).value, '');
        expect(c.cellAt(0, 2).value, '');
        expect(c.cellAt(1, 0).value, 'd');
        c.dispose();
      });
    });

    group('clearColumn', () {
      test('clears all cells in the column', () {
        final c = _makeController([
          ['a', 'b'],
          ['c', 'd'],
        ]);
        c.clearColumn(1);
        expect(c.cellAt(0, 1).value, '');
        expect(c.cellAt(1, 1).value, '');
        expect(c.cellAt(0, 0).value, 'a');
        c.dispose();
      });
    });

    group('addRow', () {
      test('appends an empty row', () {
        final c = _makeController([
          ['a', 'b'],
        ]);
        c.addRow();
        expect(c.rowCount, 2);
        expect(c.cellAt(1, 0).value, '');
        expect(c.cellAt(1, 1).value, '');
        c.dispose();
      });
    });

    group('insertRowAt', () {
      test('inserts at the given index', () {
        final c = _makeController([
          ['a'],
          ['c'],
        ]);
        c.insertRowAt(1, cells: [DataCell('b')]);
        expect(c.rowCount, 3);
        expect(c.cellAt(1, 0).value, 'b');
        expect(c.cellAt(2, 0).value, 'c');
        c.dispose();
      });

      test('clamps to end when index exceeds rowCount', () {
        final c = _makeController([
          ['a'],
        ]);
        c.insertRowAt(99);
        expect(c.rowCount, 2);
        c.dispose();
      });
    });

    group('deleteRowAt', () {
      test('removes the row at index', () {
        final c = _makeController([
          ['a'],
          ['b'],
          ['c'],
        ]);
        c.deleteRowAt(1);
        expect(c.rowCount, 2);
        expect(c.cellAt(0, 0).value, 'a');
        expect(c.cellAt(1, 0).value, 'c');
        c.dispose();
      });

      test('does nothing for out-of-bounds index', () {
        final c = _makeController([
          ['a'],
        ]);
        expect(() => c.deleteRowAt(5), returnsNormally);
        expect(c.rowCount, 1);
        c.dispose();
      });
    });

    group('duplicateRow', () {
      test('inserts copy immediately after source', () {
        final c = _makeController([
          ['x', 'y'],
          ['z', 'w'],
        ]);
        c.duplicateRow(0);
        expect(c.rowCount, 3);
        expect(c.cellAt(1, 0).value, 'x');
        expect(c.cellAt(1, 1).value, 'y');
        expect(c.cellAt(2, 0).value, 'z');
        c.dispose();
      });
    });

    group('addColumn', () {
      test('appends an empty column to all rows', () {
        final c = _makeController([
          ['a', 'b'],
          ['c', 'd'],
        ]);
        c.addColumn();
        expect(c.colCount, 3);
        expect(c.cellAt(0, 2).value, '');
        expect(c.cellAt(1, 2).value, '');
        c.dispose();
      });

      test('grows columnFlex', () {
        final c = _makeController([
          ['a'],
        ]);
        c.addColumn();
        expect(c.columnFlex.length, 2);
        c.dispose();
      });
    });

    group('deleteColumnAt', () {
      test('removes the column at index', () {
        final c = _makeController([
          ['a', 'b', 'c'],
        ]);
        c.deleteColumnAt(1);
        expect(c.colCount, 2);
        expect(c.cellAt(0, 0).value, 'a');
        expect(c.cellAt(0, 1).value, 'c');
        c.dispose();
      });
    });

    group('duplicateColumn', () {
      test('inserts copy immediately after source', () {
        final c = _makeController([
          ['x', 'y'],
        ]);
        c.duplicateColumn(0);
        expect(c.colCount, 3);
        expect(c.cellAt(0, 1).value, 'x');
        expect(c.cellAt(0, 2).value, 'y');
        c.dispose();
      });
    });

    group('sortByColumn', () {
      test('sorts lexicographically ascending', () {
        final c = _makeController([
          ['banana'],
          ['apple'],
          ['cherry'],
        ]);
        c.sortByColumn(0);
        expect(c.cellAt(0, 0).value, 'apple');
        expect(c.cellAt(1, 0).value, 'banana');
        expect(c.cellAt(2, 0).value, 'cherry');
        c.dispose();
      });

      test('sorts descending', () {
        final c = _makeController([
          ['1'],
          ['3'],
          ['2'],
        ]);
        c.sortByColumn(0, ascending: false);
        expect(c.cellAt(0, 0).value, '3');
        expect(c.cellAt(1, 0).value, '2');
        expect(c.cellAt(2, 0).value, '1');
        c.dispose();
      });

      test('sorts numerically when values are numbers', () {
        final c = _makeController([
          ['10'],
          ['9'],
          ['100'],
        ]);
        c.sortByColumn(0);
        expect(c.cellAt(0, 0).value, '9');
        expect(c.cellAt(1, 0).value, '10');
        expect(c.cellAt(2, 0).value, '100');
        c.dispose();
      });
    });

    group('removeDuplicateRows', () {
      test('removes exact duplicates, keeping first occurrence', () {
        final c = _makeController([
          ['a', 'b'],
          ['c', 'd'],
          ['a', 'b'],
        ]);
        c.removeDuplicateRows();
        expect(c.rowCount, 2);
        expect(c.cellAt(0, 0).value, 'a');
        expect(c.cellAt(1, 0).value, 'c');
        c.dispose();
      });

      test('does nothing when there are no duplicates', () {
        final c = _makeController([
          ['a'],
          ['b'],
        ]);
        c.removeDuplicateRows();
        expect(c.rowCount, 2);
        c.dispose();
      });
    });

    group('fillDown', () {
      test('fills source value to all rows below', () {
        final c = _makeController([
          ['src', 'x'],
          ['', 'y'],
          ['', 'z'],
        ]);
        c.fillDown(0, 0);
        expect(c.cellAt(1, 0).value, 'src');
        expect(c.cellAt(2, 0).value, 'src');
        expect(c.cellAt(0, 1).value, 'x');
        c.dispose();
      });
    });

    group('fillRight', () {
      test('fills source value to all columns to the right', () {
        final c = _makeController([
          ['src', '', ''],
        ]);
        c.fillRight(0, 0);
        expect(c.cellAt(0, 1).value, 'src');
        expect(c.cellAt(0, 2).value, 'src');
        c.dispose();
      });
    });

    group('findCells', () {
      test('returns matching cell positions', () {
        final c = _makeController([
          ['hello', 'world'],
          ['foo', 'hello'],
        ]);
        final results = c.findCells('hello');
        expect(results.length, 2);
        expect(results[0], (row: 0, col: 0));
        expect(results[1], (row: 1, col: 1));
        c.dispose();
      });

      test('is case-insensitive by default', () {
        final c = _makeController([
          ['Hello'],
        ]);
        expect(c.findCells('hello').length, 1);
        c.dispose();
      });

      test('is case-sensitive when requested', () {
        final c = _makeController([
          ['Hello'],
        ]);
        expect(c.findCells('hello', caseSensitive: true).length, 0);
        c.dispose();
      });

      test('returns empty list when no match', () {
        final c = _makeController([
          ['abc'],
        ]);
        expect(c.findCells('xyz'), isEmpty);
        c.dispose();
      });
    });

    group('replaceCells', () {
      test('replaces all occurrences and returns count', () {
        final c = _makeController([
          ['foo', 'foobar'],
          ['baz', 'foo'],
        ]);
        final count = c.replaceCells('foo', 'qux');
        expect(count, 3);
        expect(c.cellAt(0, 0).value, 'qux');
        expect(c.cellAt(0, 1).value, 'quxbar');
        expect(c.cellAt(1, 1).value, 'qux');
        c.dispose();
      });

      test('returns 0 and does not modify when from is empty', () {
        final c = _makeController([
          ['abc'],
        ]);
        final count = c.replaceCells('', 'x');
        expect(count, 0);
        expect(c.cellAt(0, 0).value, 'abc');
        c.dispose();
      });
    });

    group('exportCsv / loadFromCsv', () {
      test('exportCsv produces correct output', () {
        final c = _makeController([
          ['a', 'b'],
          ['c', 'd'],
        ]);
        expect(c.exportCsv(), 'a,b\nc,d');
        c.dispose();
      });

      test('exportCsv quotes cells containing commas', () {
        final c = _makeController([
          ['hello, world'],
        ]);
        expect(c.exportCsv(), '"hello, world"');
        c.dispose();
      });

      test('exportCsv quotes cells containing double-quotes', () {
        final c = _makeController([
          ['say "hi"'],
        ]);
        expect(c.exportCsv(), '"say ""hi"""');
        c.dispose();
      });

      test('loadFromCsv round-trips with exportCsv', () {
        final c = _makeController([
          ['x', 'y'],
          ['1', '2'],
        ]);
        final csv = c.exportCsv();
        c.loadFromCsv(csv);
        expect(c.rowCount, 2);
        expect(c.cellAt(0, 0).value, 'x');
        expect(c.cellAt(1, 1).value, '2');
        c.dispose();
      });
    });

    group('undo / redo', () {
      test('canUndo is false initially', () {
        final c = _makeController([
          ['a'],
        ]);
        expect(c.canUndo, false);
        c.dispose();
      });

      test('canUndo becomes true after a mutating operation', () {
        final c = _makeController([
          ['a'],
        ]);
        c.clearCell(0, 0);
        expect(c.canUndo, true);
        c.dispose();
      });

      test('undo reverses last change', () {
        final c = _makeController([
          ['original'],
        ]);
        c.clearCell(0, 0);
        c.undo();
        expect(c.cellAt(0, 0).value, 'original');
        c.dispose();
      });

      test('redo reapplies undone change', () {
        final c = _makeController([
          ['original'],
        ]);
        c.clearCell(0, 0);
        c.undo();
        c.redo();
        expect(c.cellAt(0, 0).value, '');
        c.dispose();
      });

      test('redo stack is cleared after a new mutation', () {
        final c = _makeController([
          ['a'],
        ]);
        c.clearCell(0, 0);
        c.undo();
        c.clearCell(0, 0);
        expect(c.canRedo, false);
        c.dispose();
      });
    });
  });
}
