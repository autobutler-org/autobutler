/// A household vault from before per-account vaults, as `GET
/// /api/v0/vault/legacy` lists it (#3616): the copy an update kept on the
/// Quark, or a `vault.db` an older Quark left on a drive.
///
/// Nothing carries it into an account's vault, because only the old shared
/// master password opens it; the vault page says it was kept and where.
class LegacyVault {
  /// Builds one found household vault.
  const LegacyVault({
    required this.kind,
    required this.path,
    required this.readable,
    required this.entries,
    required this.folders,
    this.deviceName = '',
  });

  /// `kept`, the copy made on update, or `drive`, a file on a drive.
  final String kind;

  /// The file on the Quark.
  final String path;

  /// False for a file the Quark could not count.
  final bool readable;

  /// How many saved logins and folders it holds.
  final int entries;
  final int folders;

  /// The drive's name, for a `drive` file.
  final String deviceName;

  /// Whether the file is on an external drive.
  bool get onDrive => kind == 'drive';

  /// Reads one element of the response's `vaults`.
  factory LegacyVault.fromJson(Map<String, dynamic> json) => LegacyVault(
    kind: json['kind'] as String? ?? 'kept',
    path: json['path'] as String? ?? '',
    readable: json['readable'] as bool? ?? false,
    entries: (json['entries'] as num?)?.toInt() ?? 0,
    folders: (json['folders'] as num?)?.toInt() ?? 0,
    deviceName: json['deviceName'] as String? ?? '',
  );
}
