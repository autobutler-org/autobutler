import 'package:quark/controllers/albums_cache.dart';
import 'package:quark/controllers/file_browser_cache.dart';
import 'package:quark/controllers/photos_list_cache.dart';

/// Fills the listing caches from the active host's snapshots on disk, so the
/// first frame after a cold launch or a host switch shows the last content
/// instead of a spinner (#1781).
///
/// Only empty caches are filled, so a fetch that answered first is never
/// overwritten with older data. Never throws.
Future<void> hydrateListingSnapshots() {
  return Future.wait([
    FileBrowserCache.instance.hydrate(),
    PhotosListCache.instance.hydrate(),
    AlbumsCache.instance.hydrate(),
  ]);
}
