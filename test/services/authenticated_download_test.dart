import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:quark/services/authenticated_service.dart';
import 'package:quark/utils/error_text.dart';

/// A client that answers from the test rather than the network.
class _StubClient extends http.BaseClient {
  _StubClient(this._respond);

  final http.StreamedResponse Function(http.BaseRequest request) _respond;

  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async =>
      _respond(request);
}

class _TestService with AuthenticatedService {
  _TestService(this._client);

  final http.Client _client;

  @override
  http.Client get httpClient => _client;
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  /// The download has to land on disk byte-exact with its headers still
  /// reachable. The body used to arrive whole as `bodyBytes` and get copied a
  /// second time on its way to the file (#1723).
  test('streams the body to a file and keeps the headers', () async {
    final payload = List<int>.generate(512 * 1024, (i) => i % 256);
    final service = _TestService(
      _StubClient(
        (_) => http.StreamedResponse(
          // Delivered in pieces, so a body that never fits one chunk is
          // covered — the whole point of streaming it.
          Stream.fromIterable([
            payload.sublist(0, 100 * 1024),
            payload.sublist(100 * 1024),
          ]),
          200,
          headers: {'content-disposition': 'inline; filename=report.bin'},
        ),
      ),
    );

    final downloaded = await service.authenticatedDownload(
      Uri.parse('https://quark.local/api/v0/files/download'),
    );

    try {
      expect(
        await File(downloaded.path).readAsBytes(),
        equals(payload),
        reason: 'the file on disk must match the response body exactly',
      );
      expect(
        downloaded.headers['content-disposition'],
        contains('report.bin'),
        reason: 'the caller resolves the save name from these headers',
      );
    } finally {
      await downloaded.delete();
    }

    expect(
      await File(downloaded.path).exists(),
      isFalse,
      reason: 'delete() must clean up the temp file',
    );
  });

  /// A failed download must surface as an ApiException rather than a temp file
  /// holding an error body.
  test('throws on a non-success status', () async {
    final service = _TestService(
      _StubClient(
        (_) => http.StreamedResponse(const Stream.empty(), 404),
      ),
    );

    await expectLater(
      service.authenticatedDownload(Uri.parse('https://quark.local/missing')),
      throwsA(isA<ApiException>()),
    );
  });
}
