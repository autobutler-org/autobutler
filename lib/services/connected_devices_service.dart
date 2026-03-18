import 'dart:convert';

import 'package:autobutler/services/app_settings.dart';
import 'package:flutter/foundation.dart';
import 'package:http/http.dart' as http;

class ConnectedDevice {
  const ConnectedDevice({
    required this.id,
    required this.ipAddress,
    required this.userAgent,
    required this.firstSeenAt,
    required this.lastSeenAt,
    required this.requestCount,
  });

  factory ConnectedDevice.fromJson(Map<String, dynamic> json) {
    return ConnectedDevice(
      id: json['id'] as int,
      ipAddress: json['ipAddress'] as String,
      userAgent: json['userAgent'] as String? ?? '',
      firstSeenAt: DateTime.parse(json['firstSeenAt'] as String),
      lastSeenAt: DateTime.parse(json['lastSeenAt'] as String),
      requestCount: json['requestCount'] as int,
    );
  }

  final int id;
  final String ipAddress;
  final String userAgent;
  final DateTime firstSeenAt;
  final DateTime lastSeenAt;
  final int requestCount;
}

class ConnectedDevicesService {
  static Uri get _apiBaseUri {
    final configured = AppSettings.instance.activeHost;
    final base = configured ??
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

  static Future<List<ConnectedDevice>> listDevices() async {
    final uri = _apiBaseUri.resolve('/api/v1/devices');
    final response = await http.get(uri);
    if (response.statusCode < 200 || response.statusCode >= 300) {
      throw Exception('Failed to fetch devices (${response.statusCode})');
    }
    final json = jsonDecode(response.body) as Map<String, dynamic>;
    final data = json['data'] as List<dynamic>;
    return data
        .cast<Map<String, dynamic>>()
        .map(ConnectedDevice.fromJson)
        .toList(growable: false);
  }

  static Future<void> deleteDevice(int id) async {
    final uri = _apiBaseUri.resolve('/api/v1/devices/$id');
    final response = await http.delete(uri);
    if (response.statusCode < 200 || response.statusCode >= 300) {
      throw Exception('Failed to delete device (${response.statusCode})');
    }
  }
}
