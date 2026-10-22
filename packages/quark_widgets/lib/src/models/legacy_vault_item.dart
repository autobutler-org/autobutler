import 'package:flutter/foundation.dart';

/// A household vault from before per-account vaults, as `VaultLegacyNotice`
/// lists it: where the file is and what it holds, never its contents.
@immutable
class LegacyVaultItem {
  /// Creates one found household vault.
  const LegacyVaultItem({
    required this.path,
    this.onDrive = false,
    this.deviceName = '',
    this.readable = true,
    this.entries = 0,
    this.folders = 0,
  });

  /// The file on the Quark, for whoever manages it.
  final String path;

  /// Whether it is a file an older Quark left on an external drive, rather
  /// than the copy kept on the Quark when it was updated.
  final bool onDrive;

  /// The drive's name when [onDrive]. Empty when it is not known.
  final String deviceName;

  /// False for a file that could not be read; [entries] and [folders] are
  /// then unknown.
  final bool readable;

  /// How many saved logins it holds.
  final int entries;

  /// How many folders it holds.
  final int folders;
}
