import 'package:data_table/src/models/data_cell.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('DataCell', () {
    test('stores value', () {
      final cell = DataCell('hello');
      expect(cell.value, 'hello');
    });

    test('unnamed constructor stores value', () {
      final cell = DataCell.unnamed('world');
      expect(cell.value, 'world');
    });

    test('value is mutable', () {
      final cell = DataCell('before');
      cell.value = 'after';
      expect(cell.value, 'after');
    });

    group('toJson', () {
      test('returns the value string', () {
        expect(DataCell('abc').toJson(), 'abc');
      });

      test('returns empty string for empty cell', () {
        expect(DataCell('').toJson(), '');
      });
    });

    group('fromJson', () {
      test('parses a string value', () {
        expect(DataCell.fromJson('foo').value, 'foo');
      });

      test('converts non-string to string', () {
        expect(DataCell.fromJson(42).value, '42');
      });

      test('returns empty string for null', () {
        expect(DataCell.fromJson(null).value, '');
      });
    });
  });
}
