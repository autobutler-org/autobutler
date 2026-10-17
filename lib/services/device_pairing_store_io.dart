/// Platform secure storage for one phone's remote-access pairing (#2876).
///
/// Values are opaque strings. The facade owns what they mean. A failure to
/// read looks like a missing value; a failure to write is reported as
/// [Errors.keyringUnavailable], the same way a sign-in is.
library;

import 'package:flutter/services.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:quark/utils/error_text.dart';

const FlutterSecureStorage _storage = FlutterSecureStorage();

/// The value stored at [key], or null when it is missing or the store cannot
/// be read.
Future<String?> readValue(String key) async {
  try {
    return await _storage.read(key: key);
  } on PlatformException {
    return null;
  }
}

/// Saves [value] at [key].
Future<void> writeValue(String key, String value) async {
  try {
    await _storage.write(key: key, value: value);
  } on PlatformException {
    throw const MessageException(Errors.keyringUnavailable);
  }
}

/// Removes [key]. A store that cannot be reached leaves the value in place.
Future<void> deleteValue(String key) async {
  try {
    await _storage.delete(key: key);
  } on PlatformException {
    // The next read still sees the old value, and the next write retries.
  }
}
