// cspell:words reallocated celsius
import 'drive_health_verdict.dart';

/// One drive's health, as the Quark reported it.
///
/// Every field except [verdict] is omitted when the Quark did not send it.
/// The detail panel prints those fields and does not invent a manufacture
/// year from [powerOnHours].
class DriveHealth {
  /// Creates a health report whose verdict is [verdict].
  const DriveHealth({
    required this.verdict,
    this.reason,
    this.smartStatus,
    this.reallocatedSectors,
    this.pendingSectors,
    this.powerOnHours,
    this.temperatureCelsius,
    this.media,
    this.linkSpeed,
    this.speedClass,
  });

  /// The verdict the Quark assigned.
  final DriveHealthVerdict verdict;

  /// The Quark's one-sentence reason. Null when it sent none.
  final String? reason;

  /// Overall SMART status, such as `passed` or `failed`. Null when unread.
  final String? smartStatus;

  /// Reallocated sector count. Null when the Quark sent none.
  final int? reallocatedSectors;

  /// Pending sector count. Null when the Quark sent none.
  final int? pendingSectors;

  /// Hours the drive has been powered on. Null when the Quark sent none.
  final int? powerOnHours;

  /// Drive temperature in Celsius. Null when the Quark sent none.
  final int? temperatureCelsius;

  /// `spinning` or `solid-state`, or another word the Quark sent.
  /// Null when the Quark sent none.
  final String? media;

  /// Link speed, in the Quark's words. Null when the Quark sent none.
  final String? linkSpeed;

  /// The existing speed class (`fast`, `medium`, `slow`, `unknown`).
  /// Null when the Quark sent none.
  final String? speedClass;

  /// [reason] with surrounding space removed. Null when it is blank.
  String? get reasonText {
    final text = reason?.trim();
    if (text == null || text.isEmpty) return null;
    return text;
  }

  /// [smartStatus] as the Quark sent it. Null when it is blank.
  String? get smartStatusText => _present(smartStatus);

  /// Power-on hours as a count of hours. Null when [powerOnHours] is null.
  ///
  /// This is the hour count only. It never names a year.
  String? get powerOnHoursText {
    final hours = powerOnHours;
    if (hours == null) return null;
    return hours == 1 ? '1 hour' : '$hours hours';
  }

  /// Temperature as degrees Celsius. Null when [temperatureCelsius] is null.
  String? get temperatureText {
    final degrees = temperatureCelsius;
    if (degrees == null) return null;
    return '$degrees°C';
  }

  /// Spinning or solid-state, or the media word the Quark sent.
  /// Null when [media] is blank.
  String? get mediaText {
    final raw = _present(media);
    if (raw == null) return null;
    switch (raw) {
      case 'spinning':
        return 'Spinning';
      case 'solid-state':
        return 'Solid-state';
      default:
        return raw;
    }
  }

  /// [linkSpeed] as the Quark sent it. Null when it is blank.
  String? get linkSpeedText => _present(linkSpeed);

  /// [speedClass] in title case for the four classes the Quark sends.
  /// Any other word is shown as sent. Null when it is blank.
  String? get speedClassText {
    final raw = _present(speedClass);
    if (raw == null) return null;
    switch (raw) {
      case 'fast':
        return 'Fast';
      case 'medium':
        return 'Medium';
      case 'slow':
        return 'Slow';
      case 'unknown':
        return 'Unknown';
      default:
        return raw;
    }
  }

  /// Whether the detail panel has a row to reveal.
  bool get hasDetail =>
      smartStatusText != null ||
      reallocatedSectors != null ||
      pendingSectors != null ||
      powerOnHoursText != null ||
      temperatureText != null ||
      mediaText != null ||
      linkSpeedText != null ||
      speedClassText != null;
}

String? _present(String? value) {
  final text = value?.trim();
  if (text == null || text.isEmpty) return null;
  return text;
}
