// cspell:words reallocated celsius replacesoon
import 'package:quark/services/storage_service.dart';
import 'package:quark_widgets/quark_widgets.dart';

/// What the Storage tab asks before a drive is committed to.
///
/// [sendAcknowledgment] is true only for Replace soon: the request then
/// carries `acknowledgeHealth`. Watch and Unknown never set it. The Quark
/// decides [DriveHealthPrompt.strongerWarning] with `worseThanDefault`; this
/// type does not rank verdicts.
class DriveHealthPrompt {
  /// Creates a prompt.
  const DriveHealthPrompt({
    required this.sendAcknowledgment,
    required this.strongerWarning,
    required this.keepsExistingFiles,
    required this.addsBackup,
  });

  /// Whether the request must send `acknowledgeHealth`.
  final bool sendAcknowledgment;

  /// Whether default storage would move to a drive the Quark marked as worse.
  final bool strongerWarning;

  /// Whether the dialog says files already on the drive stay.
  final bool keepsExistingFiles;

  /// Whether the dialog says the backup is added beside those files.
  final bool addsBackup;
}

/// A role change or a backup start, the moments that commit to a drive.
enum DriveHealthAction {
  /// Assign `default-storage`.
  setDefaultStorage,

  /// Assign `snapshot-backup`.
  setBackupRole,

  /// Start a snapshot backup onto the drive.
  startBackup,
}

/// Maps the Quark's health payload into the widgets, and decides when a
/// commit needs a dialog.
///
/// The verdict, the sentence, and `worseThanDefault` are used as sent. Nothing
/// here reads SMART attributes or scores one drive against another.
class DriveHealthController {
  const DriveHealthController._();

  /// The widget model for [report]. A missing report is Unknown, with no sentence.
  static DriveHealth view(StorageDriveHealth? report) {
    if (report == null) {
      return const DriveHealth(verdict: DriveHealthVerdict.unknown);
    }
    return DriveHealth(
      verdict: _verdict(report.verdict),
      reason: report.reason,
      smartStatus: report.smartStatus,
      reallocatedSectors: report.reallocatedSectors,
      pendingSectors: report.pendingSectors,
      powerOnHours: report.powerOnHours,
      temperatureCelsius: report.temperatureCelsius,
      media: report.media,
      linkSpeed: report.linkSpeed,
      speedClass: report.speedClass,
    );
  }

  /// A stable id for the health keys on [device].
  static String idFor(StorageDevice device) {
    if (device.serial.isNotEmpty) return device.serial;
    if (device.devicePath.isNotEmpty) return device.devicePath;
    if (device.name.isNotEmpty) return device.name;
    return 'drive';
  }

  /// The dialog to show before [action], or null when the action proceeds.
  ///
  /// Replace soon always asks, and confirming sends the acknowledgment.
  /// [worseThanDefault] adds the stronger warning only when [action] is
  /// [DriveHealthAction.setDefaultStorage]. Watch does not ask unless that
  /// warning applies, and it never sends the acknowledgment.
  static DriveHealthPrompt? prompt({
    required DriveHealthVerdict verdict,
    required DriveHealthAction action,
    required bool worseThanDefault,
  }) {
    final replace = verdict == DriveHealthVerdict.replaceSoon;
    final stronger =
        action == DriveHealthAction.setDefaultStorage && worseThanDefault;
    if (!replace && !stronger) return null;
    final backup = action != DriveHealthAction.setDefaultStorage;
    return DriveHealthPrompt(
      sendAcknowledgment: replace,
      strongerWarning: stronger,
      keepsExistingFiles: true,
      addsBackup: backup,
    );
  }

  static DriveHealthVerdict _verdict(String raw) {
    final compact = raw.toLowerCase().replaceAll(RegExp(r'[^a-z]'), '');
    switch (compact) {
      case 'healthy':
        return DriveHealthVerdict.healthy;
      case 'watch':
        return DriveHealthVerdict.watch;
      case 'replacesoon':
        return DriveHealthVerdict.replaceSoon;
      default:
        return DriveHealthVerdict.unknown;
    }
  }
}
