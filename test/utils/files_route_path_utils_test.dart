import 'package:flutter_test/flutter_test.dart';
import 'package:quark/utils/files_route_path_utils.dart';

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

  group('filesRouteDisplayPath', () {
    test('formats empty and non-empty files paths', () {
      expect(filesRouteDisplayPath(''), '/files');
      expect(
        filesRouteDisplayPath('/Documents/report.abdoc'),
        '/files/Documents/report.abdoc',
      );
    });
  });

  group('supported editor helpers', () {
    test('recognize editor-backed file paths and types', () {
      expect(hasSupportedFilesEditorForPath('/Documents/report.abdoc'), isTrue);
      expect(
        hasSupportedFilesEditorForPath('/Documents/budget.absheet'),
        isTrue,
      );
      expect(hasSupportedFilesEditorForPath('/Documents/photo.jpg'), isFalse);

      expect(hasSupportedFilesEditorForType('abdoc'), isTrue);
      expect(hasSupportedFilesEditorForType('absheet'), isTrue);
      expect(hasSupportedFilesEditorForType('image'), isFalse);
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
