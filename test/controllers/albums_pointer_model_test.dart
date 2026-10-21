import 'package:flutter_test/flutter_test.dart';
import 'package:photo_manager/photo_manager.dart';
import 'package:quark/controllers/albums_cache.dart';
import 'package:quark/controllers/photo_bytes_cache.dart';
import 'package:quark/controllers/photos_controller.dart';
import 'package:quark/controllers/photos_list_cache.dart';
import 'package:quark/models/paginated_photos_response.dart' as wire;
import 'package:quark/models/photo_album.dart';
import 'package:quark/models/photo_sort.dart';
import 'package:quark/services/app_settings.dart';
import 'package:quark_widgets/quark_widgets.dart';

/// A Quark that keeps albums the way the server does (#3488): an album holds
/// pointers to files, its count is how many pointers it holds, deleting a
/// file drops every pointer to it, and a copy is a new file no album points
/// at. `pointer_model_test.go` pins the server half; this is the app's.
class _PointerQuark {
  _PointerQuark(List<String> files) : files = {...files};

  /// Every photo on the internal drive, by path.
  final Set<String> files;

  /// Each album's pointers, in the order they were added.
  final Map<int, List<String>> pointers = {1: [], 2: []};

  static const _names = {1: 'Trips', 2: 'Best'};

  final scope = 'quark-a';
  late final listCache = PhotosListCache(scope: () => scope);
  late final albumsCache = AlbumsCache(scope: () => scope);

  /// Sorts [paths] the way the Quark sorts a listing: by name, or by the
  /// order they were added for anything else.
  List<String> _sorted(
    Iterable<String> paths,
    PhotoSortField sort,
    PhotoSortOrder order,
  ) {
    final list = paths.toList();
    if (sort == PhotoSortField.name) list.sort();
    return order == PhotoSortOrder.desc ? list.reversed.toList() : list;
  }

  PhotosController controller() => PhotosController(
    isWeb: false,
    listCache: listCache,
    albumsCache: albumsCache,
    activeHost: () => 'https://quark.local',
    searchDelay: Duration.zero,
    pendingDatesDelay: Duration.zero,
    bytesCache: PhotoBytesCache.instance,
    getPhotos:
        ({
          int offset = 0,
          int limit = 50,
          String? serial,
          PhotoSortField sort = PhotoSortField.added,
          PhotoSortOrder order = PhotoSortOrder.desc,
        }) async => wire.PaginatedPhotosResponse(
          photos: [
            for (final path in _sorted(files, sort, order))
              wire.PhotoItem(
                relPath: path,
                fileName: path,
                size: 1,
                mtime: 0,
                serial: '',
              ),
          ],
          total: files.length,
          offset: 0,
          limit: limit,
        ),
    loadDeviceAssets: () async => const <AssetEntity>[],
    listFavoriteKeys: () async => {},
    thumbnailUrl: (path, {String? serial, String? size}) =>
        Uri.parse('https://quark.local/thumb/$path'),
    listAlbums: ({bool tree = false}) async => [
      for (final MapEntry(key: id, value: held) in pointers.entries)
        PhotoAlbum(
          id: id,
          name: _names[id]!,
          createdAt: DateTime(2024),
          updatedAt: DateTime(2024),
          itemCount: held.length,
        ),
    ],
    addPhotoToAlbum:
        (albumId, {required deviceSerial, required relPath}) async {
          final held = pointers[albumId]!;
          if (held.contains(relPath)) return false;
          held.add(relPath);
          // What AlbumService does for the real call.
          albumsCache.dropItems(albumId);
          return true;
        },
    removePhotoFromAlbum:
        (albumId, {required deviceSerial, required relPath}) async {
          pointers[albumId]!.remove(relPath);
          albumsCache.dropItems(albumId);
        },
    listAlbumItems:
        (
          albumId, {
          PhotoSortField sort = PhotoSortField.added,
          PhotoSortOrder order = PhotoSortOrder.desc,
        }) async => [
          for (final (i, path) in _sorted(
            pointers[albumId]!,
            sort,
            order,
          ).indexed)
            PhotoAlbumItem(
              id: i,
              albumId: albumId,
              deviceSerial: '',
              relPath: path,
              addedAt: DateTime(2024),
            ),
        ],
    copyPhoto: (relPath, {serial}) async {
      final copy = relPath.replaceFirst('.jpg', ' (copy).jpg');
      files.add(copy);
      return copy;
    },
    deleteFile: (rootDir, fileName, {deviceSerial}) async {
      final path = rootDir.isEmpty ? fileName : '$rootDir/$fileName';
      files.remove(path);
      for (final held in pointers.values) {
        held.remove(path);
      }
    },
  );
}

/// Each album's count as the sidebar shows it, by id.
Map<int, int> _counts(PhotosController controller) => {
  for (final album in controller.albums) album.id: album.itemCount,
};

/// The names of the photos the grid shows, in order.
List<String> _shown(PhotosController controller) => [
  for (final photo in controller.photos) photo.name,
];

/// Selects [path] in the library and adds it to [albumId].
Future<void> _add(PhotosController controller, int albumId, String path) async {
  controller.enterSelectionMode();
  controller.toggleSelection(':$path');
  final outcome = await controller.addSelectedToAlbum(albumId);
  expect(outcome.added, 1, reason: 'adding $path to album $albumId');
}

void main() {
  setUp(() {
    PhotoBytesCache.instance.clear();
    AppSettings.instance.photoSortField.value = PhotoSortField.added;
    AppSettings.instance.photoSortOrder.value = PhotoSortOrder.asc;
    AppSettings.instance.albumSort.value = AlbumSort.nameAsc;
  });

  test('a photo in two albums counts in both, and removing it from one '
      'leaves it in the other and in the library (#3522)', () async {
    final quark = _PointerQuark(['a.jpg', 'b.jpg']);
    final controller = quark.controller();
    await controller.refresh();

    await _add(controller, 1, 'a.jpg');
    await _add(controller, 2, 'a.jpg');
    await _add(controller, 1, 'b.jpg');
    expect(_counts(controller), {1: 2, 2: 1});

    await controller.showAlbum(1);
    await controller.removeFromSelectedAlbum(':a.jpg');
    expect(_counts(controller), {1: 1, 2: 1});
    expect(_shown(controller), ['b.jpg']);

    await controller.showAlbum(2);
    expect(_shown(controller), ['a.jpg']);
    await controller.showAlbum(null);
    expect(_shown(controller), ['a.jpg', 'b.jpg']);
  });

  test('deleting a photo drops it from every album and every count, '
      'including an album opened before (#3522)', () async {
    final quark = _PointerQuark(['a.jpg', 'b.jpg']);
    final controller = quark.controller();
    await controller.refresh();
    await _add(controller, 1, 'a.jpg');
    await _add(controller, 1, 'b.jpg');
    await _add(controller, 2, 'a.jpg');
    // Opened once, so its items are kept for the next visit.
    await controller.showAlbum(2);
    expect(_shown(controller), ['a.jpg']);

    await controller.showAlbum(1);
    await controller.deletePhoto(':a.jpg');
    expect(_counts(controller), {1: 1, 2: 0});
    expect(_shown(controller), ['b.jpg']);

    final opening = controller.showAlbum(2);
    expect(
      _shown(controller),
      isEmpty,
      reason: 'the first frame shows no item kept from before the delete',
    );
    await opening;
    expect(_shown(controller), isEmpty);
  });

  test('a copy is its own photo: the album keeps only the original, and '
      'deleting the original leaves the copy (#3522)', () async {
    final quark = _PointerQuark(['a.jpg']);
    final controller = quark.controller();
    await controller.refresh();
    await _add(controller, 1, 'a.jpg');

    await controller.copyPhoto(':a.jpg');
    expect(_counts(controller), {1: 1, 2: 0});
    await controller.showAlbum(1);
    expect(_shown(controller), ['a.jpg']);

    await controller.deletePhoto(':a.jpg');
    expect(_counts(controller), {1: 0, 2: 0});
    await controller.showAlbum(null);
    expect(_shown(controller), ['a (copy).jpg']);
  });

  test('the album cover is its first photo in the sort, and falls back to '
      'the next real photo when that one is deleted (#3522)', () async {
    final quark = _PointerQuark(['a.jpg', 'b.jpg', 'c.jpg']);
    final controller = quark.controller();
    await controller.refresh();
    for (final path in ['b.jpg', 'a.jpg', 'c.jpg']) {
      await _add(controller, 1, path);
    }
    await controller.showAlbum(1);
    String? cover() =>
        controller.thumbnailUrl(controller.photos.first.id)?.path;

    expect(_shown(controller), ['b.jpg', 'a.jpg', 'c.jpg']);
    expect(cover(), '/thumb/b.jpg');

    await controller.setSort(PhotoSortField.name, PhotoSortOrder.asc);
    expect(_shown(controller), ['a.jpg', 'b.jpg', 'c.jpg']);
    expect(cover(), '/thumb/a.jpg');
    await controller.setSort(PhotoSortField.name, PhotoSortOrder.desc);
    expect(cover(), '/thumb/c.jpg');

    await controller.setSort(PhotoSortField.name, PhotoSortOrder.asc);
    await controller.deletePhoto(':a.jpg');
    expect(_shown(controller), ['b.jpg', 'c.jpg']);
    expect(cover(), '/thumb/b.jpg');

    await controller.deletePhoto(':b.jpg');
    await controller.deletePhoto(':c.jpg');
    expect(controller.photos, isEmpty);
    expect(_counts(controller)[1], 0);
  });
}
