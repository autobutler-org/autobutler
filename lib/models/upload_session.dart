/// Models for the resumable upload session API (#1629).
///
/// The server owns the offset; everything here is a reading of what it said,
/// never a client-side tally the two could disagree about.
library;

/// A session the server has just opened.
class UploadSession {
  const UploadSession({
    required this.sessionId,
    required this.offset,
    this.expiresAt,
  });

  factory UploadSession.fromJson(Map<String, dynamic> json) {
    return UploadSession(
      sessionId: json['sessionId'] as String? ?? '',
      offset: (json['offset'] as num?)?.toInt() ?? 0,
      expiresAt: DateTime.tryParse(json['expiresAt'] as String? ?? ''),
    );
  }

  final String sessionId;

  /// Bytes already committed. Zero for a session that was just created.
  final int offset;

  final DateTime? expiresAt;
}

/// What the server says about a session that already exists.
///
/// [totalSize] and [fileName] are what a resume is checked against: a stored
/// record that disagrees with either is describing a different file, and
/// appending to it would corrupt the upload.
class UploadSessionStatus {
  const UploadSessionStatus({
    required this.sessionId,
    required this.offset,
    required this.totalSize,
    required this.fileName,
    required this.rootDir,
    this.expiresAt,
  });

  factory UploadSessionStatus.fromJson(Map<String, dynamic> json) {
    return UploadSessionStatus(
      sessionId: json['sessionId'] as String? ?? '',
      offset: (json['offset'] as num?)?.toInt() ?? 0,
      totalSize: (json['totalSize'] as num?)?.toInt() ?? 0,
      fileName: json['fileName'] as String? ?? '',
      rootDir: json['rootDir'] as String? ?? '',
      expiresAt: DateTime.tryParse(json['expiresAt'] as String? ?? ''),
    );
  }

  final String sessionId;
  final int offset;
  final int totalSize;
  final String fileName;
  final String rootDir;
  final DateTime? expiresAt;
}

/// The result of sending one chunk.
///
/// Typed rather than an exception with a message in it because the caller acts
/// on the difference: a mismatched offset is resynced and the upload carries
/// on, a vanished session means starting the file again, and only the rest is
/// a failure to report.
sealed class ChunkUploadOutcome {
  const ChunkUploadOutcome();
}

/// The chunk was written. [offset] is what the server now holds.
class ChunkAccepted extends ChunkUploadOutcome {
  const ChunkAccepted({
    required this.offset,
    required this.complete,
    this.path,
  });

  factory ChunkAccepted.fromJson(Map<String, dynamic> json) {
    return ChunkAccepted(
      offset: (json['offset'] as num?)?.toInt() ?? 0,
      complete: json['complete'] as bool? ?? false,
      path: json['path'] as String?,
    );
  }

  final int offset;

  /// True on the last chunk, after which the file is in place and the session
  /// is gone.
  final bool complete;

  /// Where the finished file landed. Only set when [complete].
  final String? path;
}

/// The chunk did not start where the server's offset is (409).
///
/// The real offset rides along on the response, so resyncing costs nothing
/// extra — no follow-up `GET` needed.
class ChunkOffsetMismatch extends ChunkUploadOutcome {
  const ChunkOffsetMismatch({required this.offset});

  final int offset;
}

/// The session is unknown, finished, or expired (404).
///
/// Not retryable: sessions live in the server's memory, so a restart drops
/// them and the only cure is a new session from zero.
class ChunkSessionGone extends ChunkUploadOutcome {
  const ChunkSessionGone();
}

/// Anything else — a malformed range, no writable destination, a 500.
class ChunkRejected extends ChunkUploadOutcome {
  const ChunkRejected({required this.statusCode, required this.message});

  final int statusCode;
  final String message;

  @override
  String toString() => 'chunk rejected ($statusCode): $message';
}
