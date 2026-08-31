import 'dart:async';
import 'dart:convert';

import 'package:flutter/foundation.dart';
import 'package:http/http.dart' as http;
import 'package:quark/services/app_settings.dart';
import 'package:quark/services/authenticated_service.dart';
import 'package:quark/utils/error_text.dart';

/// Result of a successful [AuthService.checkStatus] call.
class AuthStatus {
  /// Whether the quark has been set up with a local account.
  final bool setupComplete;

  const AuthStatus({required this.setupComplete});
}

/// Result of [AuthService.setup] — shown once, must be surfaced to the user.
class SetupResult {
  final String sessionToken;

  /// Recovery phrase shown exactly once. Store it somewhere safe.
  final String recoveryPhrase;

  const SetupResult({required this.sessionToken, required this.recoveryPhrase});
}

/// Result of [AuthService.login].
class LoginResult {
  final String sessionToken;
  const LoginResult({required this.sessionToken});
}

/// How long an auth request may go unanswered before the Quark counts as
/// unreachable.
///
/// These all hit a local device over a LAN and return a few bytes, so anything
/// slower than this is a dead host, not a slow one. Bounds the whole request,
/// not just the connect phase, so a host that accepts the connection and then
/// goes silent still fails fast. [isQuarkUnreachableError] treats the resulting
/// [TimeoutException] as unreachable and routes to the disconnected UI.
const Duration kAuthRequestTimeout = Duration(seconds: 5);

/// Builds the client every auth call goes out through. Overridable in tests.
@visibleForTesting
http.Client Function() authHttpClientFactory = buildLocalTrustHttpClient;

/// Communicates with the quark auth API.
class AuthService {
  static Uri get _baseUri => Uri.parse(apiBaseUrl);

  /// Checks whether initial setup has been completed on the quark.
  static Future<AuthStatus> checkStatus() async {
    final uri = _baseUri.resolve('/api/v0/auth/status');
    final client = authHttpClientFactory();
    final http.Response response;
    try {
      response = await client.get(uri).timeout(kAuthRequestTimeout);
    } finally {
      client.close();
    }
    if (response.statusCode < 200 || response.statusCode >= 300) {
      throw ApiException(response.statusCode, 'Failed to check auth status');
    }
    final body = jsonDecode(response.body) as Map<String, dynamic>;
    return AuthStatus(setupComplete: body['setup'] as bool? ?? false);
  }

  /// Creates the owner account on first boot.
  /// Returns a [SetupResult] containing the session token and recovery phrase.
  /// The recovery phrase is shown exactly once — the caller must surface it.
  static Future<SetupResult> setup({
    required String username,
    required String password,
  }) async {
    final uri = _baseUri.resolve('/api/v0/auth/setup');
    final client = authHttpClientFactory();
    final http.Response response;
    try {
      response = await client
          .post(
            uri,
            headers: {'Content-Type': 'application/json'},
            body: jsonEncode({'username': username, 'password': password}),
          )
          .timeout(kAuthRequestTimeout);
    } finally {
      client.close();
    }
    if (response.statusCode < 200 || response.statusCode >= 300) {
      final body = _tryDecodeError(response.body);
      throwApiError(response.statusCode, body, 'Setup failed');
    }
    final body = jsonDecode(response.body) as Map<String, dynamic>;
    final token = body['token'] as String;
    final phrase = body['recoveryPhrase'] as String;
    await AppSettings.instance.setSessionToken(token);
    return SetupResult(sessionToken: token, recoveryPhrase: phrase);
  }

  /// Authenticates with username and password, returns a session token.
  static Future<LoginResult> login({
    required String username,
    required String password,
  }) async {
    final uri = _baseUri.resolve('/api/v0/auth/login');
    final client = authHttpClientFactory();
    final http.Response response;
    try {
      response = await client
          .post(
            uri,
            headers: {'Content-Type': 'application/json'},
            body: jsonEncode({'username': username, 'password': password}),
          )
          .timeout(kAuthRequestTimeout);
    } finally {
      client.close();
    }
    if (response.statusCode == 401) {
      throw const MessageException('Invalid username or password.');
    }
    if (response.statusCode < 200 || response.statusCode >= 300) {
      final body = _tryDecodeError(response.body);
      throwApiError(response.statusCode, body, 'Login failed');
    }
    final body = jsonDecode(response.body) as Map<String, dynamic>;
    final token = body['token'] as String;
    await AppSettings.instance.setSessionToken(token);
    return LoginResult(sessionToken: token);
  }

  /// Resets the password using the recovery phrase and returns a new session.
  static Future<LoginResult> recover({
    required String recoveryPhrase,
    required String newPassword,
  }) async {
    final uri = _baseUri.resolve('/api/v0/auth/recover');
    final client = authHttpClientFactory();
    final http.Response response;
    try {
      response = await client
          .post(
            uri,
            headers: {'Content-Type': 'application/json'},
            body: jsonEncode({
              'recoveryPhrase': recoveryPhrase,
              'newPassword': newPassword,
            }),
          )
          .timeout(kAuthRequestTimeout);
    } finally {
      client.close();
    }
    if (response.statusCode < 200 || response.statusCode >= 300) {
      final body = _tryDecodeError(response.body);
      throwApiError(response.statusCode, body, 'Recovery failed');
    }
    final body = jsonDecode(response.body) as Map<String, dynamic>;
    final token = body['token'] as String;
    await AppSettings.instance.setSessionToken(token);
    return LoginResult(sessionToken: token);
  }

  /// Logs out — clears the in-memory session token and notifies the server.
  static Future<void> logout() async {
    final token = AppSettings.instance.sessionToken;
    await AppSettings.instance.setSessionToken(null);
    if (token == null) return;
    try {
      final uri = _baseUri.resolve('/api/v0/auth/logout');
      final client = authHttpClientFactory();
      try {
        await client
            .post(uri, headers: {'Authorization': 'Bearer $token'})
            .timeout(kAuthRequestTimeout);
      } finally {
        client.close();
      }
    } catch (_) {
      // Best-effort — token is already cleared locally.
    }
  }

  static String? _tryDecodeError(String body) {
    try {
      final decoded = jsonDecode(body) as Map<String, dynamic>;
      return decoded['error'] as String? ?? decoded['message'] as String?;
    } catch (_) {
      debugPrint('[auth_service.dart] Error in catch block');
      return null;
    }
  }
}
