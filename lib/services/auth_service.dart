import 'dart:convert';

import 'package:autobutler/services/app_settings.dart';
import 'package:http/http.dart' as http;

/// Result of a successful [AuthService.checkStatus] call.
class AuthStatus {
  /// Whether the butler has been set up with a local account.
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

/// Communicates with the butler auth API.
class AuthService {
  static Uri get _baseUri {
    final configured = AppSettings.instance.activeHost;
    final base =
        configured ??
        const String.fromEnvironment(
          'API_BASE_URL',
          defaultValue: 'http://localhost:8080',
        );
    return Uri.parse(base);
  }

  /// Checks whether initial setup has been completed on the butler.
  static Future<AuthStatus> checkStatus() async {
    final uri = _baseUri.resolve('/api/v1/auth/status');
    final response = await http.get(uri);
    if (response.statusCode < 200 || response.statusCode >= 300) {
      throw Exception('Failed to check auth status (${response.statusCode})');
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
    final uri = _baseUri.resolve('/api/v1/auth/setup');
    final response = await http.post(
      uri,
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode({'username': username, 'password': password}),
    );
    if (response.statusCode < 200 || response.statusCode >= 300) {
      final body = _tryDecodeError(response.body);
      throw Exception(body ?? 'Setup failed (${response.statusCode})');
    }
    final body = jsonDecode(response.body) as Map<String, dynamic>;
    final token = body['token'] as String;
    final phrase = body['recoveryPhrase'] as String;
    AppSettings.instance.setSessionToken(token);
    return SetupResult(sessionToken: token, recoveryPhrase: phrase);
  }

  /// Authenticates with username and password, returns a session token.
  static Future<LoginResult> login({
    required String username,
    required String password,
  }) async {
    final uri = _baseUri.resolve('/api/v1/auth/login');
    final response = await http.post(
      uri,
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode({'username': username, 'password': password}),
    );
    if (response.statusCode == 401) {
      throw Exception('Invalid username or password.');
    }
    if (response.statusCode < 200 || response.statusCode >= 300) {
      final body = _tryDecodeError(response.body);
      throw Exception(body ?? 'Login failed (${response.statusCode})');
    }
    final body = jsonDecode(response.body) as Map<String, dynamic>;
    final token = body['token'] as String;
    AppSettings.instance.setSessionToken(token);
    return LoginResult(sessionToken: token);
  }

  /// Resets the password using the recovery phrase and returns a new session.
  static Future<LoginResult> recover({
    required String recoveryPhrase,
    required String newPassword,
  }) async {
    final uri = _baseUri.resolve('/api/v1/auth/recover');
    final response = await http.post(
      uri,
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode({
        'recoveryPhrase': recoveryPhrase,
        'newPassword': newPassword,
      }),
    );
    if (response.statusCode < 200 || response.statusCode >= 300) {
      final body = _tryDecodeError(response.body);
      throw Exception(body ?? 'Recovery failed (${response.statusCode})');
    }
    final body = jsonDecode(response.body) as Map<String, dynamic>;
    final token = body['token'] as String;
    AppSettings.instance.setSessionToken(token);
    return LoginResult(sessionToken: token);
  }

  /// Logs out — clears the in-memory session token and notifies the server.
  static Future<void> logout() async {
    final token = AppSettings.instance.sessionToken;
    AppSettings.instance.setSessionToken(null);
    if (token == null) return;
    try {
      final uri = _baseUri.resolve('/api/v1/auth/logout');
      await http.post(uri, headers: {'Authorization': 'Bearer $token'});
    } catch (_) {
      // Best-effort — token is already cleared locally.
    }
  }

  static String? _tryDecodeError(String body) {
    try {
      final decoded = jsonDecode(body) as Map<String, dynamic>;
      return decoded['error'] as String? ?? decoded['message'] as String?;
    } catch (_) {
      return null;
    }
  }
}
