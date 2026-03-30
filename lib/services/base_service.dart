import 'package:autobutler/services/app_settings.dart';
import 'package:flutter/foundation.dart';

mixin ApiBaseUri {
  static Uri get apiBaseUri {
    final configured = AppSettings.instance.activeHost;
    final base =
        configured ??
        const String.fromEnvironment(
          'API_BASE_URL',
          defaultValue: 'http://localhost:8080',
        );
    final uri = Uri.parse(base);
    final isLoopback =
        uri.host == 'localhost' || uri.host == '127.0.0.1' || uri.host == '::1';
    if (!kIsWeb &&
        defaultTargetPlatform == TargetPlatform.android &&
        isLoopback) {
      return uri.replace(host: '10.0.2.2');
    }
    return uri;
  }
}
