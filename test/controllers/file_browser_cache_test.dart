import 'package:flutter_test/flutter_test.dart';
import 'package:quark/controllers/file_browser_cache.dart';

void main() {
  final cache = FileBrowserCache.instance;

  setUp(cache.clearOpenFile);
  tearDown(cache.clearOpenFile);

  group('open-file tracking', () {
    test('nothing is open by default', () {
      expect(cache.isFileOpen('/notes.txt'), isFalse);
      expect(cache.openFilePath, isNull);
    });

    test('marking a file open then closed leaves nothing open', () {
      cache.markFileOpen('/report.pdf');
      expect(cache.isFileOpen('/report.pdf'), isTrue);

      cache.markFileClosed('/report.pdf');
      expect(cache.isFileOpen('/report.pdf'), isFalse);
      expect(cache.openFilePath, isNull);
    });

    // The whitespace case from #1604 — the reported trigger.
    test('a whitespace name round-trips the same as any other', () {
      cache.markFileOpen('/my doc.qdoc');
      expect(cache.isFileOpen('/my doc.qdoc'), isTrue);

      cache.markFileClosed('/my doc.qdoc');
      expect(cache.isFileOpen('/my doc.qdoc'), isFalse);
      expect(cache.openFilePath, isNull);
    });

    // markFileOpen stored the raw path while the didUpdateWidget guard looked
    // it up normalized, so the two call sites never agreed on the key (#1604).
    test('keys are normalized so callers cannot disagree on the format', () {
      cache.markFileOpen('my doc.qdoc');
      expect(cache.isFileOpen('/my doc.qdoc'), isTrue);
      expect(cache.isFileOpen('my doc.qdoc'), isTrue);
      expect(cache.openFilePath, '/my doc.qdoc');

      cache.markFileClosed('my doc.qdoc');
      expect(cache.isFileOpen('/my doc.qdoc'), isFalse);
    });

    test('trailing slashes and surrounding whitespace normalize alike', () {
      cache.markFileOpen('  /folder/my doc.qdoc  ');
      expect(cache.isFileOpen('/folder/my doc.qdoc'), isTrue);
    });

    test('a different file is not reported open', () {
      cache.markFileOpen('/a.qdoc');
      expect(cache.isFileOpen('/b.qdoc'), isFalse);
    });

    test('closing a different path leaves the marker alone', () {
      cache.markFileOpen('/a.qdoc');
      cache.markFileClosed('/b.qdoc');
      expect(cache.isFileOpen('/a.qdoc'), isTrue);
    });

    test('an empty path is never reported open while nothing is open', () {
      expect(cache.isFileOpen(''), isFalse);
    });
  });
}
