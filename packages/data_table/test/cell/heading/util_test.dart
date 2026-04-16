import 'package:data_table/src/data_sheet/cell/heading/util.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('columnLabel', () {
    test('single letters A-Z', () {
      expect(columnLabel(0), 'A');
      expect(columnLabel(1), 'B');
      expect(columnLabel(25), 'Z');
    });

    test('two-letter labels AA-AZ', () {
      expect(columnLabel(26), 'AA');
      expect(columnLabel(27), 'AB');
      expect(columnLabel(51), 'AZ');
    });

    test('two-letter labels BA-ZZ', () {
      expect(columnLabel(52), 'BA');
      expect(columnLabel(701), 'ZZ');
    });

    test('three-letter labels AAA', () {
      expect(columnLabel(702), 'AAA');
      expect(columnLabel(703), 'AAB');
    });
  });
}
