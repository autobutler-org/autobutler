import 'package:cached_network_image/cached_network_image.dart';
import 'package:flutter/painting.dart';
import 'package:flutter_cache_manager/flutter_cache_manager.dart';
import 'package:quark/services/files_service.dart';
import 'package:quark/utils/thumbnail_cache_config.dart';

/// The one disk cache every thumbnail in the app goes through (#1777).
///
/// Keyed by [FilesService.thumbnailCacheKey] rather than the URL, so a new
/// session token after a re-login does not throw the whole cache away, and
/// budgeted for a photo grid rather than `DefaultCacheManager`'s 200 objects.
class ThumbnailCacheManager {
  ThumbnailCacheManager._();

  static CacheManager? _instance;

  /// The shared manager, created on first use so importing this file costs
  /// nothing.
  static CacheManager get instance => _instance ??= CacheManager(
    Config(
      ThumbnailCacheConfig.cacheKey,
      stalePeriod: ThumbnailCacheConfig.stalePeriod,
      maxNrOfCacheObjects: ThumbnailCacheConfig.maxObjects,
    ),
  );

  /// Drops every cached copy of one photo's thumbnails, on disk and decoded in
  /// memory, so the next load fetches it again. Covers both sizes the app
  /// requests and the resized decode the grids keep.
  static Future<void> evict(String filePath, {String? serial}) async {
    for (final size in const <String?>[null, 'sm']) {
      final url = FilesService.constructThumbnailUrl(
        filePath,
        serial: serial,
        size: size,
      ).toString();
      final key = FilesService.thumbnailCacheKey(
        filePath,
        serial: serial,
        size: size,
      );
      await CachedNetworkImage.evictFromCache(
        url,
        cacheKey: key,
        cacheManager: instance,
      );
      final provider = CachedNetworkImageProvider(url, cacheKey: key);
      await provider.evict();
      await ResizeImage(
        provider,
        width: ThumbnailCacheConfig.gridTileDecodeWidth,
      ).evict();
    }
  }
}
