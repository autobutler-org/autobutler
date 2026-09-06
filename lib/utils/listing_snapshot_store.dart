import 'dart:async';
import 'dart:convert';

import 'package:flutter/foundation.dart';
import 'package:quark/utils/listing_snapshot.dart';
import 'package:quark/utils/listing_snapshot_config.dart';
import 'package:quark/utils/listing_snapshot_store_io.dart'
    if (dart.library.js_interop) 'package:quark/utils/listing_snapshot_store_web.dart'
    as platform;

/// The store this platform keeps listing snapshots in.
ListingSnapshotStore get listingSnapshotStore =>
    platform.listingSnapshotStorePlatform;

/// Writes the in-memory listing caches to disk and reads them back, scoped to
/// the active host (#1781).
///
/// Writes are debounced per snapshot and skipped when the encoded content
/// matches what was last written or read, so the refresh loop putting the
/// same listing every few seconds never touches the disk. Switching hosts
/// drops whatever was still pending: a write scheduled for one Quark must
/// never land in another's directory. Every failure is logged and swallowed;
/// a snapshot is a convenience, and the next fetch corrects it anyway.
class ListingSnapshots {
  ListingSnapshots._();
  static final instance = ListingSnapshots._();

  /// Replaceable so tests can point at a temporary directory.
  ListingSnapshotStore store = listingSnapshotStore;

  String? _directory;
  final Map<String, Timer> _pending = {};
  final Map<String, Map<String, dynamic> Function()> _encoders = {};
  final Map<String, String> _written = {};
  final Set<Future<void>> _inFlight = {};

  /// The directory snapshots are read from and written to, or null when no
  /// host is active.
  String? get directory => _directory;

  /// Points the store at [hostKey]'s directory, or at nothing when null.
  void setHost(String? hostKey) {
    final directory = hostKey == null
        ? null
        : listingSnapshotDirectoryName(hostKey);
    if (directory == _directory) return;
    _cancelPending();
    _written.clear();
    _directory = directory;
  }

  /// Queues a write of [name] with whatever [encode] returns once the debounce
  /// settles. A later call for the same [name] replaces an earlier one.
  void schedule(String name, Map<String, dynamic> Function() encode) {
    if (_directory == null) return;
    _pending[name]?.cancel();
    _encoders[name] = encode;
    _pending[name] = Timer(
      ListingSnapshotConfig.writeDebounce,
      () => _flushOne(name),
    );
  }

  /// Writes every pending snapshot now and waits for the writes to finish.
  Future<void> flush() async {
    for (final name in _pending.keys.toList()) {
      _flushOne(name);
    }
    await Future.wait(_inFlight.toList());
  }

  /// The decoded snapshot [name] for the active host, or null when there is
  /// none. A file that cannot be read or is not a JSON object is deleted.
  Future<Map<String, dynamic>?> read(String name) async {
    final directory = _directory;
    if (directory == null) return null;
    try {
      final contents = await store.read(directory, name);
      if (contents == null || _directory != directory) return null;
      final decoded = jsonDecode(contents);
      if (decoded is Map<String, dynamic>) {
        _written[name] = contents;
        return decoded;
      }
    } catch (e) {
      debugPrint('[listing_snapshot_store.dart] Cannot read $name: $e');
    }
    await _remove(directory, name);
    return null;
  }

  /// Deletes the snapshot [name] for the active host.
  Future<void> discard(String name) async {
    final directory = _directory;
    if (directory == null) return;
    _pending.remove(name)?.cancel();
    _encoders.remove(name);
    _written.remove(name);
    await _remove(directory, name);
  }

  /// Deletes every snapshot kept for [hostKey].
  Future<void> removeHost(String hostKey) async {
    final directory = listingSnapshotDirectoryName(hostKey);
    if (directory == _directory) {
      _cancelPending();
      _written.clear();
    }
    await Future.wait(_inFlight.toList());
    try {
      await store.removeHost(directory);
    } catch (e) {
      debugPrint('[listing_snapshot_store.dart] Cannot remove $directory: $e');
    }
  }

  void _cancelPending() {
    for (final timer in _pending.values) {
      timer.cancel();
    }
    _pending.clear();
    _encoders.clear();
  }

  void _flushOne(String name) {
    _pending.remove(name)?.cancel();
    final encode = _encoders.remove(name);
    final directory = _directory;
    if (encode == null || directory == null) return;
    final write = _write(directory, name, encode);
    _inFlight.add(write);
    write.whenComplete(() => _inFlight.remove(write));
  }

  Future<void> _write(
    String directory,
    String name,
    Map<String, dynamic> Function() encode,
  ) async {
    try {
      final contents = jsonEncode(encode());
      if (_written[name] == contents) return;
      await store.write(directory, name, contents);
      if (_directory == directory) _written[name] = contents;
    } catch (e) {
      debugPrint('[listing_snapshot_store.dart] Cannot write $name: $e');
    }
  }

  Future<void> _remove(String directory, String name) async {
    try {
      await store.remove(directory, name);
    } catch (e) {
      debugPrint('[listing_snapshot_store.dart] Cannot delete $name: $e');
    }
  }
}
