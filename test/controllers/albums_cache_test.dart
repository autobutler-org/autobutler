import 'package:flutter_test/flutter_test.dart';
import 'package:quark/controllers/albums_cache.dart';
import 'package:quark/models/photo_album.dart';

PhotoAlbum _album(int id) => PhotoAlbum(
  id: id,
  name: 'Album $id',
  createdAt: DateTime(2026),
  updatedAt: DateTime(2026),
  itemCount: 0,
);

PhotoAlbumItem _item(int id, int albumId) => PhotoAlbumItem(
  id: id,
  albumId: albumId,
  deviceSerial: '',
  relPath: 'photos/$id.jpg',
  addedAt: DateTime(2026),
);

void main() {
  final cache = AlbumsCache.instance;

  setUp(cache.clear);
  tearDown(cache.clear);

  group('album list', () {
    test('is missing by default', () {
      expect(cache.albums, isNull);
    });

    test('stores and returns the last list', () {
      cache.putAlbums([_album(1), _album(2)]);
      expect(cache.albums?.map((a) => a.id), [1, 2]);
    });

    test('a later put replaces the earlier one', () {
      cache.putAlbums([_album(1)]);
      cache.putAlbums([_album(2)]);
      expect(cache.albums?.map((a) => a.id), [2]);
    });

    test('the stored list is unmodifiable', () {
      cache.putAlbums([_album(1)]);
      expect(() => cache.albums!.add(_album(2)), throwsUnsupportedError);
    });

    test('a later change to the source list does not leak in', () {
      final source = [_album(1)];
      cache.putAlbums(source);
      source.add(_album(2));
      expect(cache.albums?.length, 1);
    });

    test('evictAlbums drops the list but keeps items', () {
      cache.putAlbums([_album(1)]);
      cache.putItems(1, [_item(10, 1)]);

      cache.evictAlbums();

      expect(cache.albums, isNull);
      expect(cache.items(1), isNotNull);
    });
  });

  group('album items', () {
    test('are missing by default', () {
      expect(cache.items(1), isNull);
    });

    test('are stored per album', () {
      cache.putItems(1, [_item(10, 1)]);
      cache.putItems(2, [_item(20, 2), _item(21, 2)]);

      expect(cache.items(1)?.map((i) => i.id), [10]);
      expect(cache.items(2)?.map((i) => i.id), [20, 21]);
      expect(cache.items(3), isNull);
    });

    test('an empty list is cached as a value, not as a miss', () {
      cache.putItems(1, const []);
      expect(cache.items(1), isEmpty);
    });

    test('the stored list is unmodifiable', () {
      cache.putItems(1, [_item(10, 1)]);
      expect(() => cache.items(1)!.add(_item(11, 1)), throwsUnsupportedError);
    });

    test('evictItems drops only that album', () {
      cache.putAlbums([_album(1), _album(2)]);
      cache.putItems(1, [_item(10, 1)]);
      cache.putItems(2, [_item(20, 2)]);

      cache.evictItems(1);

      expect(cache.items(1), isNull);
      expect(cache.items(2), isNotNull);
      expect(cache.albums, isNotNull);
    });

    test('evicting an unknown album is a no-op', () {
      cache.putItems(1, [_item(10, 1)]);
      cache.evictItems(99);
      expect(cache.items(1), isNotNull);
    });

    test("clearItems drops every album's items but keeps the list", () {
      cache.putAlbums([_album(1), _album(2)]);
      cache.putItems(1, [_item(10, 1)]);
      cache.putItems(2, [_item(20, 2)]);

      cache.clearItems();

      expect(cache.items(1), isNull);
      expect(cache.items(2), isNull);
      expect(cache.albums, isNotNull);
    });
  });

  test("clear drops the list and every album's items", () {
    cache.putAlbums([_album(1)]);
    cache.putItems(1, [_item(10, 1)]);

    cache.clear();

    expect(cache.albums, isNull);
    expect(cache.items(1), isNull);
  });
}
