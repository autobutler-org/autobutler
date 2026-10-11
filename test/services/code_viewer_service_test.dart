import 'dart:typed_data';

import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:quark/services/authenticated_service.dart';
import 'package:quark/services/code_viewer_service.dart';
import 'package:quark/utils/error_text.dart';
import 'package:shared_preferences/shared_preferences.dart';

/// The code viewer's ranged read (#2991): one slice of the file per request,
/// through the download endpoint's `Range` support.
void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  final requests = <http.BaseRequest>[];

  void respondWith(
    http.StreamedResponse Function(http.BaseRequest request) respond,
  ) {
    resetSharedHttpClient();
    sharedHttpClientFactory = () => MockClient.streaming((request, _) async {
      requests.add(request);
      return respond(request);
    });
  }

  setUp(() {
    SharedPreferences.setMockInitialValues({});
    requests.clear();
  });

  tearDown(() {
    resetSharedHttpClient();
    sharedHttpClientFactory = buildLocalTrustHttpClient;
  });

  test('asks the download endpoint for exactly the range', () async {
    respondWith(
      (_) => http.StreamedResponse(
        Stream.value([1, 2, 3, 4]),
        206,
        headers: {'content-range': 'bytes 8-11/1000'},
      ),
    );

    final range = await CodeViewerService.readRange(
      'logs/server.log',
      start: 8,
      length: 4,
      serial: 'usb1',
    );

    expect(range.bytes, [1, 2, 3, 4]);
    expect(range.totalSize, 1000);
    final request = requests.single;
    expect(request.url.path, '/api/v0/files/download');
    expect(request.url.queryParameters['filePath'], 'logs/server.log');
    expect(request.url.queryParameters['serial'], 'usb1');
    expect(request.headers['Range'], 'bytes=8-11');
  });

  test('stops reading at the length when the range is ignored', () async {
    final whole = Uint8List.fromList(List.generate(100, (i) => i));
    respondWith(
      (_) => http.StreamedResponse(
        Stream.fromIterable([whole.sublist(0, 6), whole.sublist(6)]),
        200,
        contentLength: whole.length,
      ),
    );

    final range = await CodeViewerService.readRange(
      'a.log',
      start: 0,
      length: 4,
    );

    expect(range.bytes, [0, 1, 2, 3]);
    expect(range.totalSize, 100);
  });

  test('a start past the end of the file is no bytes', () async {
    respondWith(
      (_) => http.StreamedResponse(
        const Stream.empty(),
        416,
        headers: {'content-range': 'bytes */10'},
      ),
    );

    final range = await CodeViewerService.readRange(
      'a.log',
      start: 10,
      length: 4,
    );

    expect(range.bytes, isEmpty);
    expect(range.totalSize, 10);
  });

  test('a refused read throws its status', () async {
    respondWith((_) => http.StreamedResponse(const Stream.empty(), 403));

    await expectLater(
      CodeViewerService.readRange('secret.log', start: 0, length: 4),
      throwsA(
        isA<ApiException>().having((e) => e.statusCode, 'statusCode', 403),
      ),
    );
  });
}
