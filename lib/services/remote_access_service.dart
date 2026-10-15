import 'dart:convert';

import 'package:quark/services/app_settings.dart';
import 'package:quark/services/authenticated_service.dart';
import 'package:quark/utils/error_text.dart';

class RemoteAccessStatus {
  /// The Quark's setting: remote access was switched on.
  final bool enabled;

  /// Whether the Tailscale node has actually joined the tailnet. An enabled
  /// Quark that has not is still connecting, or failing — see [error].
  final bool connected;

  /// Set only when [connected].
  final String? remoteUrl;

  /// The Quark's last failure to start remote access, for logs only. It is a
  /// Go error's text, so the UI shows [Errors.remoteAccessFailing] instead.
  final String? error;

  const RemoteAccessStatus({
    required this.enabled,
    this.connected = false,
    this.remoteUrl,
    this.error,
  });

  factory RemoteAccessStatus.fromJson(Map<String, dynamic> json) =>
      RemoteAccessStatus(
        enabled: json['enabled'] as bool? ?? false,
        connected: json['connected'] as bool? ?? false,
        remoteUrl: json['remoteUrl'] as String?,
        error: json['error'] as String?,
      );
}

/// What `POST /api/v0/settings/remote-access/devices` returns so a phone can
/// join this Quark's tailnet (#2359, #1881).
class PairedDevice {
  /// A single-use pre-auth key in the Quark's household.
  final String authKey;

  /// The coordination server the phone registers with.
  final String controlUrl;

  /// The Quark's own URL on the tailnet, which the phone's proxy dials.
  final String quarkAddress;

  /// Creates a pairing result.
  const PairedDevice({
    required this.authKey,
    required this.controlUrl,
    required this.quarkAddress,
  });

  /// Parses the pairing response. A missing field throws [FormatException];
  /// that text is for a log, and the caller does not show it.
  factory PairedDevice.fromJson(Map<String, dynamic> json) {
    final authKey = json['authKey'] as String? ?? '';
    final controlUrl = json['controlUrl'] as String? ?? '';
    final quarkAddress = json['quarkAddress'] as String? ?? '';
    if (authKey.isEmpty || controlUrl.isEmpty || quarkAddress.isEmpty) {
      throw const FormatException('pairing response is missing a field');
    }
    return PairedDevice(
      authKey: authKey,
      controlUrl: controlUrl,
      quarkAddress: quarkAddress,
    );
  }
}

/// Calls `/api/v0/settings/remote-access` to read, enable and disable remote access.
class RemoteAccessService with AuthenticatedService {
  static final RemoteAccessService instance = RemoteAccessService._();
  RemoteAccessService._();

  static Map<String, String> get _authHeaders => instance.authHeaders;

  static Future<RemoteAccessStatus> getStatus() async {
    final uri = apiBaseUri.resolve('/api/v0/settings/remote-access');
    final response = await sharedHttpClient.get(uri, headers: _authHeaders);
    if (response.statusCode < 200 || response.statusCode >= 300) {
      throw ApiException(
        response.statusCode,
        'Failed to get remote access status',
      );
    }
    final json = jsonDecode(response.body) as Map<String, dynamic>;
    return RemoteAccessStatus.fromJson(json);
  }

  /// Asks the Quark, over the home network, for a key this phone can join with.
  static Future<PairedDevice> pairDevice() async {
    final uri = apiBaseUri.resolve('/api/v0/settings/remote-access/devices');
    final response = await sharedHttpClient.post(uri, headers: _authHeaders);
    if (response.statusCode < 200 || response.statusCode >= 300) {
      // The status is the whole signal: 409 and 503 are retried, and every
      // other code is shown through Errors. The body is a log, not copy.
      throw ApiException(response.statusCode, 'Failed to pair this device');
    }
    final json = jsonDecode(response.body) as Map<String, dynamic>;
    return PairedDevice.fromJson(json);
  }

  static Future<RemoteAccessStatus> enable() async {
    final uri = apiBaseUri.resolve('/api/v0/settings/remote-access');
    final response = await sharedHttpClient.post(
      uri,
      headers: {'Content-Type': 'application/json', ..._authHeaders},
      body: jsonEncode(<String, dynamic>{}),
    );
    if (response.statusCode < 200 || response.statusCode >= 300) {
      final body = jsonDecode(response.body) as Map<String, dynamic>?;
      throwApiError(
        response.statusCode,
        body?['error'],
        'Failed to enable remote access',
      );
    }
    final json = jsonDecode(response.body) as Map<String, dynamic>;
    return RemoteAccessStatus.fromJson(json);
  }

  static Future<RemoteAccessStatus> disable() async {
    final uri = apiBaseUri.resolve('/api/v0/settings/remote-access');
    final response = await sharedHttpClient.delete(uri, headers: _authHeaders);
    if (response.statusCode < 200 || response.statusCode >= 300) {
      final body = jsonDecode(response.body) as Map<String, dynamic>?;
      throwApiError(
        response.statusCode,
        body?['error'],
        'Failed to disable remote access',
      );
    }
    final json = jsonDecode(response.body) as Map<String, dynamic>;
    return RemoteAccessStatus.fromJson(json);
  }
}
