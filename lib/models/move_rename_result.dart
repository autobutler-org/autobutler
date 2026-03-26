/// Result returned by [promptForMoveRenamePath] when a destination device
/// picker is shown alongside the path browser.
class MoveRenameResult {
  const MoveRenameResult({required this.targetInput, this.deviceSerial});

  /// The relative or absolute path the user entered/selected.
  final String targetInput;

  /// The serial of the destination device, or `null` to keep same device.
  final String? deviceSerial;
}
