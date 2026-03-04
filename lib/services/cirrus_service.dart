import 'dart:convert';
import 'dart:io';
import 'dart:typed_data';

import 'package:autobutler/models/cirrus_file_node.dart';
import 'package:autobutler/services/app_settings.dart';
import 'package:flutter_file_dialog/flutter_file_dialog.dart';
import 'package:http/http.dart' as http;

class CirrusService {
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

    if (Platform.isAndroid && isLoopbackHost) {
      return uri.replace(host: '10.0.2.2');
    }

    return uri;
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

    final response = await http.get(uri);
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
    final response = await http.get(uri);
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

    final response = await http.delete(uri);
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
      headers: const {'Content-Type': 'application/json'},
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

    final response = await request.send();
    if (response.statusCode < 200 || response.statusCode >= 300) {
      throw Exception('Failed to create folder (${response.statusCode})');
    }
  }

  static Future<http.StreamedResponse> uploadFiles(
    String uploadPath,
    List<File> files, {
    String? serial,
  }) async {
    final formDataFiles = await Future.wait(
      files.map((file) => http.MultipartFile.fromPath('files', file.path)),
    );

    return uploadFilesFromFormData(uploadPath, formDataFiles, serial: serial);
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

    final response = await request.send();
    if (response.statusCode < 200 || response.statusCode >= 300) {
      throw Exception('Failed to upload files (${response.statusCode})');
    }

    return response;
  }

  static Future<String?> downloadFile(
    String filePath, {
    String? serial,
    String? fileName,
  }) async {
    final uri = _buildDownloadUri(filePath, serial: serial);
    final response = await http.get(uri);
    if (response.statusCode < 200 || response.statusCode >= 300) {
      throw Exception('Failed to download file (${response.statusCode})');
    }

    final resolvedName = _resolveDownloadFileName(
      response.headers['content-disposition'],
      preferredName: fileName,
      fallbackPath: filePath,
    );

    final params = SaveFileDialogParams(
      data: Uint8List.fromList(response.bodyBytes),
      fileName: resolvedName,
    );
    return FlutterFileDialog.saveFile(params: params);
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
}
