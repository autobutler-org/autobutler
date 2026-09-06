import 'package:quark/models/file_node.dart';
import 'package:quark/utils/file_browser_path_utils.dart';
import 'package:quark/utils/listing_snapshot.dart';
import 'package:quark/utils/listing_snapshot_config.dart';
import 'package:quark/utils/listing_snapshot_store.dart';

/// In-memory cache of file listings keyed by folder path.
///
/// Shared across [FileBrowserPage] instances so that go_router rebuilds (which
/// recreate the widget) can display the previous result immediately while a
/// fresh fetch is in flight.
///
/// Also tracks which file (if any) is currently open in a viewer, so that when
/// a URL update triggers a background go_router rebuild, the rebuilt page
/// doesn't try to open a second viewer on top of the existing one.
class FileBrowserCache {
  FileBrowserCache._();
  static final instance = FileBrowserCache._();

  final Map<String, List<FileNode>> _cache = {};

  /// The key the root folder is cached under: what [normalizePath] returns
  /// for `/`. Only this listing is written to disk (#1781).
  static const String rootKey = '';

  List<FileNode>? get(String path) => _cache[path];

  void put(String path, List<FileNode> files) {
    _cache[path] = List.unmodifiable(files);
    if (path == rootKey) {
      ListingSnapshots.instance.schedule(
        ListingSnapshotNames.rootFiles,
        () => _encodeRoot(files),
      );
    }
  }

  /// Fills the root listing from the active host's snapshot when nothing has
  /// been fetched yet. A snapshot that cannot be decoded is discarded.
  Future<void> hydrate() async {
    if (_cache.containsKey(rootKey)) return;
    final json = await ListingSnapshots.instance.read(
      ListingSnapshotNames.rootFiles,
    );
    if (json == null || _cache.containsKey(rootKey)) return;
    try {
      if (json['version'] != _snapshotVersion) throw const FormatException();
      final files = (json['files'] as List)
          .map((e) => FileNode.fromJson(e as Map<String, dynamic>))
          .toList();
      _cache[rootKey] = List.unmodifiable(files);
    } catch (_) {
      await ListingSnapshots.instance.discard(ListingSnapshotNames.rootFiles);
    }
  }

  static const int _snapshotVersion = 1;

  static Map<String, dynamic> _encodeRoot(List<FileNode> files) => {
    'version': _snapshotVersion,
    'files': files
        .take(ListingSnapshotConfig.maxRootFiles)
        .map((f) => f.toJson())
        .toList(),
  };

  void evict(String path) => _cache.remove(path);

  void clear() => _cache.clear();

  // ── Open-file tracking ────────────────────────────────────────────────────

  String? _openFilePath;

  /// Keys are normalized on the way in and out so callers cannot disagree about
  /// the format. `markFileOpen` used to store the raw path while the
  /// `didUpdateWidget` guard looked it up normalized, which left that guard
  /// dead for every path missing a leading slash (#1604).
  static String _key(String path) => normalizePath(path);

  /// Mark [path] as currently open in a viewer/editor overlay.
  void markFileOpen(String path) => _openFilePath = _key(path);

  /// Clear the open-file marker once the viewer/editor for [path] is dismissed.
  ///
  /// Scoped to [path] so a viewer that closes late cannot clear a marker set by
  /// whatever opened after it.
  void markFileClosed(String path) {
    if (_openFilePath == _key(path)) {
      _openFilePath = null;
    }
  }

  /// Clear the open-file marker regardless of which path set it.
  void clearOpenFile() => _openFilePath = null;

  /// Returns true if [path] is already being shown — prevents a second viewer
  /// from being pushed by a background [FileBrowserPage] rebuild.
  bool isFileOpen(String path) =>
      _openFilePath != null && _openFilePath == _key(path);

  /// The path currently marked open, normalized. Null when nothing is open.
  String? get openFilePath => _openFilePath;
}
