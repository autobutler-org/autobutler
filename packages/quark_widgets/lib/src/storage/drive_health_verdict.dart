import 'package:flutter/painting.dart';

import '../theme/quark_tokens.dart';

/// The health verdict the Quark assigned to one drive.
///
/// The widgets print [label]. They do not read SMART data or decide a
/// verdict. [unknown] is its own word, and it never shares [healthy]'s color.
enum DriveHealthVerdict {
  /// The Quark read the drive and found nothing that needs attention.
  healthy('Healthy'),

  /// The Quark wants the drive watched. Using it does not need an acknowledgment.
  watch('Watch'),

  /// The Quark says the drive should be replaced soon.
  replaceSoon('Replace soon'),

  /// The Quark could not read the drive's health.
  unknown('Unknown');

  /// Creates a verdict whose on-screen word is [label].
  const DriveHealthVerdict(this.label);

  /// The word on the badge and in the acknowledgment.
  final String label;
}

/// The status color for [verdict].
///
/// [DriveHealthVerdict.unknown] uses [QuarkTokens.secondaryForeground], so a
/// drive whose health could not be read does not wear the healthy color.
Color driveHealthVerdictColor(QuarkTokens tokens, DriveHealthVerdict verdict) {
  switch (verdict) {
    case DriveHealthVerdict.healthy:
      return tokens.success;
    case DriveHealthVerdict.watch:
      return tokens.warning;
    case DriveHealthVerdict.replaceSoon:
      return tokens.error;
    case DriveHealthVerdict.unknown:
      return tokens.secondaryForeground;
  }
}
