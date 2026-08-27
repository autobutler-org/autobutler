import 'dart:convert';

import 'package:quark/services/app_settings.dart';
import 'package:quark/services/authenticated_service.dart';

class SettingsService with AuthenticatedService {
  static final SettingsService _instance = SettingsService._();
  SettingsService._();
  static SettingsService get instance => _instance;

  static Future<bool> getAutoUpdate() async {
    final uri = apiBaseUri.resolve('/api/v0/settings');
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
    final uri = apiBaseUri.resolve('/api/v0/settings');
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
