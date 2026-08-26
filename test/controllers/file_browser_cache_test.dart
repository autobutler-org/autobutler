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
      cache.markFileOpen('/my doc.abdoc');
      expect(cache.isFileOpen('/my doc.abdoc'), isTrue);

      cache.markFileClosed('/my doc.abdoc');
      expect(cache.isFileOpen('/my doc.abdoc'), isFalse);
      expect(cache.openFilePath, isNull);
    });

    // markFileOpen stored the raw path while the didUpdateWidget guard looked
    // it up normalized, so the two call sites never agreed on the key (#1604).
    test('keys are normalized so callers cannot disagree on the format', () {
      cache.markFileOpen('my doc.abdoc');
      expect(cache.isFileOpen('/my doc.abdoc'), isTrue);
      expect(cache.isFileOpen('my doc.abdoc'), isTrue);
      expect(cache.openFilePath, '/my doc.abdoc');

      cache.markFileClosed('my doc.abdoc');
      expect(cache.isFileOpen('/my doc.abdoc'), isFalse);
    });

    test('trailing slashes and surrounding whitespace normalize alike', () {
      cache.markFileOpen('  /folder/my doc.abdoc  ');
      expect(cache.isFileOpen('/folder/my doc.abdoc'), isTrue);
    });

    test('a different file is not reported open', () {
      cache.markFileOpen('/a.abdoc');
      expect(cache.isFileOpen('/b.abdoc'), isFalse);
    });

    test('closing a different path leaves the marker alone', () {
      cache.markFileOpen('/a.abdoc');
      cache.markFileClosed('/b.abdoc');
      expect(cache.isFileOpen('/a.abdoc'), isTrue);
    });

    test('an empty path is never reported open while nothing is open', () {
      expect(cache.isFileOpen(''), isFalse);
    });
  });
}
