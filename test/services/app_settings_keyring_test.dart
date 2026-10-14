import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:quark/services/app_settings.dart';
import 'package:quark/utils/error_text.dart';
import 'package:shared_preferences/shared_preferences.dart';

/// #3148: on Linux the session token lives in the Secret Service (GNOME
/// Keyring, KWallet). A desktop with none running answers every secure-storage
/// call with a libsecret error. Launch must still reach the login page, and
/// signing in must say why it failed rather than surface a PlatformException.
void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  const secureStorage = MethodChannel(
    'plugins.it_nomads.com/flutter_secure_storage',
  );

  setUpAll(() {
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(secureStorage, (call) async {
          // What flutter_secure_storage_linux sends back when libsecret
          // cannot reach org.freedesktop.secrets.
          throw PlatformException(
            code: 'Libsecret error',
            message: 'The name org.freedesktop.secrets was not provided',
          );
        });
  });

  tearDownAll(() {
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(secureStorage, null);
  });

  final settings = AppSettings.instance;

  Future<void> loadOneHost() async {
    SharedPreferences.setMockInitialValues({
      'hosts': '[{"name":"Home","hostAddress":"https://quark.local"}]',
      'activeHostIndex': 0,
    });
    await settings.load();
  }

  test('load succeeds with no keyring and reports no session', () async {
    await loadOneHost();
    expect(settings.sessionToken, isNull);
  });

  test('storing a session without a keyring says so in plain words', () async {
    await loadOneHost();

    Object? caught;
    try {
      await settings.setSessionToken('token');
    } catch (e) {
      caught = e;
    }

    expect(caught, isA<MessageException>());
    expect(Errors.message(caught, 'sign in'), Errors.keyringUnavailable);
  });

  test('a session that could not be stored is not kept in memory', () async {
    await loadOneHost();

    await expectLater(settings.setSessionToken('token'), throwsException);

    expect(settings.sessionToken, isNull);
    expect(settings.sessionTokenNotifier.value, isNull);
  });
}
