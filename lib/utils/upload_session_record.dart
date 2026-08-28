import 'dart:convert';

import 'package:quark/utils/upload_config.dart';

/// The key a stored upload session is filed under.
///
/// Name, size and last-modified, and the directory it was going to. This is a
/// weak identity and known to be: two edits that leave a file the same length
/// within the same millisecond look identical to it, and a file the user
/// rewrote between the interruption and the resume can slip through. The
/// alternative is a checksum, and hashing several gigabytes to decide whether
/// to skip a few of them costs more than it saves — on the web it also means
/// reading the whole file, which is the thing #1629 exists to stop.
///
/// What keeps the weakness from corrupting anything is the check on the far
/// side: a resume asks the server for the session first, and any disagreement
/// about size or name throws the record away and starts over. The identity
/// only decides whether to *ask*.
String uploadFileIdentity({
  required String rootDir,
  required String fileName,
  required int size,
  DateTime? lastModified,
}) {
  final modified = lastModified?.millisecondsSinceEpoch ?? 0;
  return '$rootDir|$fileName|$size|$modified';
}

/// A session worth trying to resume after the page went away.
class UploadSessionRecord {
  const UploadSessionRecord({
    required this.fileKey,
    required this.sessionId,
    required this.offset,
    required this.totalSize,
    required this.fileName,
    required this.createdAt,
  });

  /// Rebuilds a record from [encode], or returns null when the stored string
  /// is not one.
  ///
  /// Tolerant on purpose: this reads storage the user, another tab, or an
  /// older build of the app can have written, and the only safe response to
  /// anything unrecognized is to ignore it and upload from zero.
  static UploadSessionRecord? decode(String? encoded) {
    if (encoded == null || encoded.isEmpty) {
      return null;
    }
    try {
      final decoded = jsonDecode(encoded);
      if (decoded is! Map<String, dynamic>) {
        return null;
      }
      final fileKey = decoded['fileKey'];
      final sessionId = decoded['sessionId'];
      final createdAt = DateTime.tryParse(
        decoded['createdAt'] as String? ?? '',
      );
      if (fileKey is! String ||
          fileKey.isEmpty ||
          sessionId is! String ||
          sessionId.isEmpty ||
          createdAt == null) {
        return null;
      }
      return UploadSessionRecord(
        fileKey: fileKey,
        sessionId: sessionId,
        offset: (decoded['offset'] as num?)?.toInt() ?? 0,
        totalSize: (decoded['totalSize'] as num?)?.toInt() ?? 0,
        fileName: decoded['fileName'] as String? ?? '',
        createdAt: createdAt,
      );
    } catch (_) {
      return null;
    }
  }

  /// See [uploadFileIdentity].
  final String fileKey;

  final String sessionId;

  /// The last offset the server acknowledged. Advisory: a resume trusts the
  /// server's answer over this one.
  final int offset;

  final int totalSize;
  final String fileName;
  final DateTime createdAt;

  UploadSessionRecord copyWith({int? offset}) {
    return UploadSessionRecord(
      fileKey: fileKey,
      sessionId: sessionId,
      offset: offset ?? this.offset,
      totalSize: totalSize,
      fileName: fileName,
      createdAt: createdAt,
    );
  }

  String encode() {
    return jsonEncode({
      'fileKey': fileKey,
      'sessionId': sessionId,
      'offset': offset,
      'totalSize': totalSize,
      'fileName': fileName,
      'createdAt': createdAt.toIso8601String(),
    });
  }

  /// Whether this record is too old to be worth an API call.
  ///
  /// A record from before the TTL is describing a session the server has
  /// almost certainly swept, and asking about it only buys a 404.
  bool isStale({DateTime? now, Duration ttl = UploadConfig.sessionRecordTtl}) {
    final at = now ?? DateTime.now();
    return at.difference(createdAt) > ttl;
  }
}

/// Where resumable session records live between page loads.
///
/// Web-backed by `localStorage`, which is the only place a reload can read
/// from. Native keeps them in memory: nothing there pulls the process out from
/// under a running upload, so there is no reload to survive.
abstract class UploadSessionStore {
  UploadSessionRecord? read(String fileKey);

  void write(UploadSessionRecord record);

  void remove(String fileKey);

  /// Drops every record past the TTL.
  ///
  /// Called before a resume rather than on a timer: an abandoned upload leaves
  /// a record behind and nothing else ever comes back to collect it.
  void pruneStale({
    DateTime? now,
    Duration ttl = UploadConfig.sessionRecordTtl,
  });
}

/// The native store, and a ready-made fake for tests.
class InMemoryUploadSessionStore implements UploadSessionStore {
  final Map<String, UploadSessionRecord> _records = {};

  @override
  UploadSessionRecord? read(String fileKey) => _records[fileKey];

  @override
  void write(UploadSessionRecord record) => _records[record.fileKey] = record;

  @override
  void remove(String fileKey) => _records.remove(fileKey);

  @override
  void pruneStale({
    DateTime? now,
    Duration ttl = UploadConfig.sessionRecordTtl,
  }) {
    _records.removeWhere((_, record) => record.isStale(now: now, ttl: ttl));
  }
}
