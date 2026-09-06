import 'package:quark/models/photo_album.dart';

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
  }

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
