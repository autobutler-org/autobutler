/// Tuning for photo caching, prefetching and decoding in the image viewer
/// (#1710).
///
/// Gathered here rather than spread through the viewer and its cache because
/// every one of these numbers is a tradeoff someone will want to revisit, and
/// the reasoning belongs next to the value rather than in the commit that
/// changed it.
abstract final class ImageViewerConfig {
  /// Total encoded bytes the photo cache may hold before it evicts.
  ///
  /// Encoded, never decoded: a 4000x3000 photo is a few megabytes as JPEG and
  /// ~48MB as raw RGBA, so a budget that is comfortable for one is an OOM kill
  /// for the other. At the 3-8MB a phone photo typically runs, 64MiB holds
  /// somewhere between eight and twenty of them — far more than the handful a
  /// user browses back and forth across, and small enough to sit inside a
  /// low-end device's budget alongside the decoded frame on screen.
  static const int photoCacheBudgetBytes = 64 * 1024 * 1024;

  /// How many photos either side of the current one are fetched in the
  /// background.
  ///
  /// One. The next action is almost always the next or previous photo, and
  /// every step past that spends a stranger's cellular data on a photo they
  /// may never reach. Swiping faster than one photo per download costs nothing
  /// either: navigations coalesce, and the bytes a superseded one downloaded
  /// still land in the cache, so a fast swipe warms everything it passes.
  static const int prefetchWindow = 1;

  /// Zoom scale above which the photo counts as magnified.
  ///
  /// Exactly 1.0 — `InteractiveViewer` sits there until the user pinches, so
  /// crossing it is the same instant for both things that care: the page
  /// physics has to stop scrolling so a drag pans instead of turning the page,
  /// and the downscaled decode has to give way to the full-resolution one
  /// before anything looks soft.
  static const double zoomedInScale = 1.0;
}
