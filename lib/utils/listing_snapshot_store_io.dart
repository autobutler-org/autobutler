import 'dart:io';

import 'package:path_provider/path_provider.dart';
import 'package:quark/utils/listing_snapshot.dart';
import 'package:quark/utils/listing_snapshot_config.dart';

/// Snapshots as JSON files under
/// `<base>/<host directory>/<snapshot name>.json`.
///
/// [baseDirectory] defaults to `listing_cache` in the application support
/// directory; tests hand in a temporary directory instead so no platform
/// channel is needed. A write goes to a sibling temporary file first and is
/// renamed into place, so a process killed mid-write leaves the previous
/// snapshot intact rather than half of the new one.
class FileListingSnapshotStore implements ListingSnapshotStore {
  FileListingSnapshotStore({Future<Directory> Function()? baseDirectory})
    : _baseDirectory = baseDirectory ?? _applicationSupportBase;

  final Future<Directory> Function() _baseDirectory;
  Directory? _base;

  static Future<Directory> _applicationSupportBase() async {
    final support = await getApplicationSupportDirectory();
    return Directory('${support.path}/${ListingSnapshotConfig.directoryName}');
  }

  Future<Directory> _root() async => _base ??= await _baseDirectory();

  Future<File> _file(String hostKey, String name) async {
    final root = await _root();
    return File('${root.path}/$hostKey/$name.json');
  }

  @override
  Future<String?> read(String hostKey, String name) async {
    final file = await _file(hostKey, name);
    if (!await file.exists()) return null;
    return file.readAsString();
  }

  @override
  Future<void> write(String hostKey, String name, String contents) async {
    final file = await _file(hostKey, name);
    await file.parent.create(recursive: true);
    final staging = File('${file.path}.tmp');
    await staging.writeAsString(contents, flush: true);
    await staging.rename(file.path);
  }

  @override
  Future<void> remove(String hostKey, String name) async {
    final file = await _file(hostKey, name);
    if (await file.exists()) await file.delete();
  }

  @override
  Future<void> removeHost(String hostKey) async {
    final root = await _root();
    final directory = Directory('${root.path}/$hostKey');
    if (await directory.exists()) await directory.delete(recursive: true);
  }
}

final ListingSnapshotStore listingSnapshotStorePlatform =
    FileListingSnapshotStore();
