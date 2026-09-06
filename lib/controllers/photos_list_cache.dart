import 'package:quark/pages/photos_page.dart';
import 'package:quark/utils/photo_grid_config.dart';

/// The Quark-stored photo pages a [PhotosPage] last loaded successfully.
class CachedPhotoList {
  const CachedPhotoList({
    required this.photos,
    required this.total,
    required this.offset,
  });

  final List<PhotoItem> photos;

  /// How many photos the Quark reported in all.
  final int total;

  /// Where the next page starts.
  final int offset;
}

/// In-memory cache of the Quark-stored photo list and the favorite keys.
///
/// Sibling to `FileBrowserCache`, which does the same job for folder
/// listings: a process-wide singleton, so a [PhotosPage] recreated by a
/// go_router rebuild can show the previous result immediately while a fresh
/// fetch is in flight. Device (photo_manager) assets are never held here.
///
/// Bounded by [PhotoGridConfig.maxCachedPhotos] so a reader who scrolled
/// through thousands of photos does not pin them all for the session.
class PhotosListCache {
  PhotosListCache._();
  static final instance = PhotosListCache._();

  CachedPhotoList? _list;
  final Set<String> _favoriteKeys = {};

  CachedPhotoList? get list => _list;

  /// A read-only view of the favorite keys held.
  Set<String> get favoriteKeys => Set.unmodifiable(_favoriteKeys);

  void put(List<PhotoItem> photos, {required int total, required int offset}) {
    final cap = PhotoGridConfig.maxCachedPhotos;
    final kept = photos.length > cap ? photos.sublist(0, cap) : photos;
    _list = CachedPhotoList(
      photos: List.unmodifiable(kept),
      total: total,
      offset: kept.length < photos.length ? kept.length : offset,
    );
  }

  void setFavoriteKeys(Iterable<String> keys) {
    _favoriteKeys
      ..clear()
      ..addAll(keys);
  }

  void setFavorite(String selectionKey, bool isFavorite) {
    if (isFavorite) {
      _favoriteKeys.add(selectionKey);
    } else {
      _favoriteKeys.remove(selectionKey);
    }
  }

  void clear() {
    _list = null;
    _favoriteKeys.clear();
  }
}
