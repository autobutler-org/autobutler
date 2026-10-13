import 'package:markdown/markdown.dart' as md;

/// The attribute [QuarkMarkdownDocument.parse] writes on every link, image and
/// task checkbox: its 0-based position among elements of its kind, in document
/// order. `QuarkMarkdownView` builds its keys from it.
const String markdownIndexAttribute = 'data-quark-index';

/// Immutable parsed Markdown, the input of `QuarkMarkdownView`.
///
/// Parsing a large file is not free, so the caller parses once, in a
/// controller, and hands the result to the view; nothing is parsed in `build`.
/// The source is read as GitHub Flavored Markdown (tables, task lists,
/// strikethrough, bare links). Raw HTML is never turned into elements: it stays
/// in the tree as the text it was written as.
///
/// ```dart
/// final document = QuarkMarkdownDocument.parse('# Notes\n\nHello *there*.');
/// ```
class QuarkMarkdownDocument {
  /// A document with nothing in it, for the time before a file has loaded.
  const QuarkMarkdownDocument.empty() : blocks = const [];

  QuarkMarkdownDocument._(this.blocks);

  /// Parses [source] as GitHub Flavored Markdown.
  factory QuarkMarkdownDocument.parse(String source) {
    final blocks = md.Document(
      extensionSet: md.ExtensionSet.gitHubFlavored,
      encodeHtml: false,
    ).parse(source);
    final counts = <String, int>{};
    void number(md.Node node) {
      if (node is! md.Element) return;
      if (const {'a', 'img', 'input'}.contains(node.tag)) {
        final index = counts[node.tag] ?? 0;
        node.attributes[markdownIndexAttribute] = '$index';
        counts[node.tag] = index + 1;
      }
      node.children?.forEach(number);
    }

    blocks.forEach(number);
    return QuarkMarkdownDocument._(List.unmodifiable(blocks));
  }

  /// The top-level block nodes, in source order: `md.Element`s for headings,
  /// paragraphs, lists and the rest, and an `md.Text` for a block of raw HTML.
  final List<md.Node> blocks;

  /// Whether the source held no content at all.
  bool get isEmpty => blocks.isEmpty;
}
