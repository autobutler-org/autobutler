import 'package:flutter/foundation.dart';
import 'package:web/web.dart' as web;

/// Reads [key] from the tab's `sessionStorage`, or null when it is absent or
/// storage is unavailable. Storage throws in some private modes; a cache that
/// fails costs an unlock prompt, nothing more.
Future<String?> read(String key) async {
  try {
    return web.window.sessionStorage.getItem(key);
  } catch (e) {
    debugPrint('[chat_identity_cache_web.dart] Cannot read: $e');
    return null;
  }
}

/// Writes [key] to the tab's `sessionStorage`, doing nothing when storage is
/// unavailable.
Future<void> write(String key, String value) async {
  try {
    web.window.sessionStorage.setItem(key, value);
  } catch (e) {
    debugPrint('[chat_identity_cache_web.dart] Cannot write: $e');
  }
}

/// Deletes [key] from the tab's `sessionStorage`, doing nothing when storage
/// is unavailable.
Future<void> delete(String key) async {
  try {
    web.window.sessionStorage.removeItem(key);
  } catch (e) {
    debugPrint('[chat_identity_cache_web.dart] Cannot delete: $e');
  }
}
