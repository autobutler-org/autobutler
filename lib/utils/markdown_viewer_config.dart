/// Tuning values for the Markdown viewer (#2987).
class MarkdownViewerConfig {
  MarkdownViewerConfig._();

  /// The most bytes of a file the rendered view reads. Markdown is parsed
  /// whole, so a longer file is rendered up to here and says so; the source
  /// view pages through all of it.
  static const int maxBytes = 1024 * 1024;
}
