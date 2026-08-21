import 'package:quark_icons/quark_icons.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('QuarkIcons', () {
    test('all icon codepoints are unique', () {
      final all = [
        QuarkIcons.insert_row_above,
        QuarkIcons.insert_row_below,
        QuarkIcons.delete_row,
        QuarkIcons.duplicate_row,
        QuarkIcons.clear_row,
        QuarkIcons.insert_column_left,
        QuarkIcons.insert_column_right,
        QuarkIcons.delete_column,
        QuarkIcons.duplicate_column,
        QuarkIcons.clear_column,
      ];
      final codepoints = all.map((i) => i.codePoint).toSet();
      expect(codepoints.length, equals(all.length));
    });

    test('all icons use the QuarkIcons font family', () {
      const icons = [
        QuarkIcons.insert_row_above,
        QuarkIcons.insert_column_left,
        QuarkIcons.delete_row,
      ];
      for (final icon in icons) {
        expect(icon.fontFamily, equals('QuarkIcons'));
        expect(icon.fontPackage, equals('quark_icons'));
      }
    });
  });
}
