import 'dart:convert';
import 'dart:io';

import 'package:flutter_test/flutter_test.dart';
import 'package:photo_manager/photo_manager.dart';
import 'package:quark/controllers/albums_cache.dart';
import 'package:quark/controllers/file_browser_cache.dart';
import 'package:quark/controllers/listing_snapshot_hydration.dart';
import 'package:quark/controllers/photos_list_cache.dart';
import 'package:quark/models/file_node.dart';
import 'package:quark/models/photo_album.dart';
import 'package:quark/pages/photos_page.dart';
import 'package:quark/utils/listing_snapshot.dart';
import 'package:quark/utils/listing_snapshot_config.dart';
import 'package:quark/utils/listing_snapshot_store.dart';
import 'package:quark/utils/listing_snapshot_store_io.dart';

/// Counts what reaches the disk, so a test can tell a skipped write from a
/// written one.
class _CountingStore implements ListingSnapshotStore {
  _CountingStore(this.inner);

  final ListingSnapshotStore inner;
  int writes = 0;

  @override
  Future<String?> read(String hostKey, String name) =>
      inner.read(hostKey, name);

  @override
  Future<void> write(String hostKey, String name, String contents) {
    writes++;
    return inner.write(hostKey, name, contents);
  }

  @override
  Future<void> remove(String hostKey, String name) =>
      inner.remove(hostKey, name);

  @override
  Future<void> removeHost(String hostKey) => inner.removeHost(hostKey);
}

/// #1781: every in-memory cache died with the process, so a cold launch on a
/// phone showed blank pages until the first listing round-tripped. These
/// snapshots are what the first frame draws instead.
void main() {
  final snapshots = ListingSnapshots.instance;
  final files = FileBrowserCache.instance;
  final photos = PhotosListCache.instance;
  final albums = AlbumsCache.instance;

  const hostA = 'https://one.local';
  const hostB = 'https://two.local:8443';

  late Directory temp;
  late _CountingStore store;
  late ListingSnapshotStore previousStore;

  String directoryOf(String host) => listingSnapshotDirectoryName(host);

  File fileOf(String host, String name) =>
      File('${temp.path}/${directoryOf(host)}/$name.json');

  void clearCaches() {
    files.clear();
    photos.clear();
    albums.clear();
  }

  setUp(() async {
    temp = await Directory.systemTemp.createTemp('listing_snapshot_test');
    store = _CountingStore(
      FileListingSnapshotStore(baseDirectory: () async => temp),
    );
    previousStore = snapshots.store;
    snapshots.store = store;
    snapshots.setHost(hostA);
    clearCaches();
  });

  tearDown(() async {
    snapshots.setHost(null);
    snapshots.store = previousStore;
    clearCaches();
    await temp.delete(recursive: true);
  });

  FileNode node(int i, {bool isDir = false}) => FileNode(
    name: 'item$i',
    size: 100 + i,
    compressedSize: 50 + i,
    isDir: isDir,
    deviceName: 'Drive $i',
    devicePath: '/mnt/drive$i',
    deviceSerial: 'serial$i',
    dirPath: '/folder/item$i',
    fileType: isDir ? '' : 'image',
  );

  void expectSameNode(FileNode actual, FileNode expected) {
    expect(actual.name, expected.name);
    expect(actual.size, expected.size);
    expect(actual.compressedSize, expected.compressedSize);
    expect(actual.isDir, expected.isDir);
    expect(actual.deviceName, expected.deviceName);
    expect(actual.devicePath, expected.devicePath);
    expect(actual.deviceSerial, expected.deviceSerial);
    expect(actual.dirPath, expected.dirPath);
    expect(actual.fileType, expected.fileType);
  }

  PhotoAlbum album(int id, {List<PhotoAlbum> children = const []}) =>
      PhotoAlbum(
        id: id,
        name: 'Album $id',
        parentId: id > 10 ? 1 : null,
        smartType: id == 1 ? 'favorites' : null,
        createdAt: DateTime.utc(2024, 1, id),
        updatedAt: DateTime.utc(2024, 2, id, 12, 30),
        itemCount: id * 3,
        children: children,
      );

  void expectSameAlbum(PhotoAlbum actual, PhotoAlbum expected) {
    expect(actual.id, expected.id);
    expect(actual.name, expected.name);
    expect(actual.parentId, expected.parentId);
    expect(actual.smartType, expected.smartType);
    expect(actual.isFavorites, expected.isFavorites);
    expect(actual.createdAt, expected.createdAt);
    expect(actual.updatedAt, expected.updatedAt);
    expect(actual.itemCount, expected.itemCount);
    expect(actual.children.length, expected.children.length);
    for (var i = 0; i < expected.children.length; i++) {
      expectSameAlbum(actual.children[i], expected.children[i]);
    }
  }

  group('root files', () {
    test('round-trips every field through the disk', () async {
      final listing = [node(1, isDir: true), node(2)];
      files.put(FileBrowserCache.rootKey, listing);
      await snapshots.flush();
      expect(
        fileOf(hostA, ListingSnapshotNames.rootFiles).existsSync(),
        isTrue,
      );

      files.clear();
      await files.hydrate();

      final restored = files.get(FileBrowserCache.rootKey)!;
      expect(restored.length, 2);
      expectSameNode(restored[0], listing[0]);
      expectSameNode(restored[1], listing[1]);
    });

    test('only the root listing is written', () async {
      files.put('/folder', [node(1)]);
      await snapshots.flush();

      expect(store.writes, 0);
      expect(
        Directory('${temp.path}/${directoryOf(hostA)}').existsSync(),
        isFalse,
      );
    });

    test('is bounded by the configured item count', () async {
      files.put(
        FileBrowserCache.rootKey,
        List.generate(ListingSnapshotConfig.maxRootFiles + 5, node),
      );
      await snapshots.flush();
      files.clear();
      await files.hydrate();

      expect(
        files.get(FileBrowserCache.rootKey)!.length,
        ListingSnapshotConfig.maxRootFiles,
      );
    });
  });

  group('photos', () {
    test('round-trips the list, its paging and the favorites', () async {
      final items = [
        PhotoItem.fromFiles(node(1), hasLiveVideo: true),
        PhotoItem.fromFiles(node(2)),
      ];
      photos.put(items, total: 40, offset: 2);
      photos.setFavoriteKeys({items[1].selectionKey});
      await snapshots.flush();

      photos.clear();
      await photos.hydrate();

      final restored = photos.list!;
      expect(restored.total, 40);
      expect(restored.offset, 2);
      expect(restored.photos.length, 2);
      expect(restored.photos[0].isFiles, isTrue);
      expect(restored.photos[0].hasLiveVideo, isTrue);
      expect(restored.photos[1].hasLiveVideo, isFalse);
      expectSameNode(restored.photos[0].quark!, items[0].quark!);
      expectSameNode(restored.photos[1].quark!, items[1].quark!);
      expect(restored.photos[1].selectionKey, items[1].selectionKey);
      expect(photos.favoriteKeys, {items[1].selectionKey});
    });

    test('never writes a device asset', () async {
      photos.put(
        [
          PhotoItem.fromFiles(node(1)),
          PhotoItem.fromAsset(
            AssetEntity(id: 'asset-1', typeInt: 1, width: 1, height: 1),
          ),
        ],
        total: 2,
        offset: 2,
      );
      await snapshots.flush();

      final written = jsonDecode(
        fileOf(hostA, ListingSnapshotNames.photos).readAsStringSync(),
      );
      expect(written['photos'], hasLength(1));
      expect(written['photos'][0]['file']['name'], 'item1');
    });

    test('keeps the first pages and moves the offset to match', () async {
      final cap = ListingSnapshotConfig.maxPhotos;
      photos.put(
        List.generate(cap + 20, (i) => PhotoItem.fromFiles(node(i))),
        total: cap + 200,
        offset: cap + 20,
      );
      await snapshots.flush();
      photos.clear();
      await photos.hydrate();

      expect(photos.list!.photos.length, cap);
      expect(photos.list!.offset, cap);
      expect(photos.list!.total, cap + 200);
    });

    test('a favorite toggle reaches the disk', () async {
      photos.put([PhotoItem.fromFiles(node(1))], total: 1, offset: 1);
      await snapshots.flush();
      photos.setFavorite('serial1:folder/item1', true);
      await snapshots.flush();

      photos.clear();
      await photos.hydrate();
      expect(photos.favoriteKeys, {'serial1:folder/item1'});
    });
  });

  group('albums', () {
    test('round-trips the tree with its nested children', () async {
      final tree = [
        album(1),
        album(2, children: [album(11), album(12)]),
      ];
      albums.putAlbums(tree);
      await snapshots.flush();

      albums.clear();
      await albums.hydrate();

      final restored = albums.albums!;
      expect(restored.length, 2);
      expectSameAlbum(restored[0], tree[0]);
      expectSameAlbum(restored[1], tree[1]);
    });

    test('is bounded by the configured item count', () async {
      albums.putAlbums(
        List.generate(ListingSnapshotConfig.maxAlbums + 3, album),
      );
      await snapshots.flush();
      albums.clear();
      await albums.hydrate();

      expect(albums.albums!.length, ListingSnapshotConfig.maxAlbums);
    });
  });

  group('hydration', () {
    test('fills all three caches at once', () async {
      files.put(FileBrowserCache.rootKey, [node(1)]);
      photos.put([PhotoItem.fromFiles(node(2))], total: 1, offset: 1);
      albums.putAlbums([album(1)]);
      await snapshots.flush();
      clearCaches();

      await hydrateListingSnapshots();

      expect(files.get(FileBrowserCache.rootKey), hasLength(1));
      expect(photos.list!.photos, hasLength(1));
      expect(albums.albums, hasLength(1));
    });

    test('does nothing when there is no snapshot', () async {
      await hydrateListingSnapshots();

      expect(files.get(FileBrowserCache.rootKey), isNull);
      expect(photos.list, isNull);
      expect(photos.favoriteKeys, isEmpty);
      expect(albums.albums, isNull);
    });

    test('never overwrites what a fetch already put', () async {
      files.put(FileBrowserCache.rootKey, [node(1), node(2)]);
      albums.putAlbums([album(1), album(2)]);
      await snapshots.flush();
      clearCaches();
      files.put(FileBrowserCache.rootKey, [node(9)]);
      albums.putAlbums([album(9)]);

      await hydrateListingSnapshots();

      expect(files.get(FileBrowserCache.rootKey)!.single.name, 'item9');
      expect(albums.albums!.single.id, 9);
    });

    test('does not rewrite what it just read', () async {
      files.put(FileBrowserCache.rootKey, [node(1)]);
      await snapshots.flush();
      expect(store.writes, 1);
      files.clear();
      await files.hydrate();

      files.put(FileBrowserCache.rootKey, files.get(FileBrowserCache.rootKey)!);
      await snapshots.flush();

      expect(store.writes, 1);
    });
  });

  group('corrupt files', () {
    test('a file that is not JSON is deleted and ignored', () async {
      final file = fileOf(hostA, ListingSnapshotNames.rootFiles);
      file.createSync(recursive: true);
      file.writeAsStringSync('{not json');

      await files.hydrate();

      expect(files.get(FileBrowserCache.rootKey), isNull);
      expect(file.existsSync(), isFalse);
    });

    test('a JSON file of the wrong shape is deleted and ignored', () async {
      final file = fileOf(hostA, ListingSnapshotNames.albums);
      file.createSync(recursive: true);
      file.writeAsStringSync('{"version":1,"albums":"nope"}');

      await albums.hydrate();

      expect(albums.albums, isNull);
      expect(file.existsSync(), isFalse);
    });

    test('a snapshot from another version is deleted and ignored', () async {
      final file = fileOf(hostA, ListingSnapshotNames.photos);
      file.createSync(recursive: true);
      file.writeAsStringSync(
        '{"version":99,"total":0,"offset":0,"photos":[],"favoriteKeys":[]}',
      );

      await photos.hydrate();

      expect(photos.list, isNull);
      expect(file.existsSync(), isFalse);
    });
  });

  group('per-host isolation', () {
    test('one host never sees another host\'s snapshot', () async {
      files.put(FileBrowserCache.rootKey, [node(1)]);
      await snapshots.flush();
      files.clear();

      snapshots.setHost(hostB);
      await files.hydrate();
      expect(files.get(FileBrowserCache.rootKey), isNull);

      snapshots.setHost(hostA);
      await files.hydrate();
      expect(files.get(FileBrowserCache.rootKey), hasLength(1));
    });

    test('a write still pending when the host switches is dropped', () async {
      files.put(FileBrowserCache.rootKey, [node(1)]);
      snapshots.setHost(hostB);
      await snapshots.flush();

      expect(
        fileOf(hostA, ListingSnapshotNames.rootFiles).existsSync(),
        isFalse,
      );
      expect(
        fileOf(hostB, ListingSnapshotNames.rootFiles).existsSync(),
        isFalse,
      );
    });

    test('nothing is written without an active host', () async {
      snapshots.setHost(null);
      files.put(FileBrowserCache.rootKey, [node(1)]);
      await snapshots.flush();

      expect(store.writes, 0);
    });

    test('directory names are filesystem-safe and distinct', () {
      final a = directoryOf(hostA);
      final b = directoryOf(hostB);
      expect(a, isNot(contains('/')));
      expect(a, isNot(contains(':')));
      expect(b, isNot(contains('/')));
      expect(b, isNot(contains(':')));
      expect(a, isNot(b));
      expect(directoryOf('https://a:1'), isNot(directoryOf('https://a_1')));
      expect(directoryOf('..'), isNot('..'));
      expect(directoryOf(''), isNotEmpty);
    });
  });

  group('removal', () {
    test('removing a host deletes its directory and nothing else', () async {
      files.put(FileBrowserCache.rootKey, [node(1)]);
      albums.putAlbums([album(1)]);
      await snapshots.flush();
      snapshots.setHost(hostB);
      files.put(FileBrowserCache.rootKey, [node(2)]);
      await snapshots.flush();

      await snapshots.removeHost(hostA);

      expect(
        Directory('${temp.path}/${directoryOf(hostA)}').existsSync(),
        isFalse,
      );
      expect(
        fileOf(hostB, ListingSnapshotNames.rootFiles).existsSync(),
        isTrue,
      );
    });

    test('a write pending for the removed host is dropped', () async {
      files.put(FileBrowserCache.rootKey, [node(1)]);

      await snapshots.removeHost(hostA);
      await snapshots.flush();

      expect(
        fileOf(hostA, ListingSnapshotNames.rootFiles).existsSync(),
        isFalse,
      );
    });

    test('a discarded snapshot is written again on the next put', () async {
      files.put(FileBrowserCache.rootKey, [node(1)]);
      await snapshots.flush();
      await snapshots.discard(ListingSnapshotNames.rootFiles);
      expect(
        fileOf(hostA, ListingSnapshotNames.rootFiles).existsSync(),
        isFalse,
      );

      files.put(FileBrowserCache.rootKey, [node(1)]);
      await snapshots.flush();

      expect(
        fileOf(hostA, ListingSnapshotNames.rootFiles).existsSync(),
        isTrue,
      );
    });
  });
}
