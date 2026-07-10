import 'dart:convert';

import 'package:autobutler/services/app_settings.dart';
import 'package:autobutler/services/authenticated_service.dart';
import 'package:flutter/foundation.dart';
import 'package:http/http.dart' as http;

class SmbStatus {
  const SmbStatus({
    required this.linux,
    required this.installed,
    required this.configured,
    required this.running,
    required this.filesDir,
  });

  factory SmbStatus.fromJson(Map<String, dynamic> json) {
    return SmbStatus(
      linux: json['linux'] as bool? ?? false,
      installed: json['installed'] as bool? ?? false,
      configured: json['configured'] as bool? ?? false,
      running: json['running'] as bool? ?? false,
      filesDir: json['filesDir'] as String? ?? '',
    );
  }

  final bool linux;
  final bool installed;
  final bool configured;
  final bool running;
  final String filesDir;
}

class SmbService with AuthenticatedService {
  static final SmbService instance = SmbService._();
  SmbService._();

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

  static Map<String, String> get _authHeaders => instance.authHeaders;

  static Future<SmbStatus> getStatus() async {
    final uri = _apiBaseUri.resolve('/api/v0/smb/status');
    final response = await http.get(uri, headers: _authHeaders);
    if (response.statusCode < 200 || response.statusCode >= 300) {
      throw Exception('Failed to get SMB status (${response.statusCode})');
    }
    final json = jsonDecode(response.body) as Map<String, dynamic>;
    return SmbStatus.fromJson(json['data'] as Map<String, dynamic>);
  }

  static Future<SmbStatus> setup(String user, String password) async {
    final uri = _apiBaseUri.resolve('/api/v0/smb/setup');
    final response = await http.post(
      uri,
      headers: {'Content-Type': 'application/json', ..._authHeaders},
      body: jsonEncode({'user': user, 'password': password}),
    );
    if (response.statusCode < 200 || response.statusCode >= 300) {
      final body = jsonDecode(response.body) as Map<String, dynamic>?;
      final msg =
          body?['error'] as String? ?? 'Setup failed (${response.statusCode})';
      throw Exception(msg);
    }
    final json = jsonDecode(response.body) as Map<String, dynamic>;
    return SmbStatus.fromJson(json['data'] as Map<String, dynamic>);
  }

  static Future<SmbStatus> teardown() async {
    final uri = _apiBaseUri.resolve('/api/v0/smb');
    final response = await http.delete(uri, headers: _authHeaders);
    if (response.statusCode < 200 || response.statusCode >= 300) {
      final body = jsonDecode(response.body) as Map<String, dynamic>?;
      final msg =
          body?['error'] as String? ??
          'Teardown failed (${response.statusCode})';
      throw Exception(msg);
    }
    final json = jsonDecode(response.body) as Map<String, dynamic>;
    return SmbStatus.fromJson(json['data'] as Map<String, dynamic>);
  }
}
