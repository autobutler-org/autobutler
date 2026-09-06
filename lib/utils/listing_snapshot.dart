/// Names of the snapshots kept per host, each its own file.
abstract final class ListingSnapshotNames {
  static const String rootFiles = 'root_files';
  static const String photos = 'photos';
  static const String albums = 'albums';
}

/// Where the encoded snapshots for a host live on this platform.
///
/// [hostKey] is already a filesystem-safe directory name, see
/// [listingSnapshotDirectoryName]. Failures are thrown, not swallowed: the
/// caller decides what a missing or unwritable snapshot means.
abstract class ListingSnapshotStore {
  Future<String?> read(String hostKey, String name);
  Future<void> write(String hostKey, String name, String contents);
  Future<void> remove(String hostKey, String name);
  Future<void> removeHost(String hostKey);
}

/// A store that keeps nothing, for platforms with no usable filesystem.
class NoopListingSnapshotStore implements ListingSnapshotStore {
  const NoopListingSnapshotStore();

  @override
  Future<String?> read(String hostKey, String name) async => null;

  @override
  Future<void> write(String hostKey, String name, String contents) async {}

  @override
  Future<void> remove(String hostKey, String name) async {}

  @override
  Future<void> removeHost(String hostKey) async {}
}

const _safeCodeUnits =
    'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz'
    '0123456789._-';

/// The directory name for a host key, as `AppSettings` keys session tokens.
///
/// Percent-encodes every byte outside `[A-Za-z0-9._-]`, so `https://a:1`
/// and `https://a_1` land in different directories and no key can name a
/// parent directory.
String listingSnapshotDirectoryName(String hostKey) {
  final buffer = StringBuffer();
  for (final unit in hostKey.codeUnits) {
    if (unit < 128 && _safeCodeUnits.contains(String.fromCharCode(unit))) {
      buffer.writeCharCode(unit);
    } else {
      buffer.write('%${unit.toRadixString(16).padLeft(4, '0').toUpperCase()}');
    }
  }
  final name = buffer.toString();
  return name.isEmpty || name.startsWith('.') ? '_$name' : name;
}
