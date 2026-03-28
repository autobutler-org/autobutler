import 'dart:convert';

import 'package:autobutler/services/app_settings.dart';
import 'package:autobutler/services/authenticated_service.dart';
import 'package:flutter/foundation.dart';
import 'package:http/http.dart' as http;

class BranchBuild {
  const BranchBuild({
    required this.branch,
    required this.prNumber,
    required this.prTitle,
    required this.builtAt,
    required this.artifactId,
  });

  factory BranchBuild.fromJson(Map<String, dynamic> json) {
    return BranchBuild(
      branch: json['branch'] as String? ?? '',
      prNumber: json['prNumber'] as int? ?? 0,
      prTitle: json['prTitle'] as String? ?? '',
      builtAt: DateTime.tryParse(json['builtAt'] as String? ?? '') ??
          DateTime.fromMillisecondsSinceEpoch(0),
      artifactId: (json['artifactId'] as num?)?.toInt() ?? 0,
    );
  }

  final String branch;
  final int prNumber;
  final String prTitle;
  final DateTime builtAt;
  final int artifactId;
}

class BranchService with AuthenticatedService {
  static final BranchService _instance = BranchService._();
  BranchService._();
  static BranchService get instance => _instance;

  static Map<String, String> get _authHeaders => instance.authHeaders;

  static Uri get _apiBaseUri {
    final configured = AppSettings.instance.activeHost;
    final base = configured ??
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

  static Future<bool> isDevModeEnabled() async {
    final uri = _apiBaseUri.resolve('/api/v1/settings/dev-mode');
    final response = await http.get(uri, headers: _authHeaders);
    if (response.statusCode < 200 || response.statusCode >= 300) {
      return false;
    }
    final decoded = jsonDecode(response.body);
    if (decoded is! Map<String, dynamic>) return false;
    return decoded['enabled'] as bool? ?? false;
  }

  static Future<List<BranchBuild>> listBranches() async {
    final uri = _apiBaseUri.resolve('/api/v1/version/branches');
    final response = await http.get(uri, headers: _authHeaders);
    if (response.statusCode < 200 || response.statusCode >= 300) {
      throw Exception('Failed to list branches (${response.statusCode})');
    }
    final decoded = jsonDecode(response.body);
    if (decoded is! List) {
      throw Exception('Unexpected branches response format');
    }
    return decoded
        .whereType<Map<String, dynamic>>()
        .map(BranchBuild.fromJson)
        .toList(growable: false);
  }

  static Future<void> deployBranch(String branch) async {
    final uri = _apiBaseUri.resolve('/api/v1/version/update');
    final body = jsonEncode({'branch': branch});
    await http.post(
      uri,
      headers: {'Content-Type': 'application/json', ..._authHeaders},
      body: body,
    );
  }

  static Future<void> returnToRelease(String targetVersion) async {
    final uri = _apiBaseUri.resolve('/api/v1/version/update');
    final body = jsonEncode({'version': targetVersion});
    await http.post(
      uri,
      headers: {'Content-Type': 'application/json', ..._authHeaders},
      body: body,
    );
  }

  static Future<bool> checkServerReady() async {
    try {
      final uri = _apiBaseUri.resolve('/api/v1/version');
      final response = await http
          .get(uri, headers: _authHeaders)
          .timeout(const Duration(seconds: 5));
      return response.statusCode >= 200 && response.statusCode < 300;
    } catch (_) {
      return false;
    }
  }

  static Future<String?> getLatestReleaseVersion() async {
    final uri = _apiBaseUri.resolve('/api/v1/version/available');
    final response = await http.get(uri, headers: _authHeaders);
    if (response.statusCode < 200 || response.statusCode >= 300) {
      throw Exception('Failed to fetch available versions (${response.statusCode})');
    }
    final decoded = jsonDecode(response.body);
    if (decoded is! List || decoded.isEmpty) return null;
    final first = decoded.first;
    if (first is! Map<String, dynamic>) return null;
    return first['version'] as String?;
  }
}
