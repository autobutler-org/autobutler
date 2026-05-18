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

  group('supported editor helpers', () {
    test('recognize editor-backed file paths and types', () {
      expect(
        hasSupportedCirrusEditorForPath('/Documents/report.abdoc'),
        isTrue,
      );
      expect(
        hasSupportedCirrusEditorForPath('/Documents/budget.absheet'),
        isTrue,
      );
      expect(hasSupportedCirrusEditorForPath('/Documents/photo.jpg'), isTrue);
      expect(hasSupportedCirrusEditorForPath('/Documents/clip.mp4'), isTrue);
      expect(hasSupportedCirrusEditorForPath('/Documents/notes.txt'), isFalse);

      expect(hasSupportedCirrusEditorForType('abdoc'), isTrue);
      expect(hasSupportedCirrusEditorForType('absheet'), isTrue);
      expect(hasSupportedCirrusEditorForType('image'), isTrue);
      expect(hasSupportedCirrusEditorForType('video'), isTrue);
      expect(hasSupportedCirrusEditorForType('generic'), isFalse);
    });
  });
}
