/// Web stand-in for the pairing store. A browser keeps nothing, and this
/// file does not call `dart:io`.
library;

/// Always empty on web.
Future<String?> readValue(String key) async => null;

/// Does nothing on web.
Future<void> writeValue(String key, String value) async {}

/// Does nothing on web.
Future<void> deleteValue(String key) async {}
