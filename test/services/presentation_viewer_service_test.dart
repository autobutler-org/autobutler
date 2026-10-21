import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:quark/services/authenticated_service.dart';
import 'package:quark/services/previews_service.dart';
import 'package:quark/utils/error_text.dart';
import 'package:quark_slides/quark_slides.dart';

/// `GET /previews/slides` as the presentation viewer calls it (#3268).
void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  final requests = <http.Request>[];

  void answer(int status, List<int> body) {
    resetSharedHttpClient();
    sharedHttpClientFactory = () => MockClient((request) async {
      requests.add(request);
      return http.Response.bytes(body, status);
    });
  }

  setUp(requests.clear);
  tearDown(() {
    resetSharedHttpClient();
    sharedHttpClientFactory = buildLocalTrustHttpClient;
  });

  test('reads the deck and its warnings', () async {
    answer(
      200,
      utf8.encode(
        jsonEncode({
          'deck': {
            'schemaVersion': QslideCodec.schemaVersion,
            'title': 'Café',
            'slides': [
              {'id': 's1', 'notes': 'Café ß'},
            ],
          },
          'warnings': [
            {'slide': 1, 'message': 'SmartArt not shown'},
          ],
        }),
      ),
    );

    final preview = await PreviewsService.slides(
      'talks/q1.pptx',
      serial: 'usb1',
    );

    final request = requests.single;
    expect(request.method, 'GET');
    expect(request.url.path, '/api/v0/previews/slides');
    expect(request.url.queryParameters, {
      'filePath': 'talks/q1.pptx',
      'serial': 'usb1',
    });
    expect(preview.deck.slides.single.notes, 'Café ß');
    expect(preview.warnings.single, (slide: 1, message: 'SmartArt not shown'));
  });

  test('a body cut off by a server error is refused', () async {
    answer(200, utf8.encode('{"deck":{"schemaVersion":6,"slides":[{"id"'));
    await expectLater(
      PreviewsService.slides('talks/q1.pptx'),
      throwsA(isA<FormatException>()),
    );
  });

  test('a refused preview throws an ApiException', () async {
    answer(400, utf8.encode('{"error":"bad"}'));
    await expectLater(
      PreviewsService.slides('talks/q1.pptx'),
      throwsA(isA<ApiException>().having((e) => e.statusCode, 'status', 400)),
    );
  });
}
