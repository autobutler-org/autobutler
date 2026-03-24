import 'package:autobutler/utils/clipboard_utils_stub.dart'
    if (dart.library.js_interop) 'package:autobutler/utils/clipboard_utils_web.dart';

/// Whether the Clipboard API is available in the current context.
///
/// Returns `true` on native platforms (always available) and on web when
/// running in a secure context (HTTPS or localhost). Returns `false` on
/// plain HTTP web contexts where the browser blocks clipboard access.
bool get isClipboardAvailable => isClipboardAvailablePlatform;
