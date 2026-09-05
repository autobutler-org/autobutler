import 'dart:io';

import 'package:flutter/foundation.dart';
import 'package:http/http.dart' as http;
import 'package:http/io_client.dart';
import 'package:quark/controllers/app_caches.dart';
import 'package:quark/services/app_settings.dart';
import 'package:quark/services/local_trust.dart';
import 'package:quark/utils/error_text.dart';

/// Thrown when an API call returns 401 — session expired or invalid.
class UnauthorizedException implements Exception {
  const UnauthorizedException();
  @override
  String toString() => 'Session expired. Please log in again.';
}

/// How long to wait for a TCP connection before giving up on the Quark.
///
/// Bounds only the connect phase, so it is safe on requests with a large body
/// (chunked uploads) — those are bounded separately by the upload manager's
/// own per-attempt timeout.
/// A host that is silent rather than actively refusing otherwise hangs for
/// however long the OS feels like waiting.
const Duration kConnectTimeout = Duration(seconds: 5);

/// Returns an [http.Client] that trusts self-signed certificates when the
/// active host is a local/LAN address (see [isLocalTrustHost]).
///
/// On web, the browser manages TLS trust natively and imposes its own connect
/// deadline, so the default client is returned unchanged.
http.Client buildLocalTrustHttpClient() {
  if (kIsWeb) return http.Client();

  final host = _extractHost(AppSettings.instance.activeHost);

  final inner = HttpClient()..connectionTimeout = kConnectTimeout;
  if (isLocalTrustHost(host)) {
    inner.badCertificateCallback = (cert, host, port) => true;
  }
  return IOClient(inner);
}

/// Extracts the hostname portion from a URL string (or returns the raw string
/// when it cannot be parsed as a URI).
String _extractHost(String? url) {
  if (url == null || url.isEmpty) return '';
  try {
    return Uri.parse(url).host;
  } catch (_) {
    return url;
  }
}

/// Mixin providing a shared auth header helper for services that talk to the quark API.
///
/// Include this mixin in any service class that makes HTTP calls to the quark.
/// Use [authHeaders] to inject the session token into every request.
/// Call [checkUnauthorized] after every response to handle session expiry.
mixin AuthenticatedService {
  /// Returns an Authorization header map when a session token is set, empty otherwise.
  /// Spread into your http call headers: `headers: authHeaders`.
  Map<String, String> get authHeaders {
    final token = AppSettings.instance.sessionToken;
    if (token == null) return {};
    return {'Authorization': 'Bearer $token'};
  }

  /// Checks if [response] is a 401 and, if so, clears the session token and
  /// throws [UnauthorizedException]. The router's redirect guard will pick up
  /// the cleared token on the next navigation and send the user to the login page.
  ///
  /// Call this after every authenticated API response:
  /// ```dart
  /// final response = await http.get(uri, headers: _authHeaders);
  /// checkUnauthorized(response);
  /// if (response.statusCode != 200) throw Exception('...');
  /// ```
  void checkUnauthorized(http.Response response) {
    if (response.statusCode == 401) {
      AppSettings.instance.setSessionToken(null);
      AppCaches.clearAll();
      throw const UnauthorizedException();
    }
  }

  /// Returns an HTTP client that trusts self-signed certs when the active host
  /// is a local/LAN address. Safe to call on every request; the returned client
  /// must be closed by the caller when no longer needed.
  http.Client get httpClient => buildLocalTrustHttpClient();

  /// Authenticated GET — injects auth headers and checks for 401 automatically.
  Future<http.Response> authenticatedGet(
    Uri uri, {
    Map<String, String>? headers,
  }) async {
    final client = httpClient;
    try {
      final response = await client.get(
        uri,
        headers: {...authHeaders, ...?headers},
      );
      checkUnauthorized(response);
      return response;
    } finally {
      client.close();
    }
  }

  /// Authenticated GET streamed straight onto disk, returning where it landed.
  ///
  /// The body never enters memory. A download used to arrive whole as
  /// `response.bodyBytes` and get copied a second time on its way to the file,
  /// so saving a large file cost twice its size in RAM on a phone (#1723).
  ///
  /// The caller owns the returned file and must [DownloadedFile.delete] it.
  /// Not available on web, which has no filesystem to stream onto.
  Future<DownloadedFile> authenticatedDownload(
    Uri uri, {
    Map<String, String>? headers,
  }) async {
    final client = httpClient;
    try {
      final request = http.Request('GET', uri)
        ..headers.addAll({...authHeaders, ...?headers});
      final response = await client.send(request);

      if (response.statusCode == 401) {
        AppSettings.instance.setSessionToken(null);
        throw const UnauthorizedException();
      }
      if (response.statusCode < 200 || response.statusCode >= 300) {
        throw ApiException(response.statusCode, 'Failed to download file');
      }

      final dir = await Directory.systemTemp.createTemp('quark_download_');
      final file = File('${dir.path}/download');
      final sink = file.openWrite();
      try {
        await response.stream.pipe(sink);
      } finally {
        await sink.close();
      }
      return DownloadedFile._(dir, file.path, response.headers);
    } finally {
      client.close();
    }
  }

  /// Authenticated POST — injects auth headers and checks for 401 automatically.
  Future<http.Response> authenticatedPost(
    Uri uri, {
    Map<String, String>? headers,
    Object? body,
  }) async {
    final client = httpClient;
    try {
      final response = await client.post(
        uri,
        headers: {...authHeaders, ...?headers},
        body: body,
      );
      checkUnauthorized(response);
      return response;
    } finally {
      client.close();
    }
  }

  /// Authenticated PATCH — injects auth headers and checks for 401 automatically.
  Future<http.Response> authenticatedPatch(
    Uri uri, {
    Map<String, String>? headers,
    Object? body,
  }) async {
    final client = httpClient;
    try {
      final response = await client.patch(
        uri,
        headers: {...authHeaders, ...?headers},
        body: body,
      );
      checkUnauthorized(response);
      return response;
    } finally {
      client.close();
    }
  }

  /// Authenticated DELETE — injects auth headers and checks for 401 automatically.
  Future<http.Response> authenticatedDelete(
    Uri uri, {
    Map<String, String>? headers,
    Object? body,
  }) async {
    final client = httpClient;
    try {
      final response = await client.delete(
        uri,
        headers: {...authHeaders, ...?headers},
        body: body,
      );
      checkUnauthorized(response);
      return response;
    } finally {
      client.close();
    }
  }

  /// Authenticated PUT — injects auth headers and checks for 401 automatically.
  Future<http.Response> authenticatedPut(
    Uri uri, {
    Map<String, String>? headers,
    Object? body,
  }) async {
    final client = httpClient;
    try {
      final response = await client.put(
        uri,
        headers: {...authHeaders, ...?headers},
        body: body,
      );
      checkUnauthorized(response);
      return response;
    } finally {
      client.close();
    }
  }
}

/// A download that landed on disk instead of in memory, plus the response
/// headers the caller still needs (Content-Disposition, for the file name).
class DownloadedFile {
  const DownloadedFile._(this._dir, this.path, this.headers);

  final Directory _dir;

  /// Path of the downloaded file. Valid until [delete].
  final String path;

  /// Response headers from the download.
  final Map<String, String> headers;

  /// Removes the temporary file and the directory holding it. Failure to clean
  /// up is not worth failing a completed download over, so it is swallowed.
  Future<void> delete() async {
    try {
      await _dir.delete(recursive: true);
    } catch (_) {
      // Best effort: the OS reclaims its own temp directory.
    }
  }
}
