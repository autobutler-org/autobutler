import 'package:web/web.dart' as web;

/// Web — check window.isSecureContext to determine clipboard availability.
bool get isClipboardAvailablePlatform => web.window.isSecureContext;
