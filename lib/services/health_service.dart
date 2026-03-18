import 'dart:convert';

import 'package:autobutler/services/app_settings.dart';
import 'package:autobutler/services/authenticated_service.dart';
import 'package:flutter/foundation.dart';
import 'package:http/http.dart' as http;

class HealthStatus {
  const HealthStatus({
    required this.healthy,
    required this.alerts,
    required this.cpuPercent,
    required this.cpuCorePercents,
    required this.memPercent,
    required this.memUsedBytes,
    required this.memTotalBytes,
    required this.diskPercent,
    required this.diskUsedBytes,
    required this.diskTotalBytes,
    required this.temperatureCelsius,
  });

  factory HealthStatus.fromJson(Map<String, dynamic> json) {
    return HealthStatus(
      healthy: json['healthy'] as bool? ?? true,
      alerts:
          (json['alerts'] as List<dynamic>?)?.cast<String>().toList(
            growable: false,
          ) ??
          const [],
      cpuPercent: (json['cpuPercent'] as num?)?.toDouble() ?? 0,
      cpuCorePercents:
          (json['cpuCorePercents'] as List<dynamic>?)
              ?.map((e) => (e as num).toDouble())
              .toList(growable: false) ??
          const [],
      memPercent: (json['memPercent'] as num?)?.toDouble() ?? 0,
      memUsedBytes: (json['memUsedBytes'] as num?)?.toInt() ?? 0,
      memTotalBytes: (json['memTotalBytes'] as num?)?.toInt() ?? 0,
      diskPercent: (json['diskPercent'] as num?)?.toDouble() ?? 0,
      diskUsedBytes: (json['diskUsedBytes'] as num?)?.toInt() ?? 0,
      diskTotalBytes: (json['diskTotalBytes'] as num?)?.toInt() ?? 0,
      temperatureCelsius: (json['temperatureCelsius'] as num?)?.toDouble() ?? 0,
    );
  }

  final bool healthy;
  final List<String> alerts;
  final double cpuPercent;
  final List<double> cpuCorePercents;
  final int memUsedBytes;
  final int memTotalBytes;
  final double memPercent;
  final double diskPercent;
  final int diskUsedBytes;
  final int diskTotalBytes;
  final double temperatureCelsius;
}

class HealthService with AuthenticatedService {
  static final HealthService _instance = HealthService._();
  HealthService._();
  static HealthService get instance => _instance;

  static Map<String, String> get _authHeaders => instance.authHeaders;

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

  static Future<HealthStatus> getHealth() async {
    final uri = _apiBaseUri.resolve('/api/v1/health');
    final response = await http.get(uri, headers: _authHeaders);
    if (response.statusCode < 200 || response.statusCode >= 300) {
      throw Exception('Failed to fetch health (${response.statusCode})');
    }
    final decoded = jsonDecode(response.body);
    if (decoded is! Map<String, dynamic>) {
      throw const FormatException('Invalid health response format');
    }

    final data = decoded['data'];
    if (data is Map<String, dynamic>) {
      return HealthStatus.fromJson(data);
    }

    return HealthStatus.fromJson(decoded);
  }
}
