import 'dart:io';

import 'package:autobutler/services/app_settings.dart';
import 'package:flutter/foundation.dart';
import 'package:http/http.dart' as http;
import 'package:http/io_client.dart';

/// Thrown when an API call returns 401 — session expired or invalid.
class UnauthorizedException implements Exception {
  const UnauthorizedException();
  @override
  String toString() => 'Session expired. Please log in again.';
}

/// Returns an [http.Client] that trusts self-signed certificates when the
/// active host is a local/LAN address (localhost, 127.0.0.1, or RFC-1918 IPs).
///
/// On web, the browser manages TLS trust natively, so the default client is
/// returned unchanged.
http.Client buildLocalTrustHttpClient() {
  if (kIsWeb) return http.Client();

  final host = _extractHost(AppSettings.instance.activeHost);
  final isLocal = host == 'localhost' ||
      host == '127.0.0.1' ||
      host == '::1' ||
      host == '10.0.2.2' || // Android emulator loopback
      RegExp(r'^(192\.168|10\.|172\.(1[6-9]|2[0-9]|3[01]))\.').hasMatch(host);

  if (isLocal) {
    final inner = HttpClient()
      ..badCertificateCallback = (cert, host, port) => true;
    return IOClient(inner);
  }
  return http.Client();
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

/// Mixin providing a shared auth header helper for services that talk to the butler API.
///
/// Include this mixin in any service class that makes HTTP calls to the butler.
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
