import 'package:flutter_test/flutter_test.dart';
import 'package:quark/utils/upload_session_record.dart';

/// The bookkeeping behind resuming an upload across a page reload (#1629).
///
/// Pure on purpose: identity, encoding and staleness are the parts that decide
/// whether a resumed upload appends to the right file, and none of them should
/// need a browser to be checked.
void main() {
  UploadSessionRecord record({
    String fileKey = 'photos|clip.mp4|100|0',
    String sessionId = 'abc',
    int offset = 0,
    int totalSize = 100,
    String fileName = 'clip.mp4',
    DateTime? createdAt,
  }) {
    return UploadSessionRecord(
      fileKey: fileKey,
      sessionId: sessionId,
      offset: offset,
      totalSize: totalSize,
      fileName: fileName,
      createdAt: createdAt ?? DateTime.utc(2026, 1, 1),
    );
  }

  group('uploadFileIdentity', () {
    test('separates files that differ in any of its parts', () {
      final base = uploadFileIdentity(
        rootDir: 'photos',
        fileName: 'clip.mp4',
        size: 100,
        lastModified: DateTime.utc(2026),
      );

      expect(
        base,
        isNot(
          uploadFileIdentity(
            rootDir: 'videos',
            fileName: 'clip.mp4',
            size: 100,
            lastModified: DateTime.utc(2026),
          ),
        ),
      );
      expect(
        base,
        isNot(
          uploadFileIdentity(
            rootDir: 'photos',
            fileName: 'other.mp4',
            size: 100,
            lastModified: DateTime.utc(2026),
          ),
        ),
      );
      expect(
        base,
        isNot(
          uploadFileIdentity(
            rootDir: 'photos',
            fileName: 'clip.mp4',
            size: 101,
            lastModified: DateTime.utc(2026),
          ),
        ),
      );
      expect(
        base,
        isNot(
          uploadFileIdentity(
            rootDir: 'photos',
            fileName: 'clip.mp4',
            size: 100,
            lastModified: DateTime.utc(2025),
          ),
        ),
      );
    });

    test('is stable for the same file', () {
      String key() => uploadFileIdentity(
        rootDir: '',
        fileName: 'a.bin',
        size: 42,
        lastModified: DateTime.utc(2026, 3, 4),
      );

      expect(key(), key());
    });

    test('tolerates a platform that reports no last-modified time', () {
      expect(
        uploadFileIdentity(rootDir: '', fileName: 'a.bin', size: 42),
        uploadFileIdentity(rootDir: '', fileName: 'a.bin', size: 42),
      );
    });
  });

  group('encode and decode', () {
    test('survives the round trip', () {
      final original = record(offset: 8388608);
      final decoded = UploadSessionRecord.decode(original.encode())!;

      expect(decoded.fileKey, original.fileKey);
      expect(decoded.sessionId, original.sessionId);
      expect(decoded.offset, original.offset);
      expect(decoded.totalSize, original.totalSize);
      expect(decoded.fileName, original.fileName);
      expect(decoded.createdAt, original.createdAt);
    });

    test('refuses anything it did not write', () {
      // Storage is shared with the user, other tabs and older builds of the
      // app; the only safe reading of something unrecognized is "no record".
      expect(UploadSessionRecord.decode(null), isNull);
      expect(UploadSessionRecord.decode(''), isNull);
      expect(UploadSessionRecord.decode('not json'), isNull);
      expect(UploadSessionRecord.decode('[1,2,3]'), isNull);
      expect(UploadSessionRecord.decode('{"sessionId":"abc"}'), isNull);
      expect(
        UploadSessionRecord.decode('{"fileKey":"k","sessionId":""}'),
        isNull,
      );
    });

    test('copyWith moves the offset and leaves the identity alone', () {
      final moved = record(offset: 0).copyWith(offset: 4096);

      expect(moved.offset, 4096);
      expect(moved.sessionId, 'abc');
      expect(moved.createdAt, DateTime.utc(2026, 1, 1));
    });
  });

  group('staleness', () {
    test('ages out past the TTL', () {
      final created = DateTime.utc(2026, 1, 1);
      final subject = record(createdAt: created);

      expect(
        subject.isStale(
          now: created.add(const Duration(hours: 1)),
          ttl: const Duration(hours: 2),
        ),
        isFalse,
      );
      expect(
        subject.isStale(
          now: created.add(const Duration(hours: 3)),
          ttl: const Duration(hours: 2),
        ),
        isTrue,
      );
    });
  });

  group('InMemoryUploadSessionStore', () {
    test('reads back what it wrote, by file key', () {
      final store = InMemoryUploadSessionStore();
      store.write(record(fileKey: 'a', sessionId: 'one'));
      store.write(record(fileKey: 'b', sessionId: 'two'));

      expect(store.read('a')?.sessionId, 'one');
      expect(store.read('b')?.sessionId, 'two');
      expect(store.read('c'), isNull);
    });

    test('remove drops one record and leaves the rest', () {
      final store = InMemoryUploadSessionStore();
      store.write(record(fileKey: 'a'));
      store.write(record(fileKey: 'b'));

      store.remove('a');

      expect(store.read('a'), isNull);
      expect(store.read('b'), isNotNull);
    });

    test('pruning drops the old and keeps the current', () {
      // Nothing else ever comes back for an abandoned record, so a prune is
      // the only thing standing between one interrupted upload and a store
      // that grows forever.
      final now = DateTime.utc(2026, 6, 1);
      final store = InMemoryUploadSessionStore();
      store.write(
        record(
          fileKey: 'old',
          createdAt: now.subtract(const Duration(days: 2)),
        ),
      );
      store.write(
        record(
          fileKey: 'fresh',
          createdAt: now.subtract(const Duration(minutes: 5)),
        ),
      );

      store.pruneStale(now: now, ttl: const Duration(hours: 12));

      expect(store.read('old'), isNull);
      expect(store.read('fresh'), isNotNull);
    });
  });
}
