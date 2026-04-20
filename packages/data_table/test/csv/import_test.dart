// Tests for DataSheetController.loadFromCsv() — covers #1021.
//
// The parser follows RFC-4180:
//   - Fields MAY be enclosed in double-quotes.
//   - A double-quote inside a quoted field is escaped as "".
//   - Quoted fields may contain commas and newlines.
//   - Unquoted fields: content is taken literally up to the next comma or
//     end-of-line (RFC-4180 does NOT strip leading/trailing whitespace).

import 'package:data_table/src/data_sheet/data_sheet_controller.dart';
import 'package:data_table/src/models/data_cell.dart';
import 'package:data_table/src/models/data_row.dart';
import 'package:data_table/src/models/data_table.dart';
import 'package:flutter_test/flutter_test.dart';

/// Load CSV into a fresh controller and return it.
DataSheetController _load(String csv) {
  final table = DataTable([
    DataRow([DataCell('')]),
  ]);
  final c = DataSheetController.fromTable(table);
  c.loadFromCsv(csv);
  return c;
}

/// Read cell value at [row],[col] from controller [c].
String _cell(DataSheetController c, int row, int col) =>
    c.cellAt(row, col).value.toString();

void main() {
  group('loadFromCsv', () {
    // ── Basic structure ────────────────────────────────────────────────────

    test('empty string produces empty table', () {
      final c = _load('');
      expect(c.rowCount, 0);
      c.dispose();
    });

    test('single cell, single row', () {
      final c = _load('hello');
      expect(c.rowCount, 1);
      expect(c.colCount, 1);
      expect(_cell(c, 0, 0), 'hello');
      c.dispose();
    });

    test('multiple columns on one row', () {
      final c = _load('a,b,c');
      expect(c.rowCount, 1);
      expect(c.colCount, 3);
      expect(_cell(c, 0, 0), 'a');
      expect(_cell(c, 0, 1), 'b');
      expect(_cell(c, 0, 2), 'c');
      c.dispose();
    });

    test('multiple rows separated by \\n', () {
      final c = _load('a,b\nc,d');
      expect(c.rowCount, 2);
      expect(_cell(c, 0, 0), 'a');
      expect(_cell(c, 1, 0), 'c');
      c.dispose();
    });

    test('Windows-style \\r\\n line endings are handled', () {
      final c = _load('a,b\r\nc,d');
      expect(c.rowCount, 2);
      expect(_cell(c, 0, 0), 'a');
      expect(_cell(c, 0, 1), 'b');
      expect(_cell(c, 1, 0), 'c');
      expect(_cell(c, 1, 1), 'd');
      c.dispose();
    });

    test('trailing newline does not produce an extra empty row', () {
      final c = _load('a,b\nc,d\n');
      expect(c.rowCount, 2);
      c.dispose();
    });

    // ── Quoted fields ──────────────────────────────────────────────────────

    test('quoted field containing a comma', () {
      final c = _load('"hello, world"');
      expect(c.rowCount, 1);
      expect(c.colCount, 1);
      expect(_cell(c, 0, 0), 'hello, world');
      c.dispose();
    });

    test('quoted field with escaped double-quotes', () {
      final c = _load('"say ""hi"""');
      expect(_cell(c, 0, 0), 'say "hi"');
      c.dispose();
    });

    test('quoted field that is a single double-quote character', () {
      // """" = quoted field containing one "
      final c = _load('""""');
      expect(_cell(c, 0, 0), '"');
      c.dispose();
    });

    test('quoted field containing a newline', () {
      final c = _load('"line1\nline2"');
      expect(c.rowCount, 1);
      expect(_cell(c, 0, 0), 'line1\nline2');
      c.dispose();
    });

    test('empty quoted field "" is an empty string', () {
      final c = _load('a,"",b');
      expect(_cell(c, 0, 0), 'a');
      expect(_cell(c, 0, 1), '');
      expect(_cell(c, 0, 2), 'b');
      c.dispose();
    });

    test('mix of quoted and unquoted fields on the same row', () {
      final c = _load('plain,"has,comma",also plain');
      expect(c.colCount, 3);
      expect(_cell(c, 0, 0), 'plain');
      expect(_cell(c, 0, 1), 'has,comma');
      expect(_cell(c, 0, 2), 'also plain');
      c.dispose();
    });

    // ── Whitespace handling ────────────────────────────────────────────────

    test(
      'leading and trailing whitespace in an unquoted field is preserved',
      () {
        // RFC-4180 does not strip whitespace from unquoted fields.
        final c = _load('  hello  ');
        expect(_cell(c, 0, 0), '  hello  ');
        c.dispose();
      },
    );

    test('field that is only whitespace is preserved', () {
      final c = _load('   ');
      expect(_cell(c, 0, 0), '   ');
      c.dispose();
    });

    // ── Value types (no coercion) ──────────────────────────────────────────

    test(
      'numeric strings are stored as strings — no implicit type coercion',
      () {
        final c = _load('42,3.14,-7');
        expect(_cell(c, 0, 0), '42');
        expect(_cell(c, 0, 1), '3.14');
        expect(_cell(c, 0, 2), '-7');
        c.dispose();
      },
    );

    test('boolean-looking values are stored as strings', () {
      final c = _load('true,false,TRUE,FALSE');
      expect(_cell(c, 0, 0), 'true');
      expect(_cell(c, 0, 1), 'false');
      c.dispose();
    });

    // ── Edge cases ─────────────────────────────────────────────────────────

    test('blank line in the middle of input is skipped', () {
      // The parser treats empty/whitespace-only lines as no-ops.
      final c = _load('a,b\n\nc,d');
      // After skipping the blank line we should have 2 data rows.
      expect(c.rowCount, 2);
      expect(_cell(c, 0, 0), 'a');
      expect(_cell(c, 1, 0), 'c');
      c.dispose();
    });

    test('all-empty cells in a row', () {
      final c = _load(',,');
      expect(c.rowCount, 1);
      expect(c.colCount, 3);
      expect(_cell(c, 0, 0), '');
      expect(_cell(c, 0, 1), '');
      expect(_cell(c, 0, 2), '');
      c.dispose();
    });

    test('header row followed by data rows', () {
      final c = _load('Name,Value\nAlice,100\nBob,"hello, world"');
      expect(c.rowCount, 3);
      expect(_cell(c, 0, 0), 'Name');
      expect(_cell(c, 1, 1), '100');
      expect(_cell(c, 2, 1), 'hello, world');
      c.dispose();
    });

    // ── Round-trip ─────────────────────────────────────────────────────────
    // Import → export → re-import → assert identical cell values.

    void roundTrip(String label, List<List<String>> rows) {
      test('round-trip: $label', () {
        // Build a controller from raw cell values.
        final table = DataTable(
          rows.map((r) => DataRow(r.map(DataCell.new).toList())).toList(),
        );
        final c = DataSheetController.fromTable(table);

        // Export → import → export again.
        final csv1 = c.exportCsv();
        c.loadFromCsv(csv1);
        final csv2 = c.exportCsv();

        expect(csv2, csv1, reason: 'round-trip should be stable');

        // Also verify cell values survived.
        for (var r = 0; r < rows.length; r++) {
          for (var col = 0; col < rows[r].length; col++) {
            expect(
              _cell(c, r, col),
              rows[r][col],
              reason: 'cell ($r,$col) should match original value',
            );
          }
        }

        c.dispose();
      });
    }

    roundTrip('plain values', [
      ['a', 'b', 'c'],
      ['d', 'e', 'f'],
    ]);

    roundTrip('values with commas', [
      ['hello, world', 'foo,bar'],
    ]);

    roundTrip('values with double-quotes', [
      [r'say "hi"', r'"quoted"'],
    ]);

    roundTrip('empty cells', [
      ['', 'a', ''],
      ['b', '', 'c'],
    ]);

    roundTrip('numeric strings', [
      ['1', '2.5', '-3'],
    ]);

    roundTrip('whitespace-only values', [
      ['  ', '\t'],
    ]);
  });
}
