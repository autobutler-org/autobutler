import 'package:quark/models/photo_album.dart';
import 'package:quark/utils/listing_snapshot.dart';
import 'package:quark/utils/listing_snapshot_config.dart';
import 'package:quark/utils/listing_snapshot_store.dart';

/// In-memory cache of the album tree and each album's items.
///
/// Sibling to `FileBrowserCache`: a process-wide singleton, so the album
/// sidebar and an album page reopened a moment later show what the last one
/// fetched instead of a spinner. Album membership only changes through
/// `AlbumService`, which drops the affected entries on every mutation.
class AlbumsCache {
  AlbumsCache._();
  static final instance = AlbumsCache._();

  List<PhotoAlbum>? _albums;
  final Map<int, List<PhotoAlbumItem>> _items = {};

  List<PhotoAlbum>? get albums => _albums;

  void putAlbums(List<PhotoAlbum> albums) {
    _albums = List.unmodifiable(albums);
    ListingSnapshots.instance.schedule(
      ListingSnapshotNames.albums,
      () => _encode(albums),
    );
  }

  /// Fills the album list from the active host's snapshot when nothing has
  /// been fetched yet. A snapshot that cannot be decoded is discarded.
  Future<void> hydrate() async {
    if (_albums != null) return;
    final json = await ListingSnapshots.instance.read(
      ListingSnapshotNames.albums,
    );
    if (json == null || _albums != null) return;
    try {
      if (json['version'] != _snapshotVersion) throw const FormatException();
      final albums = (json['albums'] as List)
          .map((e) => PhotoAlbum.fromJson(e as Map<String, dynamic>))
          .toList();
      _albums = List.unmodifiable(albums);
    } catch (_) {
      await ListingSnapshots.instance.discard(ListingSnapshotNames.albums);
    }
  }

  static const int _snapshotVersion = 1;

  static Map<String, dynamic> _encode(List<PhotoAlbum> albums) => {
    'version': _snapshotVersion,
    'albums': albums
        .take(ListingSnapshotConfig.maxAlbums)
        .map((a) => a.toJson())
        .toList(),
  };

  void evictAlbums() => _albums = null;

  List<PhotoAlbumItem>? items(int albumId) => _items[albumId];

  void putItems(int albumId, List<PhotoAlbumItem> items) {
    _items[albumId] = List.unmodifiable(items);
  }

  void evictItems(int albumId) => _items.remove(albumId);

  void clearItems() => _items.clear();

  void clear() {
    _albums = null;
    _items.clear();
  }
}
