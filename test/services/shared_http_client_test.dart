import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:quark/services/app_settings.dart';
import 'package:quark/services/shared_http_client.dart';

/// Records what the shared client does to the [HttpClient] underneath it.
class _RecordingHttpClient implements HttpClient {
  Duration? timeout;
  bool closed = false;
  bool Function(X509Certificate cert, String host, int port)? trust;

  @override
  Duration? get connectionTimeout => timeout;

  @override
  set connectionTimeout(Duration? value) => timeout = value;

  @override
  set badCertificateCallback(
    bool Function(X509Certificate cert, String host, int port)? callback,
  ) => trust = callback;

  @override
  void close({bool force = false}) => closed = true;

  @override
  dynamic noSuchMethod(Invocation invocation) => null;
}

class _RecordingHttpOverrides extends HttpOverrides {
  final List<_RecordingHttpClient> created = [];

  @override
  HttpClient createHttpClient(SecurityContext? context) {
    final client = _RecordingHttpClient();
    created.add(client);
    return client;
  }
}

/// #1782: every API call used to build and discard its own client, so nothing
/// ever reused a connection. The shared client must hand back one instance
/// per host and only rebuild when the host moves.
void main() {
  final settings = AppSettings.instance;
  late _RecordingHttpOverrides overrides;

  Future<void> clearHosts() async {
    while (settings.hosts.isNotEmpty) {
      await settings.removeHost(settings.hosts.length - 1);
    }
  }

  Future<void> useHost(String address) =>
      settings.addHost(HostEntry(name: address, hostAddress: address));

  Future<T> withOverrides<T>(Future<T> Function() body) =>
      HttpOverrides.runZoned(
        body,
        createHttpClient: overrides.createHttpClient,
      );

  setUp(() async {
    await clearHosts();
    SharedHttpClient.instance.reset();
    overrides = _RecordingHttpOverrides();
  });

  tearDown(() async {
    SharedHttpClient.instance.reset();
    await clearHosts();
  });

  test('hands back the same client for every call on one host', () async {
    await withOverrides(() async {
      await useHost('https://quark.local');

      final first = sharedHttpClient;
      final second = sharedHttpClient;

      expect(identical(first, second), isTrue);
      expect(overrides.created, hasLength(1));
      expect(overrides.created.single.closed, isFalse);
    });
  });

  test('keeps the connect timeout on the underlying client', () async {
    await withOverrides(() async {
      await useHost('https://quark.local');

      sharedHttpClient;

      expect(overrides.created.single.timeout, kConnectTimeout);
    });
  });

  test('rebuilds on a host change and closes the old client', () async {
    await withOverrides(() async {
      await useHost('https://one.local');
      final first = sharedHttpClient;

      await useHost('https://two.local');
      final second = sharedHttpClient;

      expect(identical(first, second), isFalse);
      expect(overrides.created, hasLength(2));
      expect(overrides.created[0].closed, isTrue);
      expect(overrides.created[1].closed, isFalse);
      expect(identical(sharedHttpClient, second), isTrue);
    });
  });

  test('trusts self-signed certificates only for local hosts', () async {
    await withOverrides(() async {
      await useHost('https://192.168.1.20');
      sharedHttpClient;
      expect(overrides.created.last.trust, isNotNull);

      await useHost('https://quark.example.com');
      sharedHttpClient;
      expect(overrides.created.last.trust, isNull);
    });
  });
}
