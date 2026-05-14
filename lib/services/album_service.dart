import 'dart:convert';

import 'package:autobutler/models/photo_album.dart';
import 'package:autobutler/services/app_settings.dart';
import 'package:autobutler/services/authenticated_service.dart';
import 'package:flutter/foundation.dart';
import 'package:http/http.dart' as http;

class AlbumService with AuthenticatedService {
  static final AlbumService _instance = AlbumService._();
  AlbumService._();
  static AlbumService get instance => _instance;

  static Map<String, String> get _authHeaders => instance.authHeaders;

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
    return resolved.replace(path: '/api/v1$path');
  }

  static Future<List<PhotoAlbum>> listAlbums({bool tree = false}) async {
    final uri = _apiUri(
      '/albums',
    ).replace(queryParameters: tree ? {'tree': 'true'} : null);
    final response = await http.get(uri, headers: _authHeaders);
    instance.checkUnauthorized(response);
    if (response.statusCode != 200) {
      throw Exception('Failed to load albums: ${response.statusCode}');
    }
    final List<dynamic> data = json.decode(response.body) as List<dynamic>;
    return data
        .map((e) => PhotoAlbum.fromJson(e as Map<String, dynamic>))
        .toList();
  }

  static Future<PhotoAlbum> getAlbum(int id) async {
    final response = await http.get(
      _apiUri('/albums/$id'),
      headers: _authHeaders,
    );
    instance.checkUnauthorized(response);
    if (response.statusCode != 200) {
      throw Exception('Album not found');
    }
    return PhotoAlbum.fromJson(
      json.decode(response.body) as Map<String, dynamic>,
    );
  }

  static Future<PhotoAlbum> createAlbum(String name, {int? parentId}) async {
    final body = <String, dynamic>{'name': name};
    if (parentId != null) body['parentId'] = parentId;
    final response = await http.post(
      _apiUri('/albums'),
      headers: {..._authHeaders, 'Content-Type': 'application/json'},
      body: json.encode(body),
    );
    instance.checkUnauthorized(response);
    if (response.statusCode != 201) {
      throw Exception('Failed to create album: ${response.statusCode}');
    }
    return PhotoAlbum.fromJson(
      json.decode(response.body) as Map<String, dynamic>,
    );
  }

  static Future<PhotoAlbum> renameAlbum(int id, String name) async {
    final response = await http.patch(
      _apiUri('/albums/$id/rename'),
      headers: {..._authHeaders, 'Content-Type': 'application/json'},
      body: json.encode({'name': name}),
    );
    instance.checkUnauthorized(response);
    if (response.statusCode != 200) {
      throw Exception('Failed to rename album');
    }
    return PhotoAlbum.fromJson(
      json.decode(response.body) as Map<String, dynamic>,
    );
  }

  static Future<PhotoAlbum> moveAlbum(int id, {int? parentId}) async {
    final response = await http.patch(
      _apiUri('/albums/$id/move'),
      headers: {..._authHeaders, 'Content-Type': 'application/json'},
      body: json.encode({'parentId': parentId}),
    );
    instance.checkUnauthorized(response);
    if (response.statusCode != 200) {
      throw Exception('Failed to move album');
    }
    return PhotoAlbum.fromJson(
      json.decode(response.body) as Map<String, dynamic>,
    );
  }

  static Future<void> deleteAlbum(int id) async {
    final response = await http.delete(
      _apiUri('/albums/$id'),
      headers: _authHeaders,
    );
    instance.checkUnauthorized(response);
    if (response.statusCode != 204) {
      throw Exception('Failed to delete album');
    }
  }

  static Future<List<PhotoAlbumItem>> listAlbumItems(int albumId) async {
    final response = await http.get(
      _apiUri('/albums/$albumId/items'),
      headers: _authHeaders,
    );
    instance.checkUnauthorized(response);
    if (response.statusCode != 200) {
      throw Exception('Failed to load album items');
    }
    final List<dynamic> data = json.decode(response.body) as List<dynamic>;
    return data
        .map((e) => PhotoAlbumItem.fromJson(e as Map<String, dynamic>))
        .toList();
  }

  static Future<PhotoAlbumItem> addPhotoToAlbum(
    int albumId, {
    required String deviceSerial,
    required String relPath,
  }) async {
    final response = await http.post(
      _apiUri('/albums/$albumId/items'),
      headers: {..._authHeaders, 'Content-Type': 'application/json'},
      body: json.encode({'deviceSerial': deviceSerial, 'relPath': relPath}),
    );
    instance.checkUnauthorized(response);
    if (response.statusCode != 201) {
      throw Exception('Failed to add photo to album');
    }
    return PhotoAlbumItem.fromJson(
      json.decode(response.body) as Map<String, dynamic>,
    );
  }

  static Future<void> removePhotoFromAlbum(
    int albumId, {
    required String deviceSerial,
    required String relPath,
  }) async {
    final response = await http.delete(
      _apiUri('/albums/$albumId/items'),
      headers: {..._authHeaders, 'Content-Type': 'application/json'},
      body: json.encode({'deviceSerial': deviceSerial, 'relPath': relPath}),
    );
    instance.checkUnauthorized(response);
    if (response.statusCode != 204) {
      throw Exception('Failed to remove photo from album');
    }
  }
}
