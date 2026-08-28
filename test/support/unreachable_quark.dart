import 'dart:io';

/// Makes every HTTP request in a test fail the way an out-of-reach Quark does:
/// no answer at all.
///
/// The test binding installs its own [HttpOverrides] that answers everything
/// with a 400, which is the opposite of the case #1637 is about — a 400 is a
/// Quark that replied. Real sockets are no good either: `testWidgets` runs in
/// fake async, so a genuine connection refusal never resolves inside
/// `pumpAndSettle`. Failing at `openUrl` gives the same exception with none of
/// the timing.
///
/// `package:http`'s `IOClient` catches the [SocketException] and rethrows it
/// as a `ClientException`, so services see exactly what they see in the field.
///
/// ```dart
/// setUp(() => HttpOverrides.global = UnreachableQuarkHttpOverrides());
/// tearDown(() => HttpOverrides.global = null);
/// ```
class UnreachableQuarkHttpOverrides extends HttpOverrides {
  @override
  HttpClient createHttpClient(SecurityContext? context) =>
      _UnreachableHttpClient();
}

/// Refuses to open a connection and shrugs at everything else — the
/// `noSuchMethod` catch-all covers the configuration setters callers apply
/// before sending, such as `badCertificateCallback`.
class _UnreachableHttpClient implements HttpClient {
  @override
  Future<HttpClientRequest> openUrl(String method, Uri url) => Future.error(
    SocketException(
      'Connection refused',
      osError: const OSError('Connection refused', 61),
      address: null,
      port: url.port,
    ),
  );

  @override
  Future<HttpClientRequest> open(
    String method,
    String host,
    int port,
    String path,
  ) =>
      openUrl(method, Uri(scheme: 'https', host: host, port: port, path: path));

  @override
  dynamic noSuchMethod(Invocation invocation) => null;
}
