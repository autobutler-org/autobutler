// IO implementation: trusts self-signed certs from local hosts.
import 'dart:io';

import 'package:web_socket_channel/io.dart';
import 'package:web_socket_channel/web_socket_channel.dart';

/// Returns true when [host] is a loopback or RFC-1918 address.
bool _isLocalHost(String host) {
  return host == 'localhost' ||
      host == '127.0.0.1' ||
      host == '::1' ||
      host == '10.0.2.2' || // Android emulator loopback
      RegExp(r'^(192\.168|10\.|172\.(1[6-9]|2[0-9]|3[01]))\.').hasMatch(host);
}

/// Connects to [uri] over WebSocket, trusting self-signed certificates when the
/// configured host is a local/LAN address (localhost, loopback, or RFC-1918).
///
/// Decision is made on [uri.host] **before** the TLS handshake — the same
/// approach used by [buildLocalTrustHttpClient] for HTTP. This avoids the iOS
/// bug where [HttpClient.badCertificateCallback] receives a hostname that
/// doesn't match the IP stored in AppSettings (e.g. mDNS resolution or SNI
/// mismatch), causing the callback to return false and the handshake to fail.
WebSocketChannel connectLocalTrustWs(Uri uri) {
  if (!_isLocalHost(uri.host)) {
    // Non-local host: use standard TLS verification.
    return IOWebSocketChannel.connect(uri);
  }

  // Local host: trust the self-signed cert unconditionally, consistent with
  // buildLocalTrustHttpClient() in authenticated_service.dart.
  final httpClient = HttpClient()
    ..badCertificateCallback = (cert, host, port) => true;
  return IOWebSocketChannel.connect(uri, customClient: httpClient);
}
