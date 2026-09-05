import 'package:quark/models/file_node.dart';
import 'package:quark/services/content_search_service.dart';

/// In-memory cache of by-type file listings, keyed by file type.
///
/// Sibling to `FileBrowserCache`, which does the same job for folder listings:
/// a process-wide singleton, so the Docs and Sheets pages can show the last
/// listing on their first frame and refresh behind it instead of scanning the
/// whole tree behind a spinner on every visit (#1780).
class FileTypeListingCache {
  FileTypeListingCache._();
  static final instance = FileTypeListingCache._();

  final Map<String, List<FileNode>> _listings = {};

  List<FileNode>? get(String fileType) => _listings[fileType];

  void put(String fileType, List<FileNode> files) {
    _listings[fileType] = List.unmodifiable(files);
  }

  void evict(String fileType) => _listings.remove(fileType);

  /// Drops every listing, and with it the content-search memo: a file that
  /// was deleted, moved or renamed can turn up in either.
  void clear() {
    _listings.clear();
    ContentSearchService.clearRecent();
  }
}
