import 'package:flutter/foundation.dart';
import 'package:quark/utils/upload_config.dart';
import 'package:quark/utils/upload_session_record.dart';
import 'package:web/web.dart' as web;

/// Prefix every record shares, so pruning can find them without keeping an
/// index alongside them that could drift out of step.
const String _keyPrefix = 'quark.upload.session.';

/// Session records in `localStorage`, which is what makes a resume across a
/// page reload possible at all.
///
/// Every call is guarded: storage throws rather than returning null when the
/// browser is in private mode or the quota is full, and a failure to remember
/// an upload is a slower upload, not a broken one.
class LocalStorageUploadSessionStore implements UploadSessionStore {
  const LocalStorageUploadSessionStore();

  @override
  UploadSessionRecord? read(String fileKey) {
    try {
      return UploadSessionRecord.decode(
        web.window.localStorage.getItem('$_keyPrefix$fileKey'),
      );
    } catch (e) {
      debugPrint('[upload_session_store_web.dart] Cannot read a record: $e');
      return null;
    }
  }

  @override
  void write(UploadSessionRecord record) {
    try {
      web.window.localStorage.setItem(
        '$_keyPrefix${record.fileKey}',
        record.encode(),
      );
    } catch (e) {
      debugPrint('[upload_session_store_web.dart] Cannot store a record: $e');
    }
  }

  @override
  void remove(String fileKey) {
    try {
      web.window.localStorage.removeItem('$_keyPrefix$fileKey');
    } catch (e) {
      debugPrint('[upload_session_store_web.dart] Cannot drop a record: $e');
    }
  }

  @override
  void pruneStale({
    DateTime? now,
    Duration ttl = UploadConfig.sessionRecordTtl,
  }) {
    try {
      final storage = web.window.localStorage;
      // Collected before removing: removing shifts the indices underneath the
      // walk, and a half-scanned prune leaves records nothing else will visit.
      final stale = <String>[];
      for (var i = 0; i < storage.length; i++) {
        final key = storage.key(i);
        if (key == null || !key.startsWith(_keyPrefix)) {
          continue;
        }
        final record = UploadSessionRecord.decode(storage.getItem(key));
        if (record == null || record.isStale(now: now, ttl: ttl)) {
          stale.add(key);
        }
      }
      for (final key in stale) {
        storage.removeItem(key);
      }
    } catch (e) {
      debugPrint('[upload_session_store_web.dart] Cannot prune records: $e');
    }
  }
}

const UploadSessionStore uploadSessionStorePlatform =
    LocalStorageUploadSessionStore();
