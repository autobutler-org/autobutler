import 'package:autobutler/services/app_settings.dart';

/// Mixin providing a shared auth header helper for services that talk to the butler API.
///
/// Include this mixin in any service class that makes HTTP calls to the butler.
/// Use [authHeaders] to inject the session token into every request.
mixin AuthenticatedService {
  /// Returns an Authorization header map when a session token is set, empty otherwise.
  /// Spread into your http call headers: `headers: authHeaders`.
  Map<String, String> get authHeaders {
    final token = AppSettings.instance.sessionToken;
    if (token == null) return {};
    return {'Authorization': 'Bearer $token'};
  }
}
