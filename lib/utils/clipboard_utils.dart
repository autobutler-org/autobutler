import 'package:flutter/foundation.dart';
import 'package:web/web.dart' as web;

/// Whether the Clipboard API is available in the current context.
///
/// Returns `true` on native platforms (always available) and on web when
/// running in a secure context (HTTPS or localhost). Returns `false` on
/// plain HTTP web contexts where the browser blocks clipboard access.
bool get isClipboardAvailable {
  if (!kIsWeb) return true;
  return web.window.isSecureContext;
}
