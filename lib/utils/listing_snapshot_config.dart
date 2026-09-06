/// Tuning for the listing snapshots kept on disk so a cold launch shows the
/// last Files, Photos and album content before the first fetch answers
/// (#1781).
abstract final class ListingSnapshotConfig {
  /// Directory under the application support directory that holds one
  /// subdirectory per host.
  static const String directoryName = 'listing_cache';

  /// Most entries of the root folder listing written to disk.
  static const int maxRootFiles = 1000;

  /// Most Quark-stored photos written to disk: the first pages, so the grid
  /// has something to draw while the rest loads as the reader scrolls.
  static const int maxPhotos = 100;

  /// Most favorite keys written to disk.
  static const int maxFavoriteKeys = 5000;

  /// Most top-level albums written to disk.
  static const int maxAlbums = 500;

  /// How long a burst of cache updates is left to settle before the disk is
  /// touched.
  static const Duration writeDebounce = Duration(seconds: 1);

  /// Longest startup waits on the snapshots before running the app without
  /// them; a read that finishes later still lands in the caches.
  static const Duration hydrateTimeout = Duration(seconds: 2);
}
