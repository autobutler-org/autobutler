import 'dart:convert';

import 'package:autobutler/services/app_settings.dart';
import 'package:autobutler/services/authenticated_service.dart';
import 'package:flutter/foundation.dart';
import 'package:http/http.dart' as http;

class RemoteAccessStatus {
  final bool enabled;
  final String? remoteUrl;

  const RemoteAccessStatus({required this.enabled, this.remoteUrl});

  factory RemoteAccessStatus.fromJson(Map<String, dynamic> json) =>
      RemoteAccessStatus(
        enabled: json['enabled'] as bool? ?? false,
        remoteUrl: json['remoteUrl'] as String?,
      );
}

class RemoteAccessService with AuthenticatedService {
  static final RemoteAccessService instance = RemoteAccessService._();
  RemoteAccessService._();

  static Uri get _apiBaseUri {
    final configured = AppSettings.instance.activeHost;
<<<<<<< HEAD
    final base =
        configured ??
=======
    final base = configured ??
>>>>>>> a943585 (feat(#498): Phase 1 remote access via Tailscale tsnet)
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

  static Future<RemoteAccessStatus> getStatus() async {
    final uri = _apiBaseUri.resolve('/api/v1/settings/remote-access');
    final response = await http.get(uri, headers: _authHeaders);
    if (response.statusCode < 200 || response.statusCode >= 300) {
      throw Exception(
        'Failed to get remote access status (${response.statusCode})',
      );
    }
    final json = jsonDecode(response.body) as Map<String, dynamic>;
    return RemoteAccessStatus.fromJson(json['data'] as Map<String, dynamic>);
  }

<<<<<<< HEAD
  static Future<RemoteAccessStatus> enable() async {
=======
  static Future<RemoteAccessStatus> enable(String authKey) async {
>>>>>>> a943585 (feat(#498): Phase 1 remote access via Tailscale tsnet)
    final uri = _apiBaseUri.resolve('/api/v1/settings/remote-access');
    final response = await http.post(
      uri,
      headers: {'Content-Type': 'application/json', ..._authHeaders},
<<<<<<< HEAD
      body: jsonEncode(<String, dynamic>{}),
    );
    if (response.statusCode < 200 || response.statusCode >= 300) {
      final body = jsonDecode(response.body) as Map<String, dynamic>?;
      final msg =
          body?['error'] as String? ??
=======
      body: jsonEncode({'authKey': authKey}),
    );
    if (response.statusCode < 200 || response.statusCode >= 300) {
      final body = jsonDecode(response.body) as Map<String, dynamic>?;
      final msg = body?['error'] as String? ??
>>>>>>> a943585 (feat(#498): Phase 1 remote access via Tailscale tsnet)
          'Failed to enable remote access (${response.statusCode})';
      throw Exception(msg);
    }
    final json = jsonDecode(response.body) as Map<String, dynamic>;
    return RemoteAccessStatus.fromJson(json['data'] as Map<String, dynamic>);
  }

  static Future<RemoteAccessStatus> disable() async {
    final uri = _apiBaseUri.resolve('/api/v1/settings/remote-access');
    final response = await http.delete(uri, headers: _authHeaders);
    if (response.statusCode < 200 || response.statusCode >= 300) {
      final body = jsonDecode(response.body) as Map<String, dynamic>?;
<<<<<<< HEAD
      final msg =
          body?['error'] as String? ??
=======
      final msg = body?['error'] as String? ??
>>>>>>> a943585 (feat(#498): Phase 1 remote access via Tailscale tsnet)
          'Failed to disable remote access (${response.statusCode})';
      throw Exception(msg);
    }
    final json = jsonDecode(response.body) as Map<String, dynamic>;
    return RemoteAccessStatus.fromJson(json['data'] as Map<String, dynamic>);
  }
}
