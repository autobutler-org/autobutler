import 'dart:convert';

import 'package:autobutler/services/app_settings.dart';
import 'package:autobutler/services/authenticated_service.dart';
import 'package:flutter/foundation.dart';

/// A single full-text search result from the backend FTS5 index.
class ContentSearchResult {
  const ContentSearchResult({
    required this.deviceSerial,
    required this.relPath,
    required this.snippet,
  });

  /// Empty string means the internal (non-USB) device.
  final String deviceSerial;

  /// Path relative to the device's CirrusDir root (e.g. "docs/meeting.abdoc").
  final String relPath;

  /// HTML fragment with matched terms wrapped in `<b>…</b>`.
  final String snippet;

  /// The filename portion of [relPath].
  String get filename {
    final idx = relPath.lastIndexOf('/');
    return idx < 0 ? relPath : relPath.substring(idx + 1);
  }

  /// Strips HTML tags from [snippet] to produce plain text.
  String get plainSnippet => snippet.replaceAll(RegExp(r'<[^>]*>'), '');

  factory ContentSearchResult.fromJson(Map<String, dynamic> json) {
    return ContentSearchResult(
      deviceSerial: (json['serial'] as String?) ?? '',
      relPath: (json['relPath'] as String?) ?? '',
      snippet: (json['snippet'] as String?) ?? '',
    );
  }
}

/// Calls `GET /api/v0/cirrus/search/content?q=<query>` and returns up to 50
/// results (the backend's default limit).
class ContentSearchService with AuthenticatedService {
  ContentSearchService._();
  static final ContentSearchService instance = ContentSearchService._();

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

  /// Returns content-search results for [query].
  ///
  /// Never throws: on a transport error, a non-2xx status, or a body that is
  /// not the expected JSON envelope, it logs and returns an empty list so the
  /// caller can clear its loading state. Note that an unmatched API path does
  /// not 404 — the server's SPA fallback answers 200 with `index.html`, so the
  /// body must be validated, not just the status code.
  static Future<List<ContentSearchResult>> search(String query) async {
    if (query.trim().isEmpty) return [];
    final uri = _apiBaseUri.replace(
      path: '/api/v0/cirrus/search/content',
      queryParameters: {'q': query.trim()},
    );
    try {
      final response = await instance.authenticatedGet(uri);
      if (response.statusCode < 200 || response.statusCode >= 300) {
        debugPrint('content search: $uri returned ${response.statusCode}');
        return [];
      }
      final decoded = jsonDecode(response.body);
      if (decoded is! Map<String, dynamic>) return [];
      final data = decoded['data'];
      if (data is! List) return [];
      return data
          .whereType<Map<String, dynamic>>()
          .map(ContentSearchResult.fromJson)
          .toList();
    } on FormatException catch (e) {
      // Reached when the SPA fallback serves HTML for an unmatched path.
      debugPrint('content search: $uri returned a non-JSON body ($e)');
      return [];
    } catch (e) {
      debugPrint('content search: $uri failed ($e)');
      return [];
    }
  }
}
