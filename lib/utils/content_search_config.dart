/// Tuning for the content-search memo in `ContentSearchService` (#1780).
abstract final class ContentSearchConfig {
  /// How many recent queries keep their results in memory.
  ///
  /// A search box is retyped far more than it is typed: backspacing over a
  /// word replays the queries that were just answered. Thirty-two covers a
  /// full session of that at a few hundred bytes per entry, which is nothing
  /// next to the FTS scan each one would otherwise repeat on the Quark.
  static const int recentQueryLimit = 32;
}
