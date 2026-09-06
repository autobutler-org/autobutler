import 'dart:convert';

import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:quark/services/app_settings.dart';
import 'package:quark/services/files_service.dart';
import 'package:shared_preferences/shared_preferences.dart';

/// #1777: the thumbnail disk cache was keyed by URL, and the URL carries the
/// session token, so every re-login threw the whole cache away.
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

  final settings = AppSettings.instance;

  setUp(() async {
    stored.clear();
    SharedPreferences.setMockInitialValues({
      'hosts': jsonEncode([
        {'name': 'One', 'hostAddress': 'http://one.local'},
      ]),
      'activeHostIndex': 0,
    });
    await settings.load();
  });

  test('the token is in the URL but never in the key', () async {
    await settings.setSessionToken('first-token');
    final url = FilesService.constructThumbnailUrl(
      'photos/a.jpg',
      serial: 'S1',
    ).toString();
    final key = FilesService.thumbnailCacheKey('photos/a.jpg', serial: 'S1');

    expect(url, contains('first-token'));
    expect(key, isNot(contains('first-token')));

    await settings.setSessionToken('second-token');
    expect(FilesService.thumbnailCacheKey('photos/a.jpg', serial: 'S1'), key);
  });

  test('path, serial and size each change the key', () {
    final base = FilesService.thumbnailCacheKey(
      'photos/a.jpg',
      serial: 'S1',
      size: 'sm',
    );

    expect(
      FilesService.thumbnailCacheKey('photos/b.jpg', serial: 'S1', size: 'sm'),
      isNot(base),
    );
    expect(
      FilesService.thumbnailCacheKey('photos/a.jpg', serial: 'S2', size: 'sm'),
      isNot(base),
    );
    expect(
      FilesService.thumbnailCacheKey('photos/a.jpg', serial: 'S1'),
      isNot(base),
    );
  });

  test('the key changes with the active host', () async {
    final key = FilesService.thumbnailCacheKey('photos/a.jpg', serial: 'S1');
    SharedPreferences.setMockInitialValues({
      'hosts': jsonEncode([
        {'name': 'Two', 'hostAddress': 'http://two.local'},
      ]),
      'activeHostIndex': 0,
    });
    await settings.load();

    expect(
      FilesService.thumbnailCacheKey('photos/a.jpg', serial: 'S1'),
      isNot(key),
    );
  });

  test('the key normalizes the path and serial the way the URL does', () {
    final key = FilesService.thumbnailCacheKey('photos/a.jpg', serial: 'S1');

    expect(FilesService.thumbnailCacheKey('/photos/a.jpg', serial: 'S1'), key);
    expect(
      FilesService.thumbnailCacheKey(' photos/a.jpg ', serial: ' S1 '),
      key,
    );
  });
}
