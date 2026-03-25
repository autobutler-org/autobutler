import 'dart:convert';

import 'package:autobutler/services/app_settings.dart';
import 'package:autobutler/services/authenticated_service.dart';
import 'package:flutter/foundation.dart';
import 'package:http/http.dart' as http;

class StorageDevice {
  const StorageDevice({
    required this.name,
    required this.devicePath,
    required this.mountPoint,
    required this.fileSystem,
    required this.totalBytes,
    required this.usedBytes,
    required this.availableBytes,
    required this.isInternal,
    required this.isEnabled,
    this.model = '',
    this.serial = '',
    this.categories = const {},
  });

  factory StorageDevice.fromJson(Map<String, dynamic> json) {
    // serial lives inside the nested usbInfo object
    final usbInfo = json['usbInfo'] as Map<String, dynamic>?;
    final serial = usbInfo?['serial'] as String? ?? '';

    final rawCats = json['categories'] as Map<String, dynamic>?;
    final categories = rawCats != null
        ? rawCats.map((k, v) => MapEntry(k, (v as num).toInt()))
        : const <String, int>{};

    return StorageDevice(
      name: json['name'] as String? ?? '',
      devicePath: json['devicePath'] as String? ?? '',
      mountPoint: json['mountPoint'] as String? ?? '',
      fileSystem: json['fileSystem'] as String? ?? '',
      totalBytes: (json['totalBytes'] as num?)?.toInt() ?? 0,
      usedBytes: (json['usedBytes'] as num?)?.toInt() ?? 0,
      availableBytes: (json['availableBytes'] as num?)?.toInt() ?? 0,
      isInternal: json['isInternal'] as bool? ?? false,
      isEnabled: json['isEnabled'] as bool? ?? false,
      model: json['model'] as String? ?? '',
      serial: serial,
      categories: categories,
    );
  }

  final String name;
  final String devicePath;
  final String mountPoint;
  final String fileSystem;
  final int totalBytes;
  final int usedBytes;
  final int availableBytes;
  final bool isInternal;
  final bool isEnabled;
  final String model;

  /// USB serial number; empty string for internal devices.
  final String serial;

  /// File category breakdown in bytes, e.g. {'documents': 1024, 'media': 2048}.
  final Map<String, int> categories;

  /// Whether this device is detected but not yet mounted.
  bool get isUnmounted => !isInternal && mountPoint.isEmpty;

  double get usedPercent => totalBytes > 0 ? usedBytes / totalBytes * 100 : 0;

  String get usedDisplay =>
      '${formatBytes(usedBytes)} / ${formatBytes(totalBytes)}';

  static String formatBytes(int bytes) {
    if (bytes >= 1e12) return '${(bytes / 1e12).toStringAsFixed(1)} TB';
    if (bytes >= 1e9) return '${(bytes / 1e9).toStringAsFixed(1)} GB';
    if (bytes >= 1e6) return '${(bytes / 1e6).toStringAsFixed(1)} MB';
    return '$bytes B';
  }
}

class StorageService with AuthenticatedService {
  static final StorageService _instance = StorageService._();
  StorageService._();
  static StorageService get instance => _instance;
  static Map<String, String> get _authHeaders => instance.authHeaders;

  static Uri get _apiBaseUri {
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

  /// Returns all storage devices with their current status and display names.
  static Future<List<StorageDevice>> listDevices() async {
    final uri = _apiBaseUri.resolve('/api/v1/storage/devices/status');
    final response = await http.get(uri, headers: _authHeaders);
    if (response.statusCode < 200 || response.statusCode >= 300) {
      throw Exception(
        'Failed to list storage devices (${response.statusCode})',
      );
    }
    final body = jsonDecode(response.body) as Map<String, dynamic>;
    final devices = body['devices'];
    if (devices == null || devices is! List) return [];
    return devices
        .whereType<Map<String, dynamic>>()
        .map(StorageDevice.fromJson)
        .toList();
  }

  /// Mounts a USB device by serial. Requires the butler to be running as root.
  static Future<void> mountDevice(String serial) async {
    final uri = _apiBaseUri.resolve('/api/v1/storage/devices/usb/$serial');
    final response = await http.post(uri, headers: _authHeaders);
    if (response.statusCode < 200 || response.statusCode >= 300) {
      final body = jsonDecode(response.body) as Map<String, dynamic>?;
      final msg =
          body?['error'] as String? ?? 'Mount failed (${response.statusCode})';
      throw Exception(msg);
    }
  }

  /// Sets a custom display name for a device identified by [devicePath].
  static Future<void> renameDevice(String devicePath, String name) async {
    // Device paths contain slashes (e.g. /dev/disk3s5) — pass as a query
    // param so they don't get misinterpreted as URL path segments.
    final uri = _apiBaseUri
        .resolve('/api/v1/storage/devices/rename')
        .replace(queryParameters: {'devicePath': devicePath});
    final response = await http.patch(
      uri,
      headers: {'Content-Type': 'application/json', ..._authHeaders},
      body: jsonEncode({'name': name}),
    );
    if (response.statusCode < 200 || response.statusCode >= 300) {
      throw Exception(
        'Failed to rename device (${response.statusCode}): ${response.body}',
      );
    }
  }
}
