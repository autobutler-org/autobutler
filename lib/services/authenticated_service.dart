import 'package:autobutler/services/app_settings.dart';
import 'package:http/http.dart' as http;

/// Thrown when an API call returns 401 — session expired or invalid.
class UnauthorizedException implements Exception {
  const UnauthorizedException();
  @override
  String toString() => 'Session expired. Please log in again.';
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

  /// Authenticated GET — injects auth headers and checks for 401 automatically.
  Future<http.Response> authenticatedGet(
    Uri uri, {
    Map<String, String>? headers,
  }) async {
    final response = await http.get(
      uri,
      headers: {...authHeaders, ...?headers},
    );
    checkUnauthorized(response);
    return response;
  }

  /// Authenticated POST — injects auth headers and checks for 401 automatically.
  Future<http.Response> authenticatedPost(
    Uri uri, {
    Map<String, String>? headers,
    Object? body,
  }) async {
    final response = await http.post(
      uri,
      headers: {...authHeaders, ...?headers},
      body: body,
    );
    checkUnauthorized(response);
    return response;
  }

  /// Authenticated PATCH — injects auth headers and checks for 401 automatically.
  Future<http.Response> authenticatedPatch(
    Uri uri, {
    Map<String, String>? headers,
    Object? body,
  }) async {
    final response = await http.patch(
      uri,
      headers: {...authHeaders, ...?headers},
      body: body,
    );
    checkUnauthorized(response);
    return response;
  }

  /// Authenticated DELETE — injects auth headers and checks for 401 automatically.
  Future<http.Response> authenticatedDelete(
    Uri uri, {
    Map<String, String>? headers,
    Object? body,
  }) async {
    final response = await http.delete(
      uri,
      headers: {...authHeaders, ...?headers},
      body: body,
    );
    checkUnauthorized(response);
    return response;
  }

  /// Authenticated PUT — injects auth headers and checks for 401 automatically.
  Future<http.Response> authenticatedPut(
    Uri uri, {
    Map<String, String>? headers,
    Object? body,
  }) async {
    final response = await http.put(
      uri,
      headers: {...authHeaders, ...?headers},
      body: body,
    );
    checkUnauthorized(response);
    return response;
  }
}
