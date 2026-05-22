// ignore_for_file: use_null_aware_elements
import 'dart:convert';

import 'package:autobutler/services/app_settings.dart';
import 'package:autobutler/services/authenticated_service.dart';
import 'package:http/http.dart' as http;
import 'package:flutter/foundation.dart';

class VaultStatus {
  final bool initialized;
  final bool locked;
  final int autoLockSeconds;
  final String storageDevice;
  final bool deviceConnected;
  final String lockReason;

  const VaultStatus({
    required this.initialized,
    required this.locked,
    required this.autoLockSeconds,
    this.storageDevice = 'internal',
    this.deviceConnected = true,
    this.lockReason = '',
  });

  bool get isExternal => storageDevice != 'internal';

  factory VaultStatus.fromJson(Map<String, dynamic> json) => VaultStatus(
    initialized: json['initialized'] as bool? ?? false,
    locked: json['locked'] as bool? ?? true,
    autoLockSeconds: json['autoLockSeconds'] as int? ?? 300,
    storageDevice: json['storageDevice'] as String? ?? 'internal',
    deviceConnected: json['deviceConnected'] as bool? ?? true,
    lockReason: json['lockReason'] as String? ?? '',
  );
}

class VaultStorageLocation {
  final String deviceSerial;
  final bool isExternal;
  final bool deviceConnected;
  final String deviceName;

  const VaultStorageLocation({
    required this.deviceSerial,
    required this.isExternal,
    required this.deviceConnected,
    required this.deviceName,
  });

  factory VaultStorageLocation.fromJson(Map<String, dynamic> json) =>
      VaultStorageLocation(
        deviceSerial: json['deviceSerial'] as String? ?? '',
        isExternal: json['isExternal'] as bool? ?? false,
        deviceConnected: json['deviceConnected'] as bool? ?? true,
        deviceName: json['deviceName'] as String? ?? 'Internal Storage',
      );
}

class VaultEntryItem {
  final int id;
  final String name;
  final String urlHost;
  final int? folderId;
  final String createdAt;
  final String updatedAt;

  const VaultEntryItem({
    required this.id,
    required this.name,
    required this.urlHost,
    this.folderId,
    required this.createdAt,
    required this.updatedAt,
  });

  factory VaultEntryItem.fromJson(Map<String, dynamic> json) => VaultEntryItem(
    id: json['id'] as int,
    name: json['name'] as String? ?? '',
    urlHost: json['urlHost'] as String? ?? '',
    folderId: json['folderId'] as int?,
    createdAt: json['createdAt'] as String? ?? '',
    updatedAt: json['updatedAt'] as String? ?? '',
  );
}

class VaultEntryDetail {
  final int id;
  final String name;
  final String url;
  final String urlHost;
  final String username;
  final String password;
  final String notes;
  final String totpSecret;
  final List<VaultCustomField> customFields;
  final int? folderId;
  final String createdAt;
  final String updatedAt;

  const VaultEntryDetail({
    required this.id,
    required this.name,
    required this.url,
    required this.urlHost,
    required this.username,
    required this.password,
    this.notes = '',
    this.totpSecret = '',
    this.customFields = const [],
    this.folderId,
    required this.createdAt,
    required this.updatedAt,
  });

  factory VaultEntryDetail.fromJson(Map<String, dynamic> json) =>
      VaultEntryDetail(
        id: json['id'] as int,
        name: json['name'] as String? ?? '',
        url: json['url'] as String? ?? '',
        urlHost: json['urlHost'] as String? ?? '',
        username: json['username'] as String? ?? '',
        password: json['password'] as String? ?? '',
        notes: json['notes'] as String? ?? '',
        totpSecret: json['totpSecret'] as String? ?? '',
        customFields:
            (json['customFields'] as List<dynamic>?)
                ?.map(
                  (e) => VaultCustomField.fromJson(e as Map<String, dynamic>),
                )
                .toList() ??
            [],
        folderId: json['folderId'] as int?,
        createdAt: json['createdAt'] as String? ?? '',
        updatedAt: json['updatedAt'] as String? ?? '',
      );
}

class VaultCustomField {
  final String label;
  final String value;
  final bool hidden;

  const VaultCustomField({
    required this.label,
    required this.value,
    this.hidden = false,
  });

  factory VaultCustomField.fromJson(Map<String, dynamic> json) =>
      VaultCustomField(
        label: json['label'] as String? ?? '',
        value: json['value'] as String? ?? '',
        hidden: json['hidden'] as bool? ?? false,
      );

  Map<String, dynamic> toJson() => {
    'label': label,
    'value': value,
    'hidden': hidden,
  };
}

class VaultFolder {
  final int id;
  final String name;
  final int? parentId;
  final int sortOrder;
  final String createdAt;

  const VaultFolder({
    required this.id,
    required this.name,
    this.parentId,
    this.sortOrder = 0,
    required this.createdAt,
  });

  factory VaultFolder.fromJson(Map<String, dynamic> json) => VaultFolder(
    id: json['id'] as int,
    name: json['name'] as String? ?? '',
    parentId: json['parentId'] as int?,
    sortOrder: json['sortOrder'] as int? ?? 0,
    createdAt: json['createdAt'] as String? ?? '',
  );
}

class VaultService with AuthenticatedService {
  static final VaultService _instance = VaultService._();
  VaultService._();
  static VaultService get instance => _instance;

  static Map<String, String> get _authHeaders => instance.authHeaders;

  static Map<String, String> get _jsonHeaders => {
    ..._authHeaders,
    'Content-Type': 'application/json',
  };

  static Uri _apiUri(String path) {
    final configured = AppSettings.instance.activeHost;
    final base =
        configured ??
        String.fromEnvironment(
          'API_BASE_URL',
          defaultValue: 'http://localhost:8080',
        );
    final uri = Uri.parse(base);
    final isLoopbackHost =
        uri.host == 'localhost' || uri.host == '127.0.0.1' || uri.host == '::1';
    Uri resolved;
    if (!kIsWeb &&
        defaultTargetPlatform == TargetPlatform.android &&
        isLoopbackHost) {
      resolved = uri.replace(host: '10.0.2.2');
    } else {
      resolved = uri;
    }
    return resolved.resolve('/api/v1$path');
  }

  static Future<VaultStatus> getStatus() async {
    final resp = await http.get(
      _apiUri('/vault/status'),
      headers: _authHeaders,
    );
    if (resp.statusCode != 200) {
      throw Exception('Failed to get vault status: ${resp.statusCode}');
    }
    return VaultStatus.fromJson(json.decode(resp.body) as Map<String, dynamic>);
  }

  static Future<void> setup(String password) async {
    final resp = await http.post(
      _apiUri('/vault/setup'),
      headers: _jsonHeaders,
      body: json.encode({'masterPassword': password}),
    );
    if (resp.statusCode != 200) {
      final body = json.decode(resp.body) as Map<String, dynamic>;
      throw Exception(body['error'] ?? 'Setup failed: ${resp.statusCode}');
    }
  }

  static Future<bool> unlock(String password) async {
    final resp = await http.post(
      _apiUri('/vault/unlock'),
      headers: _jsonHeaders,
      body: json.encode({'masterPassword': password}),
    );
    if (resp.statusCode == 401) return false;
    if (resp.statusCode != 200) {
      throw Exception('Unlock failed: ${resp.statusCode}');
    }
    return true;
  }

  static Future<void> lock() async {
    final resp = await http.post(_apiUri('/vault/lock'), headers: _jsonHeaders);
    if (resp.statusCode != 200) {
      throw Exception('Lock failed: ${resp.statusCode}');
    }
  }

  static Future<List<VaultEntryItem>> listEntries() async {
    final resp = await http.get(
      _apiUri('/vault/entries'),
      headers: _authHeaders,
    );
    if (resp.statusCode != 200) {
      throw Exception('Failed to list entries: ${resp.statusCode}');
    }
    final body = json.decode(resp.body) as Map<String, dynamic>;
    final list = body['entries'] as List<dynamic>? ?? [];
    return list
        .map((e) => VaultEntryItem.fromJson(e as Map<String, dynamic>))
        .toList();
  }

  static Future<VaultEntryDetail> getEntry(int id) async {
    final resp = await http.get(
      _apiUri('/vault/entries/$id'),
      headers: _authHeaders,
    );
    if (resp.statusCode == 423) {
      throw VaultLockedException();
    }
    if (resp.statusCode != 200) {
      throw Exception('Failed to get entry: ${resp.statusCode}');
    }
    return VaultEntryDetail.fromJson(
      json.decode(resp.body) as Map<String, dynamic>,
    );
  }

  static Future<VaultEntryDetail> createEntry({
    required String name,
    String url = '',
    String username = '',
    String password = '',
    String notes = '',
    String totpSecret = '',
    List<VaultCustomField> customFields = const [],
    int? folderId,
  }) async {
    final resp = await http.post(
      _apiUri('/vault/entries'),
      headers: _jsonHeaders,
      body: json.encode({
        'name': name,
        'url': url,
        'username': username,
        'password': password,
        'notes': notes,
        'totpSecret': totpSecret,
        'customFields': customFields.map((f) => f.toJson()).toList(),
        if (folderId != null) 'folderId': folderId,
      }),
    );
    if (resp.statusCode == 423) throw VaultLockedException();
    if (resp.statusCode != 200) {
      throw Exception('Failed to create entry: ${resp.statusCode}');
    }
    return VaultEntryDetail.fromJson(
      json.decode(resp.body) as Map<String, dynamic>,
    );
  }

  static Future<void> updateEntry({
    required int id,
    required String name,
    String url = '',
    String username = '',
    String password = '',
    String notes = '',
    String totpSecret = '',
    List<VaultCustomField> customFields = const [],
    int? folderId,
  }) async {
    final resp = await http.put(
      _apiUri('/vault/entries/$id'),
      headers: _jsonHeaders,
      body: json.encode({
        'name': name,
        'url': url,
        'username': username,
        'password': password,
        'notes': notes,
        'totpSecret': totpSecret,
        'customFields': customFields.map((f) => f.toJson()).toList(),
        if (folderId != null) 'folderId': folderId,
      }),
    );
    if (resp.statusCode == 423) throw VaultLockedException();
    if (resp.statusCode != 200) {
      throw Exception('Failed to update entry: ${resp.statusCode}');
    }
  }

  static Future<void> deleteEntry(int id) async {
    final resp = await http.delete(
      _apiUri('/vault/entries/$id'),
      headers: _authHeaders,
    );
    if (resp.statusCode != 200) {
      throw Exception('Failed to delete entry: ${resp.statusCode}');
    }
  }

  static Future<List<VaultFolder>> listFolders() async {
    final resp = await http.get(
      _apiUri('/vault/folders'),
      headers: _authHeaders,
    );
    if (resp.statusCode != 200) {
      throw Exception('Failed to list folders: ${resp.statusCode}');
    }
    final body = json.decode(resp.body) as Map<String, dynamic>;
    final list = body['folders'] as List<dynamic>? ?? [];
    return list
        .map((e) => VaultFolder.fromJson(e as Map<String, dynamic>))
        .toList();
  }

  static Future<void> createFolder({
    required String name,
    int? parentId,
  }) async {
    final resp = await http.post(
      _apiUri('/vault/folders'),
      headers: _jsonHeaders,
      body: json.encode({
        'name': name,
        if (parentId != null) 'parentId': parentId,
      }),
    );
    if (resp.statusCode != 200) {
      throw Exception('Failed to create folder: ${resp.statusCode}');
    }
  }

  static Future<void> deleteFolder(int id) async {
    final resp = await http.delete(
      _apiUri('/vault/folders/$id'),
      headers: _authHeaders,
    );
    if (resp.statusCode != 200) {
      throw Exception('Failed to delete folder: ${resp.statusCode}');
    }
  }

  static Future<String> generatePassword({
    int length = 20,
    bool uppercase = true,
    bool lowercase = true,
    bool digits = true,
    bool symbols = true,
    bool avoidAmbiguous = false,
  }) async {
    final resp = await http.post(
      _apiUri('/vault/generate'),
      headers: _jsonHeaders,
      body: json.encode({
        'length': length,
        'uppercase': uppercase,
        'lowercase': lowercase,
        'digits': digits,
        'symbols': symbols,
        'avoidAmbiguous': avoidAmbiguous,
      }),
    );
    if (resp.statusCode != 200) {
      throw Exception('Failed to generate password: ${resp.statusCode}');
    }
    final body = json.decode(resp.body) as Map<String, dynamic>;
    return body['password'] as String;
  }

  static Future<VaultStorageLocation> getStorageLocation() async {
    final resp = await http.get(
      _apiUri('/vault/storage-location'),
      headers: _authHeaders,
    );
    if (resp.statusCode != 200) {
      throw Exception('Failed to get storage location: ${resp.statusCode}');
    }
    return VaultStorageLocation.fromJson(
      json.decode(resp.body) as Map<String, dynamic>,
    );
  }

  static Future<void> setStorageLocation({
    required String targetDeviceSerial,
    required String username,
    required String password,
  }) async {
    final resp = await http.put(
      _apiUri('/vault/storage-location'),
      headers: _jsonHeaders,
      body: json.encode({
        'targetDeviceSerial': targetDeviceSerial,
        'username': username,
        'password': password,
      }),
    );
    if (resp.statusCode == 423) throw VaultLockedException();
    if (resp.statusCode == 401) {
      throw Exception('Invalid credentials');
    }
    if (resp.statusCode != 200) {
      final body = json.decode(resp.body) as Map<String, dynamic>;
      throw Exception(
        body['error'] ?? 'Failed to change storage: ${resp.statusCode}',
      );
    }
  }

  static Future<Map<String, dynamic>> importEntries({
    required List<int> fileBytes,
    required String fileName,
    String format = 'auto',
  }) async {
    final request = http.MultipartRequest('POST', _apiUri('/vault/import'));
    _authHeaders.forEach((k, v) => request.headers[k] = v);
    request.fields['format'] = format;
    request.files.add(
      http.MultipartFile.fromBytes('file', fileBytes, filename: fileName),
    );
    final streamed = await request.send();
    final body = await streamed.stream.bytesToString();
    if (streamed.statusCode == 423) throw VaultLockedException();
    if (streamed.statusCode != 200) {
      throw Exception('Import failed: ${streamed.statusCode}');
    }
    return json.decode(body) as Map<String, dynamic>;
  }

  static Future<List<int>> exportEntries({String format = 'json'}) async {
    final resp = await http.get(
      _apiUri('/vault/export?format=$format'),
      headers: _authHeaders,
    );
    if (resp.statusCode == 423) throw VaultLockedException();
    if (resp.statusCode != 200) {
      throw Exception('Export failed: ${resp.statusCode}');
    }
    return resp.bodyBytes;
  }

  static Future<void> changePassword({
    required String currentPassword,
    required String newPassword,
  }) async {
    final resp = await http.put(
      _apiUri('/vault/change-password'),
      headers: _jsonHeaders,
      body: json.encode({
        'currentPassword': currentPassword,
        'newPassword': newPassword,
      }),
    );
    if (resp.statusCode == 423) throw VaultLockedException();
    if (resp.statusCode == 401) {
      throw Exception('Current password is incorrect');
    }
    if (resp.statusCode != 200) {
      throw Exception('Failed to change password: ${resp.statusCode}');
    }
  }
}

class VaultLockedException implements Exception {
  @override
  String toString() => 'Vault is locked';
}
