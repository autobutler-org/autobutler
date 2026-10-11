/// Tuning values for the code viewer (#2991).
class CodeViewerConfig {
  CodeViewerConfig._();

  /// How many bytes of the file one request reads. The viewer holds what it
  /// has read so far, never the whole file up front.
  static const int pageBytes = 256 * 1024;

  /// The longest run of bytes without a newline kept as one line. A minified
  /// bundle or a binary file opened by mistake is cut into lines this long
  /// rather than laid out as a single multi-megabyte paragraph.
  static const int maxLineBytes = 16 * 1024;

  /// The most search hits kept. Past this the count stops, so a one-letter
  /// query over a large log stays cheap.
  static const int maxSearchMatches = 5000;
}
