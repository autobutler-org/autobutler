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
    this.role = 'unassigned',
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
      role: json['role'] as String? ?? 'unassigned',
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

  /// Device role: 'default-storage', 'snapshot-backup', or 'unassigned'.
  final String role;

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

  // ── Client-side device-list cache ─────────────────────────────────────────
  // A 10-second TTL prevents hammering the expensive /storage/devices/status
  // endpoint. Simultaneous callers share a single in-flight Future so only one
  // HTTP request is ever in flight at a time (#1022).
  static const _deviceCacheTtl = Duration(seconds: 10);
  static List<StorageDevice>? _cachedDevices;
  static DateTime? _cachedAt;
  static Future<List<StorageDevice>>? _inFlight;

  /// Invalidate the device cache immediately (e.g. after mount/rename).
  static void invalidateDeviceCache() {
    _cachedDevices = null;
    _cachedAt = null;
    _inFlight = null;
  }

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

  /// Returns all storage devices with their current status and display names.
  ///
  /// Results are cached for [_deviceCacheTtl]. Simultaneous callers share a
  /// single in-flight request so only one HTTP call is ever outstanding (#1022).
  static Future<List<StorageDevice>> listDevices() async {
    final now = DateTime.now();
    if (_cachedDevices != null &&
        _cachedAt != null &&
        now.difference(_cachedAt!) < _deviceCacheTtl) {
      return _cachedDevices!;
    }
    // Coalesce concurrent callers onto the same Future.
    _inFlight ??= _fetchDevices().then(
      (result) {
        _cachedDevices = result;
        _cachedAt = DateTime.now();
        _inFlight = null;
        return result;
      },
      onError: (Object e) {
        _inFlight = null;
        throw e;
      },
    );
    return _inFlight!;
  }

  static Future<List<StorageDevice>> _fetchDevices() async {
    final uri = _apiBaseUri.resolve('/api/v0/storage/devices/status');
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
    final uri = _apiBaseUri.resolve('/api/v0/storage/devices/usb/$serial');
    final response = await http.post(uri, headers: _authHeaders);
    if (response.statusCode < 200 || response.statusCode >= 300) {
      final body = jsonDecode(response.body) as Map<String, dynamic>?;
      final msg =
          body?['error'] as String? ?? 'Mount failed (${response.statusCode})';
      throw Exception(msg);
    }
    // Device state changed — invalidate the cache so the next listDevices()
    // returns fresh data.
    invalidateDeviceCache();
  }

  /// Sets a custom display name for a device identified by [devicePath].
  static Future<void> renameDevice(String devicePath, String name) async {
    // Device paths contain slashes (e.g. /dev/disk3s5) — pass as a query
    // param so they don't get misinterpreted as URL path segments.
    final uri = _apiBaseUri
        .resolve('/api/v0/storage/devices/rename')
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
    invalidateDeviceCache();
  }

  static Future<bool> setDeviceRole({
    required String serial,
    required String role,
    required String username,
    required String password,
    bool moveVault = false,
  }) async {
    final uri = _apiBaseUri.resolve('/api/v0/storage/devices/role');
    final response = await http.put(
      uri,
      headers: {'Content-Type': 'application/json', ..._authHeaders},
      body: jsonEncode({
        'serial': serial,
        'role': role,
        'username': username,
        'password': password,
        if (moveVault) 'moveVault': true,
      }),
    );
    if (response.statusCode < 200 || response.statusCode >= 300) {
      final body = jsonDecode(response.body) as Map<String, dynamic>?;
      throw Exception(
        body?['error'] ?? 'Failed to set role (${response.statusCode})',
      );
    }
    invalidateDeviceCache();
    final body = jsonDecode(response.body) as Map<String, dynamic>?;
    return body?['vaultMigrated'] == true;
  }

  static Future<String> startSnapshotBackup({
    required String targetDeviceSerial,
    String? username,
    String? password,
    String? recoveryPassword,
  }) async {
    final uri = _apiBaseUri.resolve('/api/v0/storage/devices/snapshot-backup');
    final body = <String, dynamic>{'targetDeviceSerial': targetDeviceSerial};
    if (username != null) body['username'] = username;
    if (password != null) body['password'] = password;
    if (recoveryPassword != null) body['recoveryPassword'] = recoveryPassword;

    final response = await http.post(
      uri,
      headers: {'Content-Type': 'application/json', ..._authHeaders},
      body: jsonEncode(body),
    );
    if (response.statusCode < 200 || response.statusCode >= 300) {
      final data = jsonDecode(response.body) as Map<String, dynamic>?;
      throw Exception(
        data?['error'] ?? 'Failed to start backup (${response.statusCode})',
      );
    }
    final data = jsonDecode(response.body) as Map<String, dynamic>;
    return (data['data'] as Map<String, dynamic>)['jobId'] as String;
  }

  static Future<BackupJobStatus> getSnapshotBackupStatus(String jobId) async {
    final uri = _apiBaseUri.resolve(
      '/api/v0/storage/devices/snapshot-backup/status/$jobId',
    );
    final response = await http.get(uri, headers: _authHeaders);
    if (response.statusCode < 200 || response.statusCode >= 300) {
      throw Exception('Failed to get backup status (${response.statusCode})');
    }
    final data = jsonDecode(response.body) as Map<String, dynamic>;
    return BackupJobStatus.fromJson(data['data'] as Map<String, dynamic>);
  }

  static Future<VerifyResult> verifySnapshotBackup({
    required String deviceSerial,
    bool full = false,
  }) async {
    final uri = _apiBaseUri.resolve(
      '/api/v0/storage/devices/snapshot-backup/verify',
    );
    final response = await http.post(
      uri,
      headers: {'Content-Type': 'application/json', ..._authHeaders},
      body: jsonEncode({'deviceSerial': deviceSerial, 'full': full}),
    );
    if (response.statusCode < 200 || response.statusCode >= 300) {
      final data = jsonDecode(response.body) as Map<String, dynamic>?;
      throw Exception(
        data?['error'] ?? 'Verify failed (${response.statusCode})',
      );
    }
    final data = jsonDecode(response.body) as Map<String, dynamic>;
    return VerifyResult.fromJson(data['data'] as Map<String, dynamic>);
  }
}

class BackupJobStatus {
  final String id;
  final String status;
  final double progress;
  final int totalFiles;
  final int filesCopied;
  final int filesSkipped;
  final int totalBytes;
  final int bytesCopied;
  final String? errorMsg;

  const BackupJobStatus({
    required this.id,
    required this.status,
    required this.progress,
    required this.totalFiles,
    required this.filesCopied,
    required this.filesSkipped,
    required this.totalBytes,
    required this.bytesCopied,
    this.errorMsg,
  });

  factory BackupJobStatus.fromJson(Map<String, dynamic> json) {
    return BackupJobStatus(
      id: json['id'] as String? ?? '',
      status: json['status'] as String? ?? '',
      progress: (json['progress'] as num?)?.toDouble() ?? 0,
      totalFiles: (json['totalFiles'] as num?)?.toInt() ?? 0,
      filesCopied: (json['filesCopied'] as num?)?.toInt() ?? 0,
      filesSkipped: (json['filesSkipped'] as num?)?.toInt() ?? 0,
      totalBytes: (json['totalBytes'] as num?)?.toInt() ?? 0,
      bytesCopied: (json['bytesCopied'] as num?)?.toInt() ?? 0,
      errorMsg: json['errorMsg'] as String?,
    );
  }

  bool get isRunning =>
      status == 'PENDING' || status == 'SCANNING' || status == 'COPYING';
  bool get isComplete => status == 'COMPLETED';
  bool get isFailed => status == 'FAILED';
}

class VerifyResult {
  final int ok;
  final List<String> missing;
  final List<String> corrupted;
  final List<String> added;

  const VerifyResult({
    required this.ok,
    this.missing = const [],
    this.corrupted = const [],
    this.added = const [],
  });

  factory VerifyResult.fromJson(Map<String, dynamic> json) {
    return VerifyResult(
      ok: (json['ok'] as num?)?.toInt() ?? 0,
      missing: (json['missing'] as List<dynamic>?)?.cast<String>() ?? [],
      corrupted: (json['corrupted'] as List<dynamic>?)?.cast<String>() ?? [],
      added: (json['added'] as List<dynamic>?)?.cast<String>() ?? [],
    );
  }

  bool get isHealthy => missing.isEmpty && corrupted.isEmpty;
}
