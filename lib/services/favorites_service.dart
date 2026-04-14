import 'dart:convert';

import 'package:autobutler/services/app_settings.dart';
import 'package:autobutler/services/authenticated_service.dart';
import 'package:flutter/foundation.dart';
import 'package:http/http.dart' as http;

class FavoriteStatus {
  final bool isFavorite;
  const FavoriteStatus({required this.isFavorite});
  factory FavoriteStatus.fromJson(Map<String, dynamic> json) =>
      FavoriteStatus(isFavorite: json['isFavorite'] as bool? ?? false);
}

class FavoritesService with AuthenticatedService {
  static final FavoritesService _instance = FavoritesService._();
  FavoritesService._();
  static FavoritesService get instance => _instance;

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

  /// Toggles favorite state. Returns the new isFavorite value.
  static Future<bool> toggle({required String relPath, String? serial}) async {
    final response = await http.post(
      _apiUri('/photos/favorite'),
      headers: {..._authHeaders, 'Content-Type': 'application/json'},
      body: json.encode({'relPath': relPath, 'deviceSerial': serial ?? ''}),
    );
    if (response.statusCode != 200) {
      throw Exception('Failed to toggle favorite: ${response.statusCode}');
    }
    final d = json.decode(response.body) as Map<String, dynamic>;
    return d['isFavorite'] as bool? ?? false;
  }

  /// Returns whether a photo is favorited.
  static Future<bool> isFavorite({
    required String relPath,
    String? serial,
  }) async {
    final uri = _apiUri('/photos/favorite').replace(
      queryParameters: {
        'relPath': relPath,
        if (serial != null && serial.isNotEmpty) 'serial': serial,
      },
    );
    final response = await http.get(uri, headers: _authHeaders);
    if (response.statusCode != 200) return false;
    final d = json.decode(response.body) as Map<String, dynamic>;
    return d['isFavorite'] as bool? ?? false;
  }
}
