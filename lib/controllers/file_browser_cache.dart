import 'package:quark/models/cirrus_file_node.dart';

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

  final Map<String, List<CirrusFileNode>> _cache = {};

  List<CirrusFileNode>? get(String path) => _cache[path];

  void put(String path, List<CirrusFileNode> files) {
    _cache[path] = List.unmodifiable(files);
  }

  void evict(String path) => _cache.remove(path);

  void clear() => _cache.clear();

  // ── Open-file tracking ────────────────────────────────────────────────────

  String? _openFilePath;

  /// Mark [path] as currently open in a viewer/editor overlay.
  void markFileOpen(String path) => _openFilePath = path;

  /// Clear the open-file marker once the viewer/editor is dismissed.
  void markFileClosed() => _openFilePath = null;

  /// Returns true if [path] is already being shown — prevents a second viewer
  /// from being pushed by a background [FileBrowserPage] rebuild.
  bool isFileOpen(String path) => _openFilePath == path;
}
