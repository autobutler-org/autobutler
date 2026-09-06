import 'dart:io';

import 'package:flutter/foundation.dart';
import 'package:http/http.dart' as http;
import 'package:http/io_client.dart';
import 'package:quark/services/app_settings.dart';
import 'package:quark/services/local_trust.dart';

/// How long to wait for a TCP connection before giving up on the Quark.
///
/// Bounds only the connect phase, so it is safe on requests with a large body
/// (chunked uploads) — those are bounded separately by the upload manager's
/// own per-attempt timeout.
/// A host that is silent rather than actively refusing otherwise hangs for
/// however long the OS feels like waiting.
const Duration kConnectTimeout = Duration(seconds: 5);

/// The one [http.Client] every API call to the active host goes out through.
///
/// Kept open for the life of the session so requests reuse the pooled
/// connections rather than paying a TCP connect and TLS handshake apiece
/// (#1782). Rebuilt when the active host changes. Callers must not close it.
http.Client get sharedHttpClient => SharedHttpClient.instance.client;

/// Holds the client behind [sharedHttpClient].
///
/// The client trusts self-signed certificates when the active host is a
/// local/LAN address (see [isLocalTrustHost]) and is handed back unchanged
/// until [AppSettings.activeHost] moves. Then the old one is closed — after
/// its in-flight requests finish — and the next access builds a fresh one
/// whose trust decision matches the new host. It is also rebuilt when
/// [HttpOverrides.current] differs from the one it was created under, since an
/// [HttpClient] binds to its overrides at construction.
///
/// On web, the browser manages TLS trust natively and imposes its own connect
/// deadline, so one default client serves every host.
class SharedHttpClient {
  SharedHttpClient._();

  static final SharedHttpClient instance = SharedHttpClient._();

  http.Client? _client;
  HttpClient? _inner;
  String? _host;
  HttpOverrides? _overrides;

  http.Client get client {
    if (kIsWeb) return _client ??= http.Client();

    final host = AppSettings.instance.activeHost;
    final overrides = HttpOverrides.current;
    final current = _client;
    if (current != null && host == _host && identical(overrides, _overrides)) {
      return current;
    }

    _inner?.close();
    _host = host;
    _overrides = overrides;
    final inner = HttpClient()..connectionTimeout = kConnectTimeout;
    if (isLocalTrustHost(_extractHost(host))) {
      inner.badCertificateCallback = (cert, host, port) => true;
    }
    _inner = inner;
    return _client = IOClient(inner);
  }

  /// Closes the current client so the next access builds a new one.
  void reset() {
    if (kIsWeb) {
      _client?.close();
    } else {
      _inner?.close();
    }
    _client = null;
    _inner = null;
    _host = null;
    _overrides = null;
  }
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
