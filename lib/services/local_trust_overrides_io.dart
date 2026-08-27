// IO implementation: applies the local-trust policy to every HttpClient the
// app creates, including ones we don't construct ourselves.
import 'dart:io';

import 'package:quark/services/app_settings.dart';
import 'package:quark/services/local_trust.dart';

/// Applies [isLocalTrustHost] to every [HttpClient] created in this isolate.
///
/// [buildLocalTrustHttpClient] only covers services that go through the
/// [AuthenticatedService] mixin. Several services call the top-level
/// `http.get`/`http.post` helpers, and widgets like `Image.network` create
/// their own clients — all of which would otherwise reject the quark's
/// self-signed certificate. Installing an override catches them all in one
/// place.
class LocalTrustHttpOverrides extends HttpOverrides {
  LocalTrustHttpOverrides();

  @override
  HttpClient createHttpClient(SecurityContext? context) {
    return super.createHttpClient(context)
      ..badCertificateCallback = _shouldTrust;
  }

  /// Trusts a bad certificate only when the connection targets the local
  /// network.
  ///
  /// [host] is checked first, but on iOS it can arrive as an mDNS-resolved
  /// address that doesn't match what the user configured, so the configured
  /// active host is accepted as well.
  static bool _shouldTrust(X509Certificate cert, String host, int port) {
    if (isLocalTrustHost(host)) return true;
    return isLocalTrustHost(_activeHost());
  }

  static String? _activeHost() {
    final configured = AppSettings.instance.activeHost;
    if (configured == null || configured.isEmpty) return null;
    try {
      return Uri.parse(configured).host;
    } catch (_) {
      return configured;
    }
  }
}

void installLocalTrustHttpOverrides() {
  HttpOverrides.global = LocalTrustHttpOverrides();
}
