import 'package:flutter/painting.dart';

import '../theme/quark_tokens.dart';

/// How protected the files are, as the Quark reported it.
///
/// The caller passes the level through. Nothing here counts drives or decides
/// that a backup is recent.
enum DataProtectionLevel {
  /// No snapshot-backup drive has a recent completed backup.
  unprotected('Unprotected'),

  /// One such drive.
  basic('Basic'),

  /// Two or more such drives.
  protected('Protected');

  /// Creates a level whose short name is [label].
  const DataProtectionLevel(this.label);

  /// The word on the meter, the badge, and the card.
  final String label;
}

/// The status color for [level]: [QuarkTokens.error], [QuarkTokens.warning],
/// or [QuarkTokens.success].
Color dataProtectionLevelColor(QuarkTokens tokens, DataProtectionLevel level) {
  switch (level) {
    case DataProtectionLevel.unprotected:
      return tokens.error;
    case DataProtectionLevel.basic:
      return tokens.warning;
    case DataProtectionLevel.protected:
      return tokens.success;
  }
}
