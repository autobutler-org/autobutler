/// Tuning for the thumbnail disk cache the file browser, the photo grid and
/// the album page share (#1777).
abstract final class ThumbnailCacheConfig {
  /// Name of the cache on disk. Its own name rather than `DefaultCacheManager`'s
  /// so its budget is not shared with anything else that caches over the
  /// network.
  static const String cacheKey = 'quarkThumbnails';

  /// Most thumbnails kept on disk before the least recently used are dropped.
  ///
  /// A grid with a few thousand photos churns through the default 200 in one
  /// scroll. Thumbnails top out at 400x400 and tens of kilobytes each, so five
  /// thousand is a couple of hundred megabytes at the very worst and usually
  /// far less.
  static const int maxObjects = 5000;

  /// How long an unused thumbnail stays on disk before it is dropped.
  static const Duration stalePeriod = Duration(days: 90);

  /// Width, in physical pixels, a grid tile's thumbnail is decoded at.
  ///
  /// The server's largest tier is 400px. A phone tile at four columns is about
  /// 300 physical pixels wide, so 320 keeps it sharp while a decoded tile
  /// costs 400KB instead of 640KB, which is that much more scroll-back the
  /// in-memory image cache can hold.
  static const int gridTileDecodeWidth = 320;
}
