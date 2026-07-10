import 'dart:convert';

import 'package:autobutler/services/app_settings.dart';
import 'package:autobutler/services/authenticated_service.dart';
import 'package:flutter/foundation.dart';

class SettingsService with AuthenticatedService {
  static final SettingsService _instance = SettingsService._();
  SettingsService._();
  static SettingsService get instance => _instance;

  static Uri get _apiBaseUri {
    final configured = AppSettings.instance.activeHost;
    final base =
        configured ??
        String.fromEnvironment(
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

  static Future<bool> getAutoUpdate() async {
    final uri = _apiBaseUri.resolve('/api/v0/settings');
    final response = await instance.authenticatedGet(uri);
    if (response.statusCode < 200 || response.statusCode >= 300) {
      throw Exception('Failed to fetch settings (${response.statusCode})');
    }
    final decoded = jsonDecode(response.body);
    if (decoded is! Map<String, dynamic>) {
      throw const FormatException('Invalid settings response format');
    }
    final data = decoded['data'];
    final map = data is Map<String, dynamic> ? data : decoded;
    return map['autoUpdate'] as bool? ?? false;
  }

  static Future<void> setAutoUpdate(bool enabled) async {
    final uri = _apiBaseUri.resolve('/api/v0/settings');
    final body = jsonEncode({'autoUpdate': enabled});
    final response = await instance.authenticatedPost(
      uri,
      headers: {'Content-Type': 'application/json'},
      body: body,
    );
    if (response.statusCode < 200 || response.statusCode >= 300) {
      throw Exception('Failed to update settings (${response.statusCode})');
    }
  }
}
