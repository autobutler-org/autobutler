import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:quark/services/authenticated_service.dart';
import 'package:quark/services/remote_access_service.dart';
import 'package:quark/services/storage_service.dart';
import 'package:quark/utils/error_text.dart';

/// #3725: a refusal whose body is not a JSON object, such as a proxy's HTML
/// error page, still carries its status, so the user reads the sentence for
/// that status rather than the generic one.
void main() {
  void respondWith(int status, String body) {
    resetSharedHttpClient();
    sharedHttpClientFactory = () =>
        MockClient((_) async => http.Response(body, status));
  }

  tearDown(() {
    resetSharedHttpClient();
    sharedHttpClientFactory = buildLocalTrustHttpClient;
  });

  Matcher apiException(int status) =>
      isA<ApiException>().having((e) => e.statusCode, 'statusCode', status);

  test('a 503 HTML page from mount reads as busy', () async {
    respondWith(503, '<html><body>Service Unavailable</body></html>');

    final error = await StorageService.mountDevice(
      'ABC123',
    ).then<Object?>((_) => null, onError: (Object e) => e);

    expect(error, apiException(503));
    expect(
      Errors.message(error, 'mount the drive'),
      'Your Quark is busy. Try again in a moment.',
    );
  });

  test(
    'a 502 HTML page from enabling remote access reads as a problem',
    () async {
      respondWith(502, '<html><body>Bad Gateway</body></html>');

      final error = await RemoteAccessService.enable().then<Object?>(
        (_) => null,
        onError: (Object e) => e,
      );

      expect(error, apiException(502));
      expect(
        Errors.message(error, 'turn on remote access'),
        'Your Quark ran into a problem. Try again.',
      );
    },
  );

  for (final body in ['', '"busy"', '[1, 2]', '{"error": 3}']) {
    test('a body of `$body` still carries the status', () async {
      respondWith(500, body);
      await expectLater(
        StorageService.unmountDevice('ABC123'),
        throwsA(apiException(500)),
      );
    });
  }

  test('a JSON object with an error passes its text through', () async {
    respondWith(409, '{"error": "drive is busy"}');
    await expectLater(
      StorageService.mountDevice('ABC123'),
      throwsA(
        isA<MessageException>().having(
          (e) => e.message,
          'message',
          'drive is busy',
        ),
      ),
    );
  });
}
