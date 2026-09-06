import 'package:quark/controllers/albums_cache.dart';
import 'package:quark/controllers/file_browser_cache.dart';
import 'package:quark/controllers/file_type_listing_cache.dart';
import 'package:quark/controllers/photos_list_cache.dart';
import 'package:quark/utils/listing_snapshot_store.dart';

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

  /// Everything [clearAll] drops, plus the on-disk snapshot for [hostKey].
  ///
  /// This is the clear for a session ending. [clearAll] is the clear for a
  /// host switch, and the difference is deliberate: a snapshot is per host and
  /// is meant to survive a switch, so switching away must not delete it, while
  /// a session ending must — otherwise the listings outlive the session on
  /// disk and are hydrated back on the next launch.
  static Future<void> endSession(String? hostKey) async {
    clearAll();
    if (hostKey != null) {
      await ListingSnapshots.instance.removeHost(hostKey);
    }
  }
}
