import 'package:flutter/foundation.dart' show immutable;

/// One search hit in a list of lines: [line] is a 0-based index into the
/// lines, and [start] and [end] are UTF-16 offsets in that line, [end]
/// exclusive.
///
/// `QuarkCodeViewer` takes a list of these and paints them. The caller
/// computes them, usually with [findCodeMatches].
@immutable
class CodeSearchMatch {
  /// Creates a hit covering `[start, end)` of line [line].
  const CodeSearchMatch({
    required this.line,
    required this.start,
    required this.end,
  });

  /// The 0-based index of the line the hit is on.
  final int line;

  /// The UTF-16 offset in the line where the hit starts.
  final int start;

  /// The UTF-16 offset in the line just past the hit.
  final int end;

  @override
  bool operator ==(Object other) =>
      other is CodeSearchMatch &&
      other.line == line &&
      other.start == start &&
      other.end == end;

  @override
  int get hashCode => Object.hash(line, start, end);

  @override
  String toString() => 'CodeSearchMatch(line: $line, $start-$end)';
}

/// Every place [query] occurs in [lines]: a case-insensitive plain-substring
/// search, with hits that do not overlap, in line order, stopping at [limit]
/// hits. An empty [query] finds nothing.
List<CodeSearchMatch> findCodeMatches(
  List<String> lines,
  String query, {
  int limit = 5000,
}) {
  final matches = <CodeSearchMatch>[];
  if (query.isEmpty) return matches;
  // A case-insensitive pattern rather than lowercasing both sides: lowercasing
  // can change a string's length, which would shift every offset after it.
  final pattern = RegExp(RegExp.escape(query), caseSensitive: false);
  for (var line = 0; line < lines.length; line++) {
    for (final match in pattern.allMatches(lines[line])) {
      if (matches.length >= limit) return matches;
      matches.add(
        CodeSearchMatch(line: line, start: match.start, end: match.end),
      );
    }
  }
  return matches;
}
