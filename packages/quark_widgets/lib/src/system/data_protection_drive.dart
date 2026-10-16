/// One backup drive on the protection card: its name, when it was last
/// backed up, and the Quark's reason when that copy does not count.
///
/// The card prints these fields. It does not decide from them whether the
/// files are unprotected, basic, or protected.
class DataProtectionDrive {
  /// Creates a row for the drive identified by [id].
  const DataProtectionDrive({
    required this.id,
    required this.name,
    this.lastBackedUp,
    this.reason,
  });

  /// Stable id for the keys `data_protection_drive_<id>`,
  /// `data_protection_drive_<id>_backed_up`, and
  /// `data_protection_drive_<id>_reason`.
  final String id;

  /// The drive's name, as the Quark reported it.
  final String name;

  /// When the newest completed backup on this drive finished. Null when the
  /// Quark has no completed backup for it.
  final DateTime? lastBackedUp;

  /// Why this drive does not count, in the Quark's words. Null when it counts
  /// or the Quark sent no reason.
  final String? reason;
}
