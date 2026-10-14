import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:quark/services/app_settings.dart';
import 'package:quark/services/auth_service.dart';
import 'package:quark/services/authenticated_service.dart';
import 'package:quark/services/events_service.dart';
import 'package:shared_preferences/shared_preferences.dart';

import 'events_service_test.dart' show FakeChannel;

/// #3209: every app load asked the Quark for `/auth/status` and the account's
/// settings twice, once from [AuthService.watchAccount] itself and once more
/// when the events socket first opened. Only a reconnect may ask again.
void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  testWidgets('loads the account once per start, again per reconnect', (
    tester,
  ) async {
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(
          const MethodChannel('plugins.it_nomads.com/flutter_secure_storage'),
          (_) async => null,
        );
    SharedPreferences.setMockInitialValues({});
    final settings = AppSettings.instance;
    await settings.load();
    await settings.addHost(
      HostEntry(name: 'Quark', hostAddress: 'https://quark.local'),
    );
    await settings.setSessionToken('a-token');

    var statusCalls = 0;
    final client = MockClient((request) async {
      if (request.url.path == '/api/v0/auth/status') statusCalls++;
      return http.Response('{"setup": true}', 200);
    });
    authHttpClientFactory = () => client;
    sharedHttpClientFactory = () => client;
    resetSharedHttpClient();
    final channels = <FakeChannel>[];
    EventsService.connectChannel = (uri, {headers}) {
      final channel = FakeChannel();
      channels.add(channel);
      return channel;
    };

    AuthService.watchAccount();
    await tester.pump(const Duration(seconds: 1));
    expect(channels, hasLength(1), reason: 'the socket opened');
    expect(statusCalls, 1);

    channels.last.closeFromServer();
    await tester.pump(const Duration(seconds: 2));
    expect(channels, hasLength(2), reason: 'the socket came back');
    expect(statusCalls, 2);

    EventsService.instance.stop();
  });
}
