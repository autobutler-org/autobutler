import 'package:autobutler/utils/cirrus_route_path_utils.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('isLikelyFilePath', () {
    test('returns false for empty and folder-like paths', () {
      expect(isLikelyFilePath(''), isFalse);
      expect(isLikelyFilePath('/Documents'), isFalse);
      expect(isLikelyFilePath('/Documents/Projects/'), isFalse);
    });

    test('returns true when the final segment looks like a file', () {
      expect(isLikelyFilePath('/Documents/report.abdoc'), isTrue);
      expect(isLikelyFilePath('/Documents/archive.zip'), isTrue);
    });
  });

  group('cirrusRouteDisplayPath', () {
    test('formats empty and non-empty Cirrus paths', () {
      expect(cirrusRouteDisplayPath(''), '/cirrus');
      expect(
        cirrusRouteDisplayPath('/Documents/report.abdoc'),
        '/cirrus/Documents/report.abdoc',
      );
    });
  });
}
