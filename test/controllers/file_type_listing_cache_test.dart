import 'package:flutter_test/flutter_test.dart';
import 'package:quark/controllers/file_type_listing_cache.dart';
import 'package:quark/models/file_node.dart';

/// #1780: Docs and Sheets rescanned the whole tree behind a spinner on every
/// visit. This is the cache that lets them show the last listing on the first
/// frame and refresh behind it.
void main() {
  final cache = FileTypeListingCache.instance;

  FileNode node(String name) => FileNode(
    name: name,
    size: 1,
    isDir: false,
    deviceName: 'Quark',
    devicePath: '/',
    deviceSerial: '',
    dirPath: name,
  );

  setUp(cache.clear);
  tearDown(cache.clear);

  test('nothing is cached until a listing is put', () {
    expect(cache.get('qdoc'), isNull);
  });

  test('a listing is found again by its file type', () {
    cache.put('qdoc', [node('a.qdoc'), node('b.qdoc')]);

    expect(cache.get('qdoc')?.map((f) => f.name), ['a.qdoc', 'b.qdoc']);
  });

  test('each file type has its own listing', () {
    cache.put('qdoc', [node('a.qdoc')]);
    cache.put('qsheet', [node('a.qsheet')]);

    expect(cache.get('qdoc')?.single.name, 'a.qdoc');
    expect(cache.get('qsheet')?.single.name, 'a.qsheet');
  });

  test('an empty listing is cached as empty, not as missing', () {
    cache.put('qdoc', const []);

    expect(cache.get('qdoc'), isEmpty);
  });

  test('a later put replaces the listing', () {
    cache.put('qdoc', [node('a.qdoc')]);
    cache.put('qdoc', [node('b.qdoc')]);

    expect(cache.get('qdoc')?.single.name, 'b.qdoc');
  });

  test(
    'the cached listing is not affected by the caller mutating its list',
    () {
      final files = [node('a.qdoc')];
      cache.put('qdoc', files);
      files.add(node('b.qdoc'));

      expect(cache.get('qdoc'), hasLength(1));
      expect(
        () => cache.get('qdoc')!.add(node('c.qdoc')),
        throwsUnsupportedError,
      );
    },
  );

  test('evict drops one type and leaves the others', () {
    cache.put('qdoc', [node('a.qdoc')]);
    cache.put('qsheet', [node('a.qsheet')]);

    cache.evict('qdoc');

    expect(cache.get('qdoc'), isNull);
    expect(cache.get('qsheet'), isNotNull);
  });

  test('clear drops every type', () {
    cache.put('qdoc', [node('a.qdoc')]);
    cache.put('qsheet', [node('a.qsheet')]);

    cache.clear();

    expect(cache.get('qdoc'), isNull);
    expect(cache.get('qsheet'), isNull);
  });
}
