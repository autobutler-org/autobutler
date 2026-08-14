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
      expect(hasSupportedCirrusEditorForPath('/Documents/photo.jpg'), isFalse);

      expect(hasSupportedCirrusEditorForType('abdoc'), isTrue);
      expect(hasSupportedCirrusEditorForType('absheet'), isTrue);
      expect(hasSupportedCirrusEditorForType('image'), isFalse);
    });
  });

  group('usesGenericFileViewer', () {
    test('covers the document types that had no viewer', () {
      // These reached the "No supported editor" dead end before #1184.
      expect(usesGenericFileViewer('pdf'), isTrue);
      expect(usesGenericFileViewer('docx'), isTrue);
      expect(usesGenericFileViewer('slideshow'), isTrue);
      expect(usesGenericFileViewer('epub'), isTrue);
    });

    test('covers unclassified files', () {
      expect(usesGenericFileViewer('generic'), isTrue);
      expect(usesGenericFileViewer(''), isTrue);
      expect(usesGenericFileViewer('  '), isTrue);
      expect(usesGenericFileViewer('PDF'), isTrue, reason: 'case-insensitive');
    });

    test('leaves types that have a real viewer alone', () {
      for (final type in [
        'abdoc',
        'absheet',
        'image',
        'video',
        'audio',
        'text',
        'archive',
        'folder',
      ]) {
        expect(usesGenericFileViewer(type), isFalse, reason: type);
      }
    });
  });
}
