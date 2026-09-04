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
      expect(isLikelyFilePath('/Documents/report.qdoc'), isTrue);
      expect(isLikelyFilePath('/Documents/archive.zip'), isTrue);
    });
  });

  group('filesRouteDisplayPath', () {
    test('formats empty and non-empty files paths', () {
      expect(filesRouteDisplayPath(''), '/files');
      expect(
        filesRouteDisplayPath('/Documents/report.qdoc'),
        '/files/Documents/report.qdoc',
      );
    });
  });

  group('supported editor helpers', () {
    test('recognize editor-backed file paths and types', () {
      expect(hasSupportedFilesEditorForPath('/Documents/report.qdoc'), isTrue);
      expect(
        hasSupportedFilesEditorForPath('/Documents/budget.qsheet'),
        isTrue,
      );
      expect(hasSupportedFilesEditorForPath('/Documents/photo.jpg'), isFalse);

      expect(hasSupportedFilesEditorForType('qdoc'), isTrue);
      expect(hasSupportedFilesEditorForType('qsheet'), isTrue);
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

    test('covers a raw workbook opened by URL', () {
      // The file browser offers to convert a workbook before it gets here, so
      // this is the deep-link fallback: download and "Open with", not the
      // dead end an unnamed type used to reach (#1741).
      expect(usesGenericFileViewer('xlsx'), isTrue);
    });

    test('covers unclassified files', () {
      expect(usesGenericFileViewer('generic'), isTrue);
      expect(usesGenericFileViewer(''), isTrue);
      expect(usesGenericFileViewer('  '), isTrue);
      expect(usesGenericFileViewer('PDF'), isTrue, reason: 'case-insensitive');
    });

    test('leaves types that have a real viewer alone', () {
      for (final type in [
        'qdoc',
        'qsheet',
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
