import 'package:flutter_test/flutter_test.dart';
import 'package:quark/controllers/albums_cache.dart';
import 'package:quark/controllers/app_caches.dart';
import 'package:quark/controllers/file_browser_cache.dart';
import 'package:quark/controllers/file_type_listing_cache.dart';
import 'package:quark/controllers/photos_list_cache.dart';
import 'package:quark/models/file_node.dart';
import 'package:quark/models/photo_album.dart';
import 'package:quark/pages/photos_page.dart';

PhotoAlbum _album(int id) => PhotoAlbum(
  id: id,
  name: 'Album $id',
  createdAt: DateTime(2026),
  updatedAt: DateTime(2026),
  itemCount: 0,
);

PhotoItem _photo() => PhotoItem.fromFiles(_node('p.jpg'));

FileNode _node(String name) => FileNode(
  name: name,
  size: 1,
  isDir: false,
  deviceName: '',
  devicePath: '',
  deviceSerial: '',
  dirPath: '/',
);

void main() {
  setUp(AppCaches.clearAll);
  tearDown(AppCaches.clearAll);

  // Every cache AppCaches knows about is filled here, so a cache added to the
  // app but not to clearAll fails this test rather than leaking one user's
  // listings to the next (see AppCaches' doc comment).
  void fillEveryCache() {
    FileBrowserCache.instance.put('/', [_node('a.txt')]);
    AlbumsCache.instance.putAlbums([_album(1)]);
    AlbumsCache.instance.putItems(1, const []);
    PhotosListCache.instance.put([_photo()], total: 1, offset: 1);
    PhotosListCache.instance.setFavoriteKeys({'k'});
    FileTypeListingCache.instance.put('doc', [_node('a.qdoc')]);
  }

  void expectEveryCacheEmpty() {
    expect(FileBrowserCache.instance.get('/'), isNull);
    expect(AlbumsCache.instance.albums, isNull);
    expect(AlbumsCache.instance.items(1), isNull);
    expect(PhotosListCache.instance.list, isNull);
    expect(PhotosListCache.instance.favoriteKeys, isEmpty);
    expect(FileTypeListingCache.instance.get('doc'), isNull);
  }

  test('clearAll empties every cache', () {
    fillEveryCache();
    AppCaches.clearAll();
    expectEveryCacheEmpty();
  });

  test('endSession clears every cache too', () async {
    fillEveryCache();
    await AppCaches.endSession(null);
    expectEveryCacheEmpty();
  });

  test('clearAll is safe to call when nothing is cached', () {
    AppCaches.clearAll();
    expect(AppCaches.clearAll, returnsNormally);
    expectEveryCacheEmpty();
  });
}
