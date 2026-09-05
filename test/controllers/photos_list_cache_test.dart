import 'package:flutter_test/flutter_test.dart';
import 'package:quark/controllers/photos_list_cache.dart';
import 'package:quark/models/file_node.dart';
import 'package:quark/pages/photos_page.dart';
import 'package:quark/utils/photo_grid_config.dart';

/// #1778: every visit to Photos recreated the page state and showed a
/// spinner until the first page and the favorites both round-tripped. This
/// is the cache that lets a returning reader see the last grid on frame one.
void main() {
  final cache = PhotosListCache.instance;

  PhotoItem photo(int i) => PhotoItem.fromFiles(
    FileNode(
      name: 'p$i.jpg',
      size: 1,
      isDir: false,
      deviceName: '',
      devicePath: '',
      deviceSerial: 'abc',
      dirPath: '/photos/p$i.jpg',
    ),
  );

  List<PhotoItem> photos(int count) =>
      List.generate(count, photo, growable: false);

  setUp(cache.clear);
  tearDown(cache.clear);

  group('list', () {
    test('holds nothing until a page is put', () {
      expect(cache.list, isNull);
    });

    test('a put page comes back with its total and offset', () {
      final page = photos(50);
      cache.put(page, total: 120, offset: 50);

      final cached = cache.list!;
      expect(cached.photos, page);
      expect(cached.total, 120);
      expect(cached.offset, 50);
    });

    test('a later put replaces the earlier one', () {
      cache.put(photos(50), total: 120, offset: 50);
      cache.put(photos(100), total: 120, offset: 100);

      expect(cache.list!.photos.length, 100);
      expect(cache.list!.offset, 100);
    });

    test('the cached list cannot be mutated by a holder', () {
      cache.put(photos(3), total: 3, offset: 3);

      expect(() => cache.list!.photos.add(photo(9)), throwsUnsupportedError);
    });

    test('a list over the cap is cut to the cap and the offset with it', () {
      final cap = PhotoGridConfig.maxCachedPhotos;
      final page = photos(cap + 250);
      cache.put(page, total: cap + 1000, offset: cap + 250);

      final cached = cache.list!;
      expect(cached.photos.length, cap);
      expect(cached.photos.first.selectionKey, page.first.selectionKey);
      expect(cached.offset, cap);
      expect(cached.total, cap + 1000);
    });

    test('a list at the cap is kept whole', () {
      final cap = PhotoGridConfig.maxCachedPhotos;
      cache.put(photos(cap), total: cap, offset: cap);

      expect(cache.list!.photos.length, cap);
      expect(cache.list!.offset, cap);
    });
  });

  group('favorites', () {
    test('holds no favorites until told', () {
      expect(cache.favoriteKeys, isEmpty);
    });

    test('setFavoriteKeys replaces the whole set', () {
      cache.setFavoriteKeys(['a', 'b']);
      cache.setFavoriteKeys(['c']);

      expect(cache.favoriteKeys, {'c'});
    });

    test('setFavorite adds and removes one key in place', () {
      cache.setFavoriteKeys(['a', 'b']);

      cache.setFavorite('c', true);
      expect(cache.favoriteKeys, {'a', 'b', 'c'});

      cache.setFavorite('a', false);
      expect(cache.favoriteKeys, {'b', 'c'});
    });

    test('toggling a favorite leaves the list alone', () {
      cache.put(photos(3), total: 3, offset: 3);
      cache.setFavorite('abc:/photos/p1.jpg', true);

      expect(cache.list!.photos.length, 3);
    });

    test('the exposed set cannot be mutated by a holder', () {
      cache.setFavoriteKeys(['a']);

      expect(() => cache.favoriteKeys.add('b'), throwsUnsupportedError);
      expect(cache.favoriteKeys, {'a'});
    });
  });

  group('clear', () {
    test('drops the list and the favorites together', () {
      cache.put(photos(3), total: 3, offset: 3);
      cache.setFavoriteKeys(['a']);

      cache.clear();

      expect(cache.list, isNull);
      expect(cache.favoriteKeys, isEmpty);
    });
  });
}
