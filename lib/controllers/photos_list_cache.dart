import 'package:quark/models/file_node.dart';
import 'package:quark/pages/photos_page.dart';
import 'package:quark/utils/listing_snapshot.dart';
import 'package:quark/utils/listing_snapshot_config.dart';
import 'package:quark/utils/listing_snapshot_store.dart';
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
    _persist();
  }

  void setFavoriteKeys(Iterable<String> keys) {
    _favoriteKeys
      ..clear()
      ..addAll(keys);
    _persist();
  }

  void setFavorite(String selectionKey, bool isFavorite) {
    if (isFavorite) {
      _favoriteKeys.add(selectionKey);
    } else {
      _favoriteKeys.remove(selectionKey);
    }
    _persist();
  }

  void clear() {
    _list = null;
    _favoriteKeys.clear();
  }

  /// Fills the list and the favorite keys from the active host's snapshot
  /// where nothing has been fetched yet. A snapshot that cannot be decoded is
  /// discarded.
  Future<void> hydrate() async {
    if (_list != null && _favoriteKeys.isNotEmpty) return;
    final json = await ListingSnapshots.instance.read(
      ListingSnapshotNames.photos,
    );
    if (json == null) return;
    try {
      if (json['version'] != _snapshotVersion) throw const FormatException();
      final photos = (json['photos'] as List)
          .map((e) => e as Map<String, dynamic>)
          .map(
            (e) => PhotoItem.fromFiles(
              FileNode.fromJson(e['file'] as Map<String, dynamic>),
              hasLiveVideo: e['hasLiveVideo'] as bool? ?? false,
            ),
          )
          .toList();
      final keys = (json['favoriteKeys'] as List).cast<String>();
      _list ??= CachedPhotoList(
        photos: List.unmodifiable(photos),
        total: json['total'] as int,
        offset: json['offset'] as int,
      );
      if (_favoriteKeys.isEmpty) _favoriteKeys.addAll(keys);
    } catch (_) {
      await ListingSnapshots.instance.discard(ListingSnapshotNames.photos);
    }
  }

  static const int _snapshotVersion = 1;

  void _persist() {
    ListingSnapshots.instance.schedule(ListingSnapshotNames.photos, _encode);
  }

  /// Only Quark-stored items are written; a device asset is a handle into the
  /// photo library that means nothing after a relaunch.
  Map<String, dynamic> _encode() {
    final list = _list;
    final quark = list?.photos.where((p) => p.isFiles).toList() ?? const [];
    final kept = quark.take(ListingSnapshotConfig.maxPhotos).toList();
    return {
      'version': _snapshotVersion,
      'total': list?.total ?? 0,
      'offset': kept.length < quark.length ? kept.length : list?.offset ?? 0,
      'photos': kept
          .map(
            (p) => {'file': p.quark!.toJson(), 'hasLiveVideo': p.hasLiveVideo},
          )
          .toList(),
      'favoriteKeys': _favoriteKeys
          .take(ListingSnapshotConfig.maxFavoriteKeys)
          .toList(),
    };
  }
}
