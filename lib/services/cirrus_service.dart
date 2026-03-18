import 'dart:convert';

import 'package:autobutler/models/cirrus_file_node.dart';
import 'package:autobutler/services/app_settings.dart';
import 'package:autobutler/utils/web_download_stub.dart'
    if (dart.library.html) 'package:autobutler/utils/web_download_web.dart'
    as web_download;
import 'package:flutter/foundation.dart';
import 'package:flutter_file_dialog/flutter_file_dialog.dart';
import 'package:http/http.dart' as http;

class CirrusService {
  /// Returns Authorization header if a session token is set, empty map otherwise.
  /// All API requests should include this to support basic-auth (#650).
  static Map<String, String> get _authHeaders {
    final token = AppSettings.instance.sessionToken;
    if (token == null || token.isEmpty) return const {};
    return {'Authorization': 'Bearer $token'};
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
    final isLoopbackHost =
        uri.host == 'localhost' || uri.host == '127.0.0.1' || uri.host == '::1';

    if (!kIsWeb &&
        defaultTargetPlatform == TargetPlatform.android &&
        isLoopbackHost) {
      return uri.replace(host: '10.0.2.2');
    }

    return uri;
  }

  static Uri constructMediaUrl(String filePath, {String? serial}) {
    final querySegments = <String>[
      'filePath=${Uri.encodeQueryComponent(filePath)}',
    ];

    final serialValue = serial?.trim() ?? '';
    if (serialValue.isNotEmpty) {
      querySegments.add('serial=${Uri.encodeQueryComponent(serialValue)}');
    }
    final endpointUri = _apiBaseUri.resolve('/api/v1/cirrus/download');
    return endpointUri.replace(query: querySegments.join('&'));
  }

  /// Construct a URL for the thumbnail endpoint.
  /// The backend exposes thumbnails at /api/v1/thumbnails/*filePath where filePath is a
  /// path-like segment. Each path segment is percent-encoded to preserve slashes.
  static Uri constructThumbnailUrl(String filePath, {String? serial}) {
    final trimmed = filePath.trim();
    final normalized = trimmed.startsWith('/') ? trimmed.substring(1) : trimmed;
    final encodedPath = normalized
        .split('/')
        .map((s) => Uri.encodeComponent(s))
        .join('/');
    final endpointUri = _apiBaseUri.resolve('/api/v1/thumbnails/$encodedPath');

    final serialValue = serial?.trim() ?? '';
    if (serialValue.isNotEmpty) {
      return endpointUri.replace(queryParameters: {'serial': serialValue});
    }
    return endpointUri;
  }

  static Future<List<CirrusFileNode>> getFiles(
    String path, {
    List<String>? serials,
  }) async {
    final normalizedPath = _normalizePath(path);
    final serialValues =
        serials
            ?.map((value) => value.trim())
            .where((value) => value.isNotEmpty)
            .toList(growable: false) ??
        const <String>[];

    final querySegments = <String>[];
    if (normalizedPath.isNotEmpty) {
      querySegments.add(
        'rootDir=${Uri.encodeQueryComponent(_toRootDir(normalizedPath))}',
      );
    }
    for (final serial in serialValues) {
      querySegments.add('serial=${Uri.encodeQueryComponent(serial)}');
    }

    final endpointUri = _apiBaseUri.resolve('/api/v1/cirrus');
    final uri = querySegments.isEmpty
        ? endpointUri
        : endpointUri.replace(query: querySegments.join('&'));

    final response = await http.get(uri, headers: _authHeaders);
    if (response.statusCode < 200 || response.statusCode >= 300) {
      throw Exception('Failed to load cirrus files (${response.statusCode})');
    }

    final decoded = jsonDecode(response.body);
    if (decoded is! List) {
      throw Exception('Unexpected cirrus response format');
    }

    return decoded
        .whereType<Map<String, dynamic>>()
        .map(CirrusFileNode.fromJson)
        .toList(growable: false);
  }

  static Future<List<CirrusFileNode>> searchFiles(
    String query, {
    List<String>? serials,
  }) async {
    final serialValues =
        serials
            ?.map((value) => value.trim())
            .where((value) => value.isNotEmpty)
            .toList(growable: false) ??
        const <String>[];
    final querySegments = <String>[];
    querySegments.add('query=${Uri.encodeQueryComponent(query)}');
    for (final serial in serialValues) {
      querySegments.add('serial=${Uri.encodeQueryComponent(serial)}');
    }
    final endpointUri = _apiBaseUri.resolve('/api/v1/cirrus/search');
    final uri = querySegments.isEmpty
        ? endpointUri
        : endpointUri.replace(query: querySegments.join('&'));
    final response = await http.get(uri, headers: _authHeaders);
    if (response.statusCode < 200 || response.statusCode >= 300) {
      throw Exception('Failed to load cirrus files (${response.statusCode})');
    }

    final decoded = jsonDecode(response.body);
    if (decoded is! List) {
      throw Exception('Unexpected cirrus response format');
    }

    return decoded
        .whereType<Map<String, dynamic>>()
        .map(CirrusFileNode.fromJson)
        .toList(growable: false);
  }

  static Future<void> deleteFile(
    String rootDir,
    String fileName, {
    String? deviceSerial,
  }) async {
    final querySegments = <String>[
      'rootDir=${Uri.encodeQueryComponent(rootDir)}',
      'filePaths=${Uri.encodeQueryComponent(fileName)}',
    ];
    final serial = deviceSerial?.trim() ?? '';
    if (serial.isNotEmpty) {
      querySegments.add('serial=${Uri.encodeQueryComponent(serial)}');
    }

    final endpointUri = _apiBaseUri.resolve('/api/v1/cirrus');
    final uri = endpointUri.replace(query: querySegments.join('&'));

    final response = await http.delete(uri, headers: _authHeaders);
    if (response.statusCode < 200 || response.statusCode >= 300) {
      throw Exception('Failed to delete file (${response.statusCode})');
    }
  }

  static Future<void> moveFile(
    String oldPath,
    String newPath, {
    String? oldDeviceSerial,
    String? newDeviceSerial,
  }) async {
    final endpointUri = _apiBaseUri.resolve('/api/v1/cirrus');
    final requestBody = <String, String>{
      'oldFilePath': oldPath,
      'newFilePath': newPath,
    };

    final oldSerial = oldDeviceSerial?.trim() ?? '';
    if (oldSerial.isNotEmpty) {
      requestBody['oldDeviceSerial'] = oldSerial;
    }

    final newSerial = newDeviceSerial?.trim() ?? '';
    if (newSerial.isNotEmpty) {
      requestBody['newDeviceSerial'] = newSerial;
    }

    final body = jsonEncode(requestBody);

    final response = await http.put(
      endpointUri,
      headers: {'Content-Type': 'application/json', ..._authHeaders},
      body: body,
    );

    if (response.statusCode < 200 || response.statusCode >= 300) {
      throw Exception('Failed to move file (${response.statusCode})');
    }
  }

  static Future<void> createFolder(String folderPath, String folderName) async {
    final trimmedFolderPath = folderPath.trim();
    final endpointPath = trimmedFolderPath.isEmpty
        ? '/api/v1/cirrus/folder/'
        : _joinPaths('/api/v1/cirrus/folder', trimmedFolderPath);
    final endpointUri = _apiBaseUri.resolve(endpointPath);

    final request = http.MultipartRequest('POST', endpointUri);
    request.fields['folderName'] = folderName;
    request.headers.addAll(_authHeaders);

    final response = await request.send();
    if (response.statusCode < 200 || response.statusCode >= 300) {
      throw Exception('Failed to create folder (${response.statusCode})');
    }
  }

  static Future<http.StreamedResponse> uploadFilesFromFormData(
    String uploadPath,
    List<http.MultipartFile> formDataFiles, {
    String? serial,
  }) async {
    final uploadEndpointPath = _joinPaths('/api/v1/cirrus/upload', uploadPath);
    final endpointUri = _apiBaseUri.resolve(uploadEndpointPath);

    final serialValue = serial?.trim() ?? '';
    final uri = serialValue.isEmpty
        ? endpointUri
        : endpointUri.replace(queryParameters: {'serial': serialValue});

    final request = http.MultipartRequest('POST', uri);
    request.files.addAll(formDataFiles);
    request.headers.addAll(_authHeaders);

    final response = await request.send();
    if (response.statusCode < 200 || response.statusCode >= 300) {
      throw Exception('Failed to upload files (${response.statusCode})');
    }

    return response;
  }

  static Future<String?> saveFile(
    String filePath, {
    String? serial,
    String? fileName,
  }) async {
    final uri = _buildDownloadUri(filePath, serial: serial);
    final response = await http.get(uri, headers: _authHeaders);
    if (response.statusCode < 200 || response.statusCode >= 300) {
      throw Exception('Failed to download file (${response.statusCode})');
    }

    final resolvedName = _resolveDownloadFileName(
      response.headers['content-disposition'],
      preferredName: fileName,
      fallbackPath: filePath,
    );

    final bytes = Uint8List.fromList(response.bodyBytes);

    if (kIsWeb) {
      return web_download.saveBytesForDownload(bytes, resolvedName);
    }

    final params = SaveFileDialogParams(data: bytes, fileName: resolvedName);

    return FlutterFileDialog.saveFile(params: params);
  }

  static Future<Uint8List?> downloadFileBytes(
    String filePath, {
    String? serial,
    String? fileName,
  }) async {
    final uri = _buildDownloadUri(filePath, serial: serial);
    final response = await http.get(uri, headers: _authHeaders);
    if (response.statusCode < 200 || response.statusCode >= 300) {
      throw Exception('Failed to download file (${response.statusCode})');
    }

    return response.bodyBytes;
  }

  /// Download thumbnail bytes for the specified filePath using the thumbnails endpoint.
  /// Returns the raw bytes of the thumbnail image, or throws on non-success status codes.
  static Future<Uint8List?> downloadThumbnailBytes(
    String filePath, {
    String? serial,
  }) async {
    final uri = constructThumbnailUrl(filePath, serial: serial);
    final response = await http.get(uri, headers: _authHeaders);
    if (response.statusCode < 200 || response.statusCode >= 300) {
      throw Exception('Failed to download thumbnail (${response.statusCode})');
    }
    return response.bodyBytes;
  }

  static Uri _buildDownloadUri(String filePath, {String? serial}) {
    final querySegments = <String>[
      'filePath=${Uri.encodeQueryComponent(filePath)}',
    ];

    final serialValue = serial?.trim() ?? '';
    if (serialValue.isNotEmpty) {
      querySegments.add('serial=${Uri.encodeQueryComponent(serialValue)}');
    }

    final endpointUri = _apiBaseUri.resolve('/api/v1/cirrus/download');
    return endpointUri.replace(query: querySegments.join('&'));
  }

  static String _resolveDownloadFileName(
    String? contentDisposition, {
    String? preferredName,
    required String fallbackPath,
  }) {
    final explicitName = preferredName?.trim() ?? '';
    if (explicitName.isNotEmpty) {
      return explicitName;
    }

    final extractedName = _extractFileNameFromContentDisposition(
      contentDisposition,
    );
    if (extractedName != null && extractedName.isNotEmpty) {
      return extractedName;
    }

    final normalized = fallbackPath.trim();
    if (normalized.isEmpty) {
      return 'download';
    }

    final withoutTrailing = normalized.endsWith('/')
        ? normalized.substring(0, normalized.length - 1)
        : normalized;
    if (withoutTrailing.isEmpty) {
      return 'download';
    }

    final lastSlash = withoutTrailing.lastIndexOf('/');
    if (lastSlash < 0 || lastSlash == withoutTrailing.length - 1) {
      return withoutTrailing;
    }
    return withoutTrailing.substring(lastSlash + 1);
  }

  static String? _extractFileNameFromContentDisposition(String? headerValue) {
    if (headerValue == null || headerValue.trim().isEmpty) {
      return null;
    }

    final utf8Match = RegExp(
      r"filename\*=UTF-8''([^;]+)",
      caseSensitive: false,
    ).firstMatch(headerValue);
    if (utf8Match != null) {
      return Uri.decodeFull(utf8Match.group(1) ?? '').replaceAll('"', '');
    }

    final basicMatch = RegExp(
      r'filename="?([^";]+)"?',
      caseSensitive: false,
    ).firstMatch(headerValue);
    if (basicMatch != null) {
      return basicMatch.group(1)?.trim();
    }

    return null;
  }

  static String _normalizePath(String path) {
    final trimmed = path.trim();
    if (trimmed.isEmpty || trimmed == '/') {
      return '';
    }

    final withLeadingSlash = trimmed.startsWith('/') ? trimmed : '/$trimmed';
    if (withLeadingSlash.endsWith('/') && withLeadingSlash.length > 1) {
      return withLeadingSlash.substring(0, withLeadingSlash.length - 1);
    }
    return withLeadingSlash;
  }

  static String _toRootDir(String normalizedPath) {
    if (normalizedPath.isEmpty) {
      return '';
    }
    return normalizedPath.substring(1);
  }

  static String _joinPaths(String basePath, String appendPath) {
    final normalizedBase = basePath.endsWith('/')
        ? basePath.substring(0, basePath.length - 1)
        : basePath;
    final normalizedAppend = appendPath.trim();

    if (normalizedAppend.isEmpty) {
      return normalizedBase;
    }

    final strippedAppend = normalizedAppend.startsWith('/')
        ? normalizedAppend.substring(1)
        : normalizedAppend;
    return '$normalizedBase/$strippedAppend';
  }

  static Future<Map<String, dynamic>> getInstalledVersion() async {
    final endpointUri = _apiBaseUri.resolve('/api/v1/version');
    final response = await http.get(endpointUri, headers: _authHeaders);
    if (response.statusCode < 200 || response.statusCode >= 300) {
      throw Exception(
        'Failed to get installed version (${response.statusCode})',
      );
    }
    final decoded = jsonDecode(response.body);
    if (decoded is! Map) {
      throw Exception('Unexpected version response format');
    }
    return Map<String, dynamic>.from(decoded);
  }

  static Future<List<Map<String, dynamic>>> listAvailableVersions({
    bool all = false,
  }) async {
    final endpointUri = _apiBaseUri.resolve('/api/v1/version/available');
    final uri = all ? endpointUri.replace(query: 'all=true') : endpointUri;
    final response = await http.get(uri, headers: _authHeaders);
    if (response.statusCode < 200 || response.statusCode >= 300) {
      throw Exception(
        'Failed to list available versions (${response.statusCode})',
      );
    }
    final decoded = jsonDecode(response.body);
    if (decoded is! List) {
      throw Exception('Unexpected available versions response format');
    }
    return decoded
        .whereType<Map<String, dynamic>>()
        .map(Map<String, dynamic>.from)
        .toList(growable: false);
  }

  static Future<void> updateToLatest() async {
    final endpointUri = _apiBaseUri.resolve('/api/v1/version/latest');
    final response = await http.post(endpointUri, headers: _authHeaders);
    if (response.statusCode < 200 || response.statusCode >= 300) {
      throw Exception(
        'Failed to update to latest (${response.statusCode}): ${response.body}',
      );
    }
  }

  static Future<void> updateToVersion(String version) async {
    final endpointUri = _apiBaseUri.resolve('/api/v1/version/update');
    final body = jsonEncode({'version': version});
    final response = await http.post(
      endpointUri,
      headers: {'Content-Type': 'application/json', ..._authHeaders},
      body: body,
    );
    if (response.statusCode < 200 || response.statusCode >= 300) {
      throw Exception(
        'Failed to perform update (${response.statusCode}): ${response.body}',
      );
    }
  }
}
