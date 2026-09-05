import 'package:flutter_test/flutter_test.dart';
import 'package:quark/models/file_node.dart';
import 'package:quark/pages/photos_page.dart';
import 'package:quark/services/app_settings.dart';
import 'package:quark/services/demo_photos_service.dart';

/// #1746: the sample library is hand-listed Dart pointing at bundled files, so
/// the two can drift apart silently — a renamed asset would only show up as a
/// grey tile in a demo. These pin the catalog to the bundle and to the keys
/// the Photos page derives from it.
void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  final photos = DemoPhotosService.photos;

  tearDown(() => AppSettings.instance.setDemoMode(false));

  test('follows the demo mode switch', () async {
    expect(DemoPhotosService.isEnabled, isFalse);

    await AppSettings.instance.setDemoMode(true);

    expect(DemoPhotosService.isEnabled, isTrue);
  });

  test('every listed photo is a bundled asset', () async {
    expect(photos, isNotEmpty);
    for (final photo in photos) {
      expect(photo.relPath, startsWith('${DemoPhotosService.assetDir}/'));
      expect(photo.relPath, endsWith('/${photo.fileName}'));
      final bytes = await DemoPhotosService.loadBytes(photo.relPath);
      expect(bytes, isNotEmpty, reason: '${photo.relPath} is not bundled');
    }
  });

  test('every photo carries the demo serial', () {
    for (final photo in photos) {
      expect(DemoPhotosService.isDemoSerial(photo.serial), isTrue);
    }
    expect(DemoPhotosService.isDemoSerial(''), isFalse);
    expect(DemoPhotosService.isDemoSerial('ABC123'), isFalse);
    expect(DemoPhotosService.isDemoSerial(null), isFalse);
  });

  test('a single page covers the whole library, so nothing pages further', () {
    final response = DemoPhotosService.getPhotos();

    expect(response.photos, photos);
    expect(response.total, photos.length);
    expect(response.offset, 0);
    expect(response.limit, greaterThanOrEqualTo(photos.length));
  });

  test('favorite keys match how the Photos page keys a photo', () {
    final keys = DemoPhotosService.favoriteKeys();
    final pageKeys = {
      for (final p in photos)
        PhotoItem.fromFiles(
          FileNode(
            name: p.fileName,
            size: p.size,
            isDir: false,
            deviceName: '',
            devicePath: '',
            deviceSerial: p.serial,
            dirPath: p.relPath,
          ),
        ).selectionKey,
    };

    expect(keys, isNotEmpty);
    expect(pageKeys, containsAll(keys));
  });

  test('albums only ever point at listed photos', () {
    final albums = DemoPhotosService.listAlbums();
    final relPaths = photos.map((p) => p.relPath).toSet();

    expect(albums, isNotEmpty);
    for (final album in albums) {
      expect(album.id, isNegative, reason: 'a demo id must not collide');
      final items = DemoPhotosService.listAlbumItems(album.id);
      expect(items.length, album.itemCount, reason: album.name);
      for (final item in items) {
        expect(item.albumId, album.id);
        expect(DemoPhotosService.isDemoSerial(item.deviceSerial), isTrue);
        expect(relPaths, contains(item.relPath));
      }
    }
  });

  test('the favorites album mirrors the favorite keys', () {
    final favorites = DemoPhotosService.listAlbums().singleWhere(
      (a) => a.isFavorites,
    );
    final items = DemoPhotosService.listAlbumItems(favorites.id);

    expect(
      items.map((i) => DemoPhotosService.selectionKey(i.relPath)).toSet(),
      DemoPhotosService.favoriteKeys(),
    );
  });

  test('an unknown album is empty rather than an error', () {
    expect(DemoPhotosService.listAlbumItems(42), isEmpty);
  });
}
