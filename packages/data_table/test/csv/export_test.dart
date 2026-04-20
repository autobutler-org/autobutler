// Tests for DataSheetController.exportCsv() — covers #1020.
//
// RFC-4180 rules enforced:
//   - Fields containing commas, double-quotes, or newlines MUST be enclosed
//     in double-quotes.
//   - A double-quote inside a quoted field MUST be escaped as "".
//   - Rows are separated by \n (LF). This implementation uses LF, not CRLF,
//     which is a known deviation from strict RFC-4180 — tests document that
//     choice intentionally.

import 'package:data_table/src/data_sheet/data_sheet_controller.dart';
import 'package:data_table/src/models/data_cell.dart';
import 'package:data_table/src/models/data_row.dart';
import 'package:data_table/src/models/data_table.dart';
import 'package:flutter_test/flutter_test.dart';

DataSheetController _ctrl(List<List<String>> rows) {
  final table = DataTable(
    rows.map((r) => DataRow(r.map(DataCell.new).toList())).toList(),
  );
  return DataSheetController.fromTable(table);
}

void main() {
  group('exportCsv', () {
    // ── Basic structure ────────────────────────────────────────────────────

    test('empty table produces empty string', () {
      final c = _ctrl([]);
      expect(c.exportCsv(), '');
      c.dispose();
    });

    test('single cell, single row', () {
      final c = _ctrl([
        ['hello'],
      ]);
      expect(c.exportCsv(), 'hello');
      c.dispose();
    });

    test('multiple rows are joined by newline', () {
      final c = _ctrl([
        ['a', 'b'],
        ['c', 'd'],
      ]);
      expect(c.exportCsv(), 'a,b\nc,d');
      c.dispose();
    });

    test('all-empty row produces the correct number of commas', () {
      // 3 columns → 2 commas → ",,"
      final c = _ctrl([
        ['', '', ''],
      ]);
      expect(c.exportCsv(), ',,');
      c.dispose();
    });

    // ── Quoting / escaping ─────────────────────────────────────────────────

    test('cell value containing a comma is quoted', () {
      final c = _ctrl([
        ['a,b'],
      ]);
      expect(c.exportCsv(), '"a,b"');
      c.dispose();
    });

    test(
      'cell value containing a double-quote is quoted and the quote escaped',
      () {
        final c = _ctrl([
          [r'say "hi"'],
        ]);
        // RFC-4180: enclose in quotes, escape inner " as ""
        expect(c.exportCsv(), '"say ""hi"""');
        c.dispose();
      },
    );

    test('cell value containing a newline is quoted', () {
      final c = _ctrl([
        ['line1\nline2'],
      ]);
      expect(c.exportCsv(), '"line1\nline2"');
      c.dispose();
    });

    test('cell value containing both a comma and a double-quote', () {
      final c = _ctrl([
        ['a,"b"'],
      ]);
      expect(c.exportCsv(), '"a,""b"""');
      c.dispose();
    });

    test('cell value that is a bare double-quote character', () {
      final c = _ctrl([
        ['"'],
      ]);
      expect(c.exportCsv(), '""""');
      c.dispose();
    });

    test('cell value containing only a comma', () {
      final c = _ctrl([
        [','],
      ]);
      expect(c.exportCsv(), '","');
      c.dispose();
    });

    test(
      'leading and trailing whitespace is preserved without extra quoting',
      () {
        final c = _ctrl([
          ['  hello  '],
        ]);
        expect(c.exportCsv(), '  hello  ');
        c.dispose();
      },
    );

    test('numeric strings are not quoted', () {
      final c = _ctrl([
        ['42', '3.14', '-7'],
      ]);
      expect(c.exportCsv(), '42,3.14,-7');
      c.dispose();
    });

    // ── Mixed rows ─────────────────────────────────────────────────────────

    test('mix of plain and quoted fields on the same row', () {
      final c = _ctrl([
        ['plain', 'has,comma', 'also plain'],
      ]);
      expect(c.exportCsv(), 'plain,"has,comma",also plain');
      c.dispose();
    });

    test('multi-row with some quoted fields', () {
      final c = _ctrl([
        ['Name', 'Value'],
        ['Alice', '100'],
        ['Bob', 'hello, world'],
      ]);
      expect(c.exportCsv(), 'Name,Value\nAlice,100\nBob,"hello, world"');
      c.dispose();
    });

    // ── Round-trip ─────────────────────────────────────────────────────────
    // Export → import → re-export → assert identical output.

    void roundTrip(String label, List<List<String>> rows) {
      test('round-trip: $label', () {
        final c = _ctrl(rows);
        final csv = c.exportCsv();
        c.loadFromCsv(csv);
        final csv2 = c.exportCsv();
        expect(csv2, csv, reason: 'round-trip should produce identical CSV');
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

    roundTrip('values with newlines', [
      ['line1\nline2', 'plain'],
    ]);

    roundTrip('empty cells', [
      ['', 'a', ''],
      ['b', '', 'c'],
    ]);

    roundTrip('all-empty cells', [
      ['', '', ''],
      ['', '', ''],
    ]);

    roundTrip('numeric strings', [
      ['1', '2.5', '-3'],
    ]);
  });
}
