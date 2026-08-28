import 'dart:convert';

import 'package:http/http.dart' as http;
import 'package:quark/services/app_settings.dart';
import 'package:quark/services/authenticated_service.dart';
import 'package:quark/utils/error_text.dart';

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

  static Map<String, String> get _authHeaders => instance.authHeaders;

  static Future<SmbStatus> getStatus() async {
    final uri = apiBaseUri.resolve('/api/v0/smb/status');
    final response = await http.get(uri, headers: _authHeaders);
    if (response.statusCode < 200 || response.statusCode >= 300) {
      throw ApiException(response.statusCode, 'Failed to get SMB status');
    }
    final json = jsonDecode(response.body) as Map<String, dynamic>;
    return SmbStatus.fromJson(json['data'] as Map<String, dynamic>);
  }

  static Future<SmbStatus> setup(String user, String password) async {
    final uri = apiBaseUri.resolve('/api/v0/smb/setup');
    final response = await http.post(
      uri,
      headers: {'Content-Type': 'application/json', ..._authHeaders},
      body: jsonEncode({'user': user, 'password': password}),
    );
    if (response.statusCode < 200 || response.statusCode >= 300) {
      final body = jsonDecode(response.body) as Map<String, dynamic>?;
      throwApiError(response.statusCode, body?['error'], 'Setup failed');
    }
    final json = jsonDecode(response.body) as Map<String, dynamic>;
    return SmbStatus.fromJson(json['data'] as Map<String, dynamic>);
  }

  static Future<SmbStatus> teardown() async {
    final uri = apiBaseUri.resolve('/api/v0/smb');
    final response = await http.delete(uri, headers: _authHeaders);
    if (response.statusCode < 200 || response.statusCode >= 300) {
      final body = jsonDecode(response.body) as Map<String, dynamic>?;
      throwApiError(response.statusCode, body?['error'], 'Teardown failed');
    }
    final json = jsonDecode(response.body) as Map<String, dynamic>;
    return SmbStatus.fromJson(json['data'] as Map<String, dynamic>);
  }
}
