import 'package:quark/controllers/albums_cache.dart';
import 'package:quark/controllers/file_browser_cache.dart';
import 'package:quark/controllers/file_type_listing_cache.dart';
import 'package:quark/controllers/photos_list_cache.dart';

/// Every process-wide listing cache, cleared together.
///
/// The caches are keyed by host, not by user, so nothing about them expires
/// when a session does. A session ends three ways — an explicit logout,
/// `AuthService._forgetLocalSession` (which both account deletion and Quark
/// reset go through), and a 401 — and a cache left holding data on any one of
/// them shows the next person to sign in the previous one's file tree.
///
/// Registering a cache here is what keeps that from depending on someone
/// remembering all three call sites. Add the cache to [clearAll]; do not add a
/// fourth hand-written clear site.
abstract final class AppCaches {
  /// Drops every cached listing.
  ///
  /// Call from every path that ends a session or points the app at a
  /// different Quark. Safe to call when the caches are already empty.
  static void clearAll() {
    FileBrowserCache.instance.clear();
    AlbumsCache.instance.clear();
    PhotosListCache.instance.clear();
    FileTypeListingCache.instance.clear();
  }
}
