/// Result returned by [promptForMoveRenamePath] when a destination device
/// picker is shown alongside the path browser.
class MoveRenameResult {
  const MoveRenameResult({required this.targetInput, this.deviceSerial});

  /// The relative or absolute path the user entered/selected.
  final String targetInput;

  /// The serial of the destination device: empty for the internal drive, and
  /// `null` when no device was offered, which keeps the item where it is.
  final String? deviceSerial;
}
