import 'package:flutter/services.dart';
import 'package:quark/models/paginated_photos_response.dart';
import 'package:quark/models/photo_album.dart';
import 'package:quark/services/app_settings.dart';

class DemoPhotosService {
  DemoPhotosService._();

  static const String deviceSerial = 'demo';
  static const String assetDir = 'assets/demo';

  static const int favoritesAlbumId = -1;
  static const int summerTripAlbumId = -2;
  static const int hikingAlbumId = -3;
  static const int homeAlbumId = -4;
  static const int cityBreakAlbumId = -5;

  static bool get isEnabled => AppSettings.instance.demoMode.value;

  static bool isDemoSerial(String? serial) => serial == deviceSerial;

  static final List<PhotoItem> photos = List.unmodifiable([
    _photo('sunrise-hills.jpg', 11215, DateTime.utc(2025, 6, 14, 5, 48)),
    _photo('mountain-lake.jpg', 16181, DateTime.utc(2025, 6, 15, 11, 20)),
    _photo('beach-day.jpg', 15909, DateTime.utc(2025, 7, 2, 14, 5)),
    _photo('city-night.jpg', 50777, DateTime.utc(2025, 7, 19, 22, 41)),
    _photo('forest-fog.jpg', 27697, DateTime.utc(2025, 8, 3, 7, 12)),
    _photo('balloons.jpg', 13344, DateTime.utc(2025, 8, 9, 6, 55)),
    _photo('desert-dunes.jpg', 9431, DateTime.utc(2025, 8, 23, 17, 30)),
    _photo('aurora.jpg', 26979, DateTime.utc(2025, 9, 12, 23, 58)),
    _photo('sailboat.jpg', 18007, DateTime.utc(2025, 9, 20, 18, 14)),
    _photo('lighthouse.jpg', 9639, DateTime.utc(2025, 9, 21, 8, 2)),
    _photo('flower-field.jpg', 33801, DateTime.utc(2025, 10, 4, 12, 26)),
    _photo('coffee.jpg', 20214, DateTime.utc(2025, 10, 11, 9, 15)),
  ]);

  static const List<String> _favoriteFiles = [
    'mountain-lake.jpg',
    'aurora.jpg',
    'sailboat.jpg',
  ];

  static final Map<int, List<String>> _albumFiles = {
    favoritesAlbumId: _favoriteFiles,
    summerTripAlbumId: [
      'beach-day.jpg',
      'sailboat.jpg',
      'lighthouse.jpg',
      'balloons.jpg',
      'sunrise-hills.jpg',
    ],
    hikingAlbumId: [
      'mountain-lake.jpg',
      'forest-fog.jpg',
      'aurora.jpg',
      'desert-dunes.jpg',
    ],
    homeAlbumId: ['coffee.jpg', 'flower-field.jpg'],
    cityBreakAlbumId: ['city-night.jpg'],
  };

  static PaginatedPhotosResponse getPhotos() => PaginatedPhotosResponse(
    photos: photos,
    total: photos.length,
    offset: 0,
    limit: photos.length,
  );

  static Future<Uint8List> loadBytes(String relPath) async {
    final data = await rootBundle.load(relPath);
    return data.buffer.asUint8List(data.offsetInBytes, data.lengthInBytes);
  }

  static String selectionKey(String relPath) => '$deviceSerial:$relPath';

  static Set<String> favoriteKeys() => {
    for (final file in _favoriteFiles) selectionKey(_relPath(file)),
  };

  static List<PhotoAlbum> listAlbums() => List.unmodifiable([
    _album(favoritesAlbumId, 'Favorites', smartType: 'favorites'),
    _album(summerTripAlbumId, 'Summer Trip'),
    _album(hikingAlbumId, 'Hiking'),
    _album(homeAlbumId, 'Home'),
    _album(cityBreakAlbumId, 'City Break'),
  ]);

  static List<PhotoAlbumItem> listAlbumItems(int albumId) {
    final files = _albumFiles[albumId] ?? const <String>[];
    return List.unmodifiable([
      for (final (index, file) in files.indexed)
        PhotoAlbumItem(
          id: albumId * 100 - index,
          albumId: albumId,
          deviceSerial: deviceSerial,
          relPath: _relPath(file),
          addedAt: _albumDate,
        ),
    ]);
  }

  static final DateTime _albumDate = DateTime.utc(2025, 10, 12, 10, 0);

  static String _relPath(String file) => '$assetDir/$file';

  static PhotoItem _photo(String file, int size, DateTime taken) => PhotoItem(
    relPath: _relPath(file),
    fileName: file,
    size: size,
    mtime: taken.millisecondsSinceEpoch ~/ 1000,
    serial: deviceSerial,
  );

  static PhotoAlbum _album(int id, String name, {String? smartType}) =>
      PhotoAlbum(
        id: id,
        name: name,
        smartType: smartType,
        createdAt: _albumDate,
        updatedAt: _albumDate,
        itemCount: _albumFiles[id]?.length ?? 0,
      );
}
