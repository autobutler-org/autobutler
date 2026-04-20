import 'package:autobutler/models/cirrus_file_node.dart';

/// In-memory cache of file listings keyed by folder path.
///
/// Shared across [FileBrowserPage] instances so that go_router rebuilds (which
/// recreate the widget) can display the previous result immediately while a
/// fresh fetch is in flight.
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
}
