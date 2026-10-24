import 'package:flutter/foundation.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:quark/utils/local_time_zone.dart';

/// #3732: iOS and Android send the device's IANA zone, asked of the platform
/// over the app's own channel, so the Quark steps a phone's events in it.
void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  const channel = MethodChannel('org.autobutler.quark/time_zone');
  final messenger =
      TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger;

  tearDown(() {
    debugDefaultTargetPlatformOverride = null;
    messenger.setMockMethodCallHandler(channel, null);
  });

  for (final platform in [TargetPlatform.iOS, TargetPlatform.android]) {
    test('${platform.name} sends the zone the platform reports', () async {
      debugDefaultTargetPlatformOverride = platform;
      messenger.setMockMethodCallHandler(
        channel,
        (call) async =>
            call.method == 'localTimeZone' ? 'America/Los_Angeles' : null,
      );

      expect(await localTimeZoneName(), 'America/Los_Angeles');
    });
  }

  test('a platform that cannot say sends no zone', () async {
    debugDefaultTargetPlatformOverride = TargetPlatform.android;
    messenger.setMockMethodCallHandler(
      channel,
      (_) async => throw PlatformException(code: 'unavailable'),
    );

    expect(await localTimeZoneName(), '');
  });

  test('desktop, which has no channel, sends no zone', () async {
    debugDefaultTargetPlatformOverride = TargetPlatform.macOS;

    expect(await localTimeZoneName(), '');
  });
}
