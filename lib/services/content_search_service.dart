import 'dart:convert';

import 'package:flutter/foundation.dart';
import 'package:http/http.dart' as http;
import 'package:quark/services/app_settings.dart';
import 'package:quark/services/authenticated_service.dart';
import 'package:quark/services/shared_http_client.dart';
import 'package:quark/utils/content_search_config.dart';

/// Swapped by tests to answer searches without a network.
http.Client Function() contentSearchHttpClientFactory = () => sharedHttpClient;

/// A single full-text search result from the backend FTS5 index.
class ContentSearchResult {
  const ContentSearchResult({
    required this.deviceSerial,
    required this.relPath,
    required this.snippet,
  });

  /// Empty string means the internal (non-USB) device.
  final String deviceSerial;

  /// Path relative to the device's files root (e.g. "docs/meeting.qdoc").
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

/// Calls `GET /api/v0/files/search/content?q=<query>` and returns up to 50
/// results (the backend's default limit).
class ContentSearchService with AuthenticatedService {
  ContentSearchService._();
  static final ContentSearchService instance = ContentSearchService._();

  @override
  http.Client get httpClient => contentSearchHttpClientFactory();

  /// Insertion order is the LRU order — a hit reinserts at the end.
  static final Map<String, List<ContentSearchResult>> _recent = {};

  /// Searches in flight, so two identical queries share one request.
  static final Map<String, Future<List<ContentSearchResult>>> _inFlight = {};

  /// The memoized results for [query], or null when it has not been answered
  /// recently. Marks the entry most-recently-used.
  static List<ContentSearchResult>? recent(String query) {
    final q = query.trim();
    final results = _recent.remove(q);
    if (results == null) return null;
    _recent[q] = results;
    return results;
  }

  /// Forgets every memoized query and any answer still in flight.
  static void clearRecent() {
    _recent.clear();
    _inFlight.clear();
  }

  static void _remember(String query, List<ContentSearchResult> results) {
    _recent.remove(query);
    _recent[query] = results;
    while (_recent.length > ContentSearchConfig.recentQueryLimit) {
      _recent.remove(_recent.keys.first);
    }
  }

  /// Returns content-search results for [query].
  ///
  /// Never throws: on a transport error, a non-2xx status, or a body that is
  /// not a JSON array, it logs and returns an empty list so the caller can
  /// clear its loading state. Such failures are not memoized; a recent
  /// successful answer is returned without a request, and identical queries
  /// in flight share one.
  static Future<List<ContentSearchResult>> search(String query) {
    final q = query.trim();
    if (q.isEmpty) return Future.value(const []);

    final hit = recent(q);
    if (hit != null) return Future.value(hit);

    final existing = _inFlight[q];
    if (existing != null) return existing;

    late final Future<List<ContentSearchResult>> pending;
    pending = _fetch(q).then((results) {
      if (results == null) return const <ContentSearchResult>[];
      if (_inFlight[q] == pending) _remember(q, results);
      return results;
    });
    _inFlight[q] = pending;
    return pending.whenComplete(() {
      if (_inFlight[q] == pending) _inFlight.remove(q);
    });
  }

  /// Null on any failure. Note that an unmatched API path does not 404 —
  /// the server's SPA fallback answers 200 with `index.html`, so the body must
  /// be validated, not just the status code.
  static Future<List<ContentSearchResult>?> _fetch(String query) async {
    final uri = apiBaseUri.replace(
      path: '/api/v0/files/search/content',
      queryParameters: {'q': query},
    );
    try {
      final response = await instance.authenticatedGet(uri);
      if (response.statusCode < 200 || response.statusCode >= 300) {
        debugPrint('content search: $uri returned ${response.statusCode}');
        return null;
      }
      // WrapApiRoute writes the handler's payload directly (c.JSON(status,
      // resp.Data)), so a list endpoint answers with a bare JSON array — there
      // is no {"data": …} envelope to unwrap.
      final decoded = jsonDecode(response.body);
      if (decoded is! List) {
        debugPrint('content search: $uri returned ${decoded.runtimeType}');
        return null;
      }
      return decoded
          .whereType<Map<String, dynamic>>()
          .map(ContentSearchResult.fromJson)
          .toList(growable: false);
    } on FormatException catch (e) {
      // Reached when the SPA fallback serves HTML for an unmatched path.
      debugPrint('content search: $uri returned a non-JSON body ($e)');
      return null;
    } catch (e) {
      debugPrint('content search: $uri failed ($e)');
      return null;
    }
  }
}
