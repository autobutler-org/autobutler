import 'dart:convert';

import 'package:quark/services/app_settings.dart';
import 'package:quark/services/authenticated_service.dart';
import 'package:quark/utils/error_text.dart';

/// What to do next, as `GET /api/v0/storage/protection` names it.
enum DataProtectionAction {
  /// Give the snapshot-backup role to a drive that does not have it.
  assignBackupRole,

  /// Run a backup on a snapshot-backup drive that does not yet count.
  startBackup,

  /// Two backup copies exist. Keep one of those drives somewhere else.
  keepElsewhere,
}

/// One snapshot-backup drive in the protection response.
class DataProtectionDriveInfo {
  /// Creates a drive row from the response fields.
  const DataProtectionDriveInfo({
    required this.serial,
    required this.name,
    required this.counts,
    this.reason,
    this.lastBackedUp,
  });

  /// The drive's serial. Empty for a drive the Quark could not name.
  final String serial;

  /// The drive's display name. Empty when the response omitted it.
  final String name;

  /// Whether this drive adds a backup copy. The level is separate: the client
  /// does not add these up.
  final bool counts;

  /// Why [counts] is false, in the Quark's words. Null when it counts.
  final String? reason;

  /// When this drive's newest completed backup finished. Null when it has none.
  final DateTime? lastBackedUp;
}

/// The protection status from `GET /api/v0/storage/protection`.
///
/// [level] is `unprotected`, `basic`, or `protected`. [sentence] is the one
/// sentence the Quark wrote. [action] is `assign-backup-role`, `start-backup`,
/// or `keep-one-elsewhere`. [drives] lists each snapshot-backup drive.
class DataProtectionStatus {
  /// Creates a status the Quark has already decided.
  const DataProtectionStatus({
    required this.level,
    required this.sentence,
    required this.action,
    required this.drives,
  });

  /// Parses the JSON body of `GET /api/v0/storage/protection`.
  factory DataProtectionStatus.fromJson(Map<String, dynamic> json) {
    final level = json['level'];
    final sentence = json['sentence'];
    final nextStep = json['nextStep'];
    if (level is! String || sentence is! String || nextStep is! String) {
      throw const FormatException('Invalid protection status');
    }
    final drives = json['drives'];
    return DataProtectionStatus(
      level: _level(level),
      sentence: sentence,
      action: _action(nextStep),
      drives: [
        if (drives is List)
          for (final item in drives)
            if (item is Map<String, dynamic>)
              DataProtectionDriveInfo(
                serial: item['serial'] as String? ?? '',
                name: item['name'] as String? ?? '',
                counts: item['counts'] as bool? ?? false,
                reason: _reason(item['reason']),
                lastBackedUp: _time(item['lastBackedUp']),
              ),
      ],
    );
  }

  /// The internal drive does not count as a backup copy.
  static const reasonInternal = 'The internal drive is not a backup copy.';

  /// No completed snapshot backup yet. A failed or unfinished job leaves this.
  static const reasonNoCompleted = 'No completed backup yet.';

  /// The newest completion is outside the 30-day window.
  static const reasonStale = 'The last backup is more than 30 days old.';

  /// A second partition of a disk that already counts.
  static const reasonSameDisk =
      'This partition is on the same disk as another backup.';

  /// `unprotected`, `basic`, or `protected`.
  final String level;

  /// The Quark's one sentence.
  final String sentence;

  /// The one recommended action.
  final DataProtectionAction action;

  /// Each snapshot-backup drive.
  final List<DataProtectionDriveInfo> drives;

  /// The serial a start-backup step should run on: the first drive that does
  /// not count and that a completed backup could still make count.
  String? get backupTargetSerial {
    String? fallback;
    for (final drive in drives) {
      if (drive.counts || drive.serial.isEmpty) continue;
      final reason = drive.reason;
      if (reason == reasonInternal || reason == reasonSameDisk) continue;
      if (reason == reasonNoCompleted || reason == reasonStale) {
        return drive.serial;
      }
      fallback ??= drive.serial;
    }
    return fallback;
  }
}

String _level(String raw) {
  switch (raw) {
    case 'unprotected':
    case 'basic':
    case 'protected':
      return raw;
    default:
      throw const FormatException('Invalid protection status');
  }
}

DataProtectionAction _action(String raw) {
  switch (raw) {
    case 'assign-backup-role':
      return DataProtectionAction.assignBackupRole;
    case 'start-backup':
      return DataProtectionAction.startBackup;
    case 'keep-one-elsewhere':
      return DataProtectionAction.keepElsewhere;
    default:
      throw const FormatException('Invalid protection status');
  }
}

String? _reason(Object? raw) {
  if (raw is! String || raw.isEmpty) return null;
  return raw;
}

DateTime? _time(Object? raw) {
  if (raw == null) return null;
  if (raw is! String || raw.isEmpty) {
    throw const FormatException('Invalid protection status');
  }
  return DateTime.parse(raw);
}

/// Reads `GET /api/v0/storage/protection`.
///
/// The body is the level, one sentence, the next step, and each backup
/// drive. This client shows that level. It does not count drives again.
class DataProtectionService with AuthenticatedService {
  DataProtectionService._();

  /// The shared service.
  static final DataProtectionService instance = DataProtectionService._();

  /// Fetches the current protection status.
  static Future<DataProtectionStatus> fetch() async {
    final response = await instance.authenticatedGet(
      apiBaseUri.resolve('/api/v0/storage/protection'),
    );
    final status = response.statusCode;
    if (status >= 200 && status < 300) {
      final body = jsonDecode(response.body);
      if (body is! Map<String, dynamic>) {
        throw const FormatException('Invalid protection status');
      }
      return DataProtectionStatus.fromJson(body);
    }
    throw ApiException(status, 'check the backup');
  }
}
