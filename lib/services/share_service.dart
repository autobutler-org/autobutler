import 'dart:convert';

import 'package:autobutler/services/app_settings.dart';
import 'package:autobutler/services/authenticated_service.dart';
import 'package:flutter/foundation.dart';

/// One public share link as returned by the /shares API.
class ShareLink {
  const ShareLink({
    required this.id,
    required this.filePath,
    required this.urlPath,
    required this.passwordProtected,
    required this.expired,
    required this.accessCount,
    this.deviceSerial = '',
    this.expiresAt,
    this.lastAccessAt,
  });

  final String id;
  final String filePath;
  final String deviceSerial;

  /// Server-relative public URL, e.g. `/api/v1/public/shares/(token)`.
  final String urlPath;
  final bool passwordProtected;
  final bool expired;
  final int accessCount;
  final DateTime? expiresAt;
  final DateTime? lastAccessAt;

  factory ShareLink.fromJson(Map<String, dynamic> json) => ShareLink(
    id: json['id'] as String? ?? '',
    filePath: json['filePath'] as String? ?? '',
    deviceSerial: json['deviceSerial'] as String? ?? '',
    urlPath: json['urlPath'] as String? ?? '',
    passwordProtected: json['passwordProtected'] as bool? ?? false,
    expired: json['expired'] as bool? ?? false,
    accessCount: (json['accessCount'] as num?)?.toInt() ?? 0,
    expiresAt: json['expiresAt'] != null
        ? DateTime.tryParse(json['expiresAt'] as String)
        : null,
    lastAccessAt: json['lastAccessAt'] != null
        ? DateTime.tryParse(json['lastAccessAt'] as String)
        : null,
  );

  /// Absolute URL a recipient can open, based on the configured host (or the
  /// web app's own origin as a fallback).
  String get fullUrl => '${ShareService.publicBaseUrl()}$urlPath';
}

class ShareService with AuthenticatedService {
  static final ShareService _instance = ShareService._();
  ShareService._();
  static ShareService get instance => _instance;

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

  /// Base URL used when handing a link to a recipient. Unlike [_apiUri], no
  /// Android-emulator loopback rewrite — the link is for someone else's
  /// device, not this app.
  static String publicBaseUrl() {
    final configured = AppSettings.instance.activeHost;
    if (configured != null && configured.isNotEmpty) {
      return configured.replaceAll(RegExp(r'/+$'), '');
    }
    if (kIsWeb) return Uri.base.origin;
    return const String.fromEnvironment(
      'API_BASE_URL',
      defaultValue: 'http://localhost:8080',
    );
  }

  /// Creates a share link for [filePath]. [expiresInHours] of 0 means the
  /// link never expires; a non-empty [password] protects it.
  static Future<ShareLink> create({
    required String filePath,
    String? serial,
    int expiresInHours = 0,
    String password = '',
  }) async {
    final response = await instance.authenticatedPost(
      _apiUri('/shares'),
      headers: {'Content-Type': 'application/json'},
      body: json.encode({
        'filePath': filePath,
        'deviceSerial': serial ?? '',
        'expiresInHours': expiresInHours,
        'password': password,
      }),
    );
    if (response.statusCode != 200) {
      throw Exception('Failed to create share link: ${response.statusCode}');
    }
    return ShareLink.fromJson(
      json.decode(response.body) as Map<String, dynamic>,
    );
  }

  /// Returns all share links, newest first.
  static Future<List<ShareLink>> list() async {
    final response = await instance.authenticatedGet(_apiUri('/shares'));
    if (response.statusCode != 200) {
      throw Exception('Failed to load share links: ${response.statusCode}');
    }
    final decoded = json.decode(response.body) as List<dynamic>;
    return decoded
        .map((item) => ShareLink.fromJson(item as Map<String, dynamic>))
        .toList();
  }

  /// Revokes a share link. The public URL stops working immediately.
  static Future<void> revoke(String id) async {
    final response = await instance.authenticatedDelete(_apiUri('/shares/$id'));
    if (response.statusCode != 200) {
      throw Exception('Failed to revoke share link: ${response.statusCode}');
    }
  }
}
