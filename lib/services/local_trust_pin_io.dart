import 'dart:io';

import 'package:crypto/crypto.dart';
import 'package:quark/services/app_settings.dart';

/// The SHA-256 of [cert]'s DER encoding as lowercase hex: what a saved Quark
/// is pinned to (#2721).
String certificateSha256(X509Certificate cert) =>
    sha256.convert(cert.der).toString();

/// Whether [cert], which failed verification, is the certificate the saved
/// Quark at [address] is pinned to, pinning it when it is the first one seen.
///
/// The second half of every `badCertificateCallback` on a client built for
/// one Quark, after the local-trust policy has said the host may present a
/// self-signed certificate at all. [address] is the address the client was
/// built for, never the host the TLS layer reports, which on iOS can be the
/// mDNS-resolved address. See [AppSettings.trustCertificate].
///
/// IO only. The browser owns TLS on web, where no certificate callback runs
/// and nothing is pinned.
bool acceptsPinnedCertificate(X509Certificate cert, String address) =>
    AppSettings.instance.trustCertificate(address, certificateSha256(cert));

/// [acceptsPinnedCertificate] for a client that was not built for one Quark
/// and knows only the [host] and [port] the TLS layer reports. See
/// [AppSettings.trustReportedCertificate].
bool acceptsReportedPinnedCertificate(
  X509Certificate cert,
  String host,
  int port,
) => AppSettings.instance.trustReportedCertificate(
  host,
  port,
  certificateSha256(cert),
);
