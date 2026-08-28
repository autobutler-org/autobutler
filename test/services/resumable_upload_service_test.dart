import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:quark/models/upload_session.dart';
import 'package:quark/services/resumable_upload_service.dart';
import 'package:quark/services/upload_chunk_source.dart';

/// How a chunk response becomes something the upload manager can act on
/// (#1629).
///
/// The chunk source is where the bytes and the wire meet, so faking it is
/// enough to exercise the whole of [ResumableUploadService.putChunk] without a
/// server — and it is the only part of the session client that does not go
/// through the shared authenticated helpers.
void main() {
  final service = ResumableUploadService.instance;

  Future<(ChunkUploadOutcome, _FakeChunkSource)> put(
    http.Response response, {
    int start = 0,
    int end = 8,
    int totalSize = 20,
  }) async {
    final source = _FakeChunkSource(response);
    final outcome = await service.putChunk(
      sessionId: 'abc',
      source: source,
      start: start,
      end: end,
      totalSize: totalSize,
    );
    return (outcome, source);
  }

  test('sends an inclusive Content-Range and raw bytes', () async {
    // The client reasons in half-open ranges; RFC 7233 does not. Converting
    // once, here, is what keeps every caller from having to remember which
    // convention it is holding.
    final (_, source) = await put(
      http.Response('{"sessionId":"abc","offset":8,"complete":false}', 200),
    );

    expect(source.headers?['Content-Range'], 'bytes 0-7/20');
    expect(source.headers?['Content-Type'], 'application/octet-stream');
    expect(source.start, 0);
    expect(source.end, 8);
    expect(source.uri.toString(), contains('/api/v0/files/upload-session/abc'));
  });

  test('a committed chunk reports the new offset', () async {
    final (outcome, _) = await put(
      http.Response('{"sessionId":"abc","offset":8,"complete":false}', 200),
    );

    expect(outcome, isA<ChunkAccepted>());
    expect((outcome as ChunkAccepted).offset, 8);
    expect(outcome.complete, isFalse);
    expect(outcome.path, isNull);
  });

  test('the last chunk reports where the file landed', () async {
    final (outcome, _) = await put(
      http.Response(
        '{"sessionId":"abc","offset":20,"complete":true,'
        '"path":"photos/2024/clip.mp4"}',
        200,
      ),
      start: 16,
      end: 20,
    );

    final accepted = outcome as ChunkAccepted;
    expect(accepted.complete, isTrue);
    expect(accepted.offset, 20);
    expect(accepted.path, 'photos/2024/clip.mp4');
  });

  test('a 404 is a session that is gone, not an error to report', () async {
    final (outcome, _) = await put(
      http.Response('{"error":"no session"}', 404),
    );

    expect(outcome, isA<ChunkSessionGone>());
  });

  test('a 409 takes the offset from the header, not the body', () async {
    // The body of an error is always {"error": "..."} — the committed offset
    // rides on X-Upload-Offset and nowhere else.
    final (outcome, _) = await put(
      http.Response(
        '{"error":"chunk starts at 0 but 8 bytes are committed"}',
        409,
        headers: const {'x-upload-offset': '8'},
      ),
    );

    expect(outcome, isA<ChunkOffsetMismatch>());
    expect((outcome as ChunkOffsetMismatch).offset, 8);
  });

  test('anything else is a rejection carrying the server message', () async {
    final (outcome, _) = await put(
      http.Response('{"error":"body length does not match"}', 400),
    );

    final rejected = outcome as ChunkRejected;
    expect(rejected.statusCode, 400);
    expect(rejected.message, 'body length does not match');
  });

  test('a rejection with no JSON body still says something', () async {
    final (outcome, _) = await put(http.Response('gateway timeout', 504));

    expect((outcome as ChunkRejected).message, 'gateway timeout');
  });

  group('session models', () {
    test('reads the nanosecond-precision expiry the server sends', () {
      final session = UploadSession.fromJson(const {
        'sessionId': 'abc',
        'offset': 0,
        'expiresAt': '2026-08-28T23:09:58.147425Z',
      });

      expect(session.expiresAt, isNotNull);
      expect(session.expiresAt!.isUtc, isTrue);
      expect(session.expiresAt!.year, 2026);
    });

    test('a status describes the file well enough to check a resume', () {
      final status = UploadSessionStatus.fromJson(const {
        'sessionId': 'abc',
        'offset': 8388608,
        'totalSize': 5368709120,
        'fileName': 'clip.mp4',
        'rootDir': 'photos/2024',
        'expiresAt': '2026-08-28T23:09:58.147425Z',
      });

      expect(status.offset, 8388608);
      expect(status.totalSize, 5368709120);
      expect(status.fileName, 'clip.mp4');
      expect(status.rootDir, 'photos/2024');
    });
  });
}

class _FakeChunkSource implements UploadChunkSource {
  _FakeChunkSource(this._response);

  final http.Response _response;

  late Uri uri;
  Map<String, String>? headers;
  int? start;
  int? end;

  @override
  int get size => 20;

  @override
  DateTime? get lastModified => null;

  @override
  Future<http.Response> putRange({
    required Uri uri,
    required Map<String, String> headers,
    required int start,
    required int end,
  }) async {
    this.uri = uri;
    this.headers = headers;
    this.start = start;
    this.end = end;
    return _response;
  }

  @override
  void release() {}
}
