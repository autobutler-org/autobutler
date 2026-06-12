import 'package:autobutler_icons/autobutler_icons.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('AutobutlerIcons', () {
    test('all icon codepoints are unique', () {
      final all = [
        AutobutlerIcons.insert_row_above,
        AutobutlerIcons.insert_row_below,
        AutobutlerIcons.delete_row,
        AutobutlerIcons.duplicate_row,
        AutobutlerIcons.clear_row,
        AutobutlerIcons.insert_column_left,
        AutobutlerIcons.insert_column_right,
        AutobutlerIcons.delete_column,
        AutobutlerIcons.duplicate_column,
        AutobutlerIcons.clear_column,
      ];
      final codepoints = all.map((i) => i.codePoint).toSet();
      expect(codepoints.length, equals(all.length));
    });

    test('all icons use the AutobutlerIcons font family', () {
      const icons = [
        AutobutlerIcons.insert_row_above,
        AutobutlerIcons.insert_column_left,
        AutobutlerIcons.delete_row,
      ];
      for (final icon in icons) {
        expect(icon.fontFamily, equals('AutobutlerIcons'));
        expect(icon.fontPackage, equals('autobutler_icons'));
      }
    });
  });
}
