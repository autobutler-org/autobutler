import 'package:flutter/foundation.dart';
import 'package:flutter/services.dart';

/// The app's own channel to the platform's zone (#3732): `MainActivity` on
/// Android and `AppDelegate` on iOS answer `localTimeZone`.
const _channel = MethodChannel('org.autobutler.quark/time_zone');

/// iOS and Android: the zone the platform reports. Desktop has no channel and
/// `dart:io` no zone name, so it sends an empty name, as does a platform that
/// fails to answer.
Future<String> localTimeZoneNamePlatform() async {
  if (defaultTargetPlatform != TargetPlatform.iOS &&
      defaultTargetPlatform != TargetPlatform.android) {
    return '';
  }
  try {
    return await _channel.invokeMethod<String>('localTimeZone') ?? '';
  } on PlatformException {
    return '';
  } on MissingPluginException {
    return '';
  }
}
