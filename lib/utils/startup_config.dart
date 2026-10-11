/// Bounds on what may hold the app's first frame at launch (#2066), read by
/// `main` and `deferFirstFrameUntilRouted`.
abstract final class StartupConfig {
  /// How long the saved settings get to load before a loader is painted over
  /// the blank native launch screen. Short enough that a stalled read never
  /// looks like a hang, long enough that a normal launch never flashes it.
  static const Duration settingsGrace = Duration(seconds: 1);

  /// The longest the first frame waits for the router's first route. Above
  /// `kAuthRequestTimeout` (5 seconds), so a signed-out launch that is only
  /// waiting on the Quark's status answer still opens straight on its page.
  static const Duration firstFrameMaxHold = Duration(seconds: 8);
}
