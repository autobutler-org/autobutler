import 'dart:convert';

import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:quark/controllers/chat_keys_controller.dart';
import 'package:quark/models/account_keys.dart';
import 'package:quark/models/chat_keys.dart';
import 'package:quark/services/app_settings.dart';
import 'package:quark/services/auth_service.dart';
import 'package:quark/services/authenticated_service.dart';
import 'package:quark/utils/error_text.dart';
import 'package:shared_preferences/shared_preferences.dart';

import '../support/auth_salt.dart';

/// A Quark that answers every request with one canned response, and remembers
/// what it was asked.
class _RecordingClient extends http.BaseClient {
  _RecordingClient({this.status = 204, this.body = ''});

  final int status;
  final String body;
  final requests = <http.Request>[];

  @override
  Future<http.StreamedResponse> send(http.BaseRequest request) async {
    requests.add(request as http.Request);
    return http.StreamedResponse(
      Stream.value(utf8.encode(body)),
      status,
      request: request,
    );
  }
}

/// #3022: a signed-in member changes their own password. #2430: the Quark is
/// sent the two passwords' auth keys, never either password. #2416: the chat
/// keys go in the same request, re-wrapped, so chat history survives.
void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  const secureStorage = MethodChannel(
    'plugins.it_nomads.com/flutter_secure_storage',
  );
  final stored = <String, String>{};

  setUpAll(() {
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(secureStorage, (call) async {
          final key = call.arguments['key'] as String?;
          switch (call.method) {
            case 'read':
              return stored[key];
            case 'write':
              stored[key!] = call.arguments['value'] as String;
              return null;
            case 'delete':
              stored.remove(key);
              return null;
            default:
              return null;
          }
        });
  });

  tearDownAll(() {
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(secureStorage, null);
  });

  /// What the fake re-wrap hands back: recognizable bytes, not real crypto,
  /// which ChatKeysController's own tests cover.
  final rewrapped = WrappedChatKeys(
    publicKeys: ChatPublicKeys(
      boxPublicKey: Uint8List(32),
      signPublicKey: Uint8List(32),
    ),
    byPassword: WrappedSecret(wrapped: Uint8List(104), salt: Uint8List(16)),
    byPhrase: WrappedSecret(wrapped: Uint8List(104), salt: Uint8List(16)),
    kdfParams: KdfParams.standard,
  );
  final rewraps = <String>[];

  /// What the re-wrap answers with: [rewrapped], or null for an account with
  /// no chat keys.
  WrappedChatKeys? chatKeys;

  setUp(() async {
    stored.clear();
    SharedPreferences.setMockInitialValues({
      'hosts': jsonEncode([
        {'name': 'One', 'hostAddress': 'http://one.local'},
      ]),
      'activeHostIndex': 0,
    });
    await AppSettings.instance.load();
    await AppSettings.instance.setSessionToken('session-token');
    await AppSettings.instance.setUsername('grace');
    rewraps.clear();
    chatKeys = rewrapped;
    chatKeysForPasswordChange =
        ({
          required currentPassword,
          required newPassword,
          required authSalt,
        }) async {
          rewraps.add('$currentPassword/$newPassword/$authSalt');
          return chatKeys;
        };
  });

  tearDown(() {
    authHttpClientFactory = () => sharedHttpClient;
    chatKeysForPasswordChange =
        ChatKeysController.instance.keysForPasswordChange;
  });

  Future<void> change() => AuthService.changePassword(
    currentPassword: 'the-old-password',
    newPassword: 'brand-new-password',
  );

  Matcher refusesWith(String message) => throwsA(
    isA<MessageException>().having((e) => e.message, 'message', message),
  );

  test('sends both passwords as keys, with the chat keys re-wrapped, and '
      'keeps the session', () async {
    final client = _RecordingClient();
    final salts = AuthSaltClient(client);
    authHttpClientFactory = () => salts;

    await change();

    final request = client.requests.single;
    expect(request.method, 'PUT');
    expect(request.url.path, '/api/v0/auth/password');
    expect(request.headers['Authorization'], 'Bearer session-token');
    expect(jsonDecode(request.body), {
      'password': await testAuthKey('the-old-password'),
      'newAuthKey': await testAuthKey('brand-new-password'),
      'chatKeys': rewrapped.toJson(),
    });
    // #2430: neither password leaves the app.
    expect(request.body, isNot(contains('the-old-password')));
    expect(request.body, isNot(contains('brand-new-password')));
    expect(salts.asked, everyElement('grace'));
    // #2416: re-wrapped under the salt the new auth key was derived with.
    expect(rewraps, ['the-old-password/brand-new-password/$testAuthSalt']);
    expect(AppSettings.instance.sessionToken, 'session-token');
    expect(AppSettings.instance.username, 'grace');
  });

  test('an account with no chat keys sends none', () async {
    final client = _RecordingClient();
    authHttpClientFactory = () => AuthSaltClient(client);
    chatKeys = null;

    await change();

    final body = jsonDecode(client.requests.single.body) as Map;
    expect(body.keys, unorderedEquals(['password', 'newAuthKey']));
  });

  test('a current password that opens no keys stops before anything is '
      'sent', () async {
    final client = _RecordingClient();
    authHttpClientFactory = () => AuthSaltClient(client);
    chatKeysForPasswordChange =
        ({required currentPassword, required newPassword, required authSalt}) =>
            Future.error(const MessageException(Errors.wrongCurrentPassword));

    await expectLater(change(), refusesWith(Errors.wrongCurrentPassword));
    expect(client.requests, isEmpty);
  });

  test('a current password the Quark refuses reads as wrong, and the '
      'session stays', () async {
    final client = _RecordingClient(
      status: 403,
      body: '{"error":"incorrect password"}',
    );
    authHttpClientFactory = () => AuthSaltClient(client);

    await expectLater(change(), refusesWith(Errors.wrongCurrentPassword));
    expect(AppSettings.instance.sessionToken, 'session-token');
  });

  test(
    'any other refusal carries its status, never the Quark\'s text',
    () async {
      final client = _RecordingClient(
        status: 403,
        body: '{"error":"account disabled"}',
      );
      authHttpClientFactory = () => AuthSaltClient(client);

      await expectLater(
        change(),
        throwsA(isA<ApiException>().having((e) => e.statusCode, 'status', 403)),
      );
    },
  );

  test('a session the Quark no longer honors is an expired session', () async {
    final client = _RecordingClient(
      status: 401,
      body: '{"error":"not authenticated"}',
    );
    authHttpClientFactory = () => AuthSaltClient(client);

    await expectLater(change(), throwsA(isA<UnauthorizedException>()));
  });

  test('too many attempts carries the 429', () async {
    final client = _RecordingClient(
      status: 429,
      body: '{"error":"too many attempts"}',
    );
    authHttpClientFactory = () => AuthSaltClient(client);

    await expectLater(
      change(),
      throwsA(isA<ApiException>().having((e) => e.statusCode, 'status', 429)),
    );
    expect(
      Errors.message(const ApiException(429), 'change your password'),
      'Too many requests. Wait a moment and try again.',
    );
  });

  group('refused before anything is sent:', () {
    late _RecordingClient client;
    late AuthSaltClient salts;

    setUp(() {
      client = _RecordingClient();
      salts = AuthSaltClient(client);
      authHttpClientFactory = () => salts;
    });

    test('no session', () async {
      await AppSettings.instance.setSessionToken(null);

      await expectLater(change(), throwsA(isA<UnauthorizedException>()));
      expect(client.requests, isEmpty);
      expect(salts.asked, isEmpty);
      expect(rewraps, isEmpty);
    });

    test('an account with no auth key yet: sign in again first', () async {
      salts.legacy = true;

      await expectLater(change(), refusesWith(Errors.accountTooOld));
      expect(client.requests, isEmpty);
      expect(rewraps, isEmpty);
    });

    test('a Quark with no salt endpoint: update it', () async {
      salts.status = 404;

      await expectLater(change(), refusesWith(Errors.quarkTooOld));
      expect(client.requests, isEmpty);
    });
  });
}
