import 'package:flutter_secure_storage/flutter_secure_storage.dart';

/// The native chat identity cache: the platform keystore.
const _storage = FlutterSecureStorage();

/// Reads [key] from the keystore.
Future<String?> read(String key) => _storage.read(key: key);

/// Writes [key] to the keystore.
Future<void> write(String key, String value) =>
    _storage.write(key: key, value: value);

/// Deletes [key] from the keystore.
Future<void> delete(String key) => _storage.delete(key: key);
