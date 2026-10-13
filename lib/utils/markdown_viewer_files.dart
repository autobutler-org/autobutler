import 'package:quark/utils/file_kind.dart';

/// Whether the file called [name] opens in the rendered Markdown viewer
/// (#2987) and not the plaintext editor.
bool opensInMarkdownViewer(String name) =>
    const {'.md', '.markdown'}.contains(fileExtension(name));

/// What a link or image address in a Markdown file points at.
enum MarkdownLinkKind {
  /// A file on the Quark; the value is its files-relative path.
  file,

  /// An `http` or `https` address; the value is that address.
  remote,

  /// Anything else — an in-page anchor, `mailto:`, `javascript:`, `data:`,
  /// something unparseable. The viewer does nothing with these.
  other,
}

/// Works out what [href], written in the Markdown file at [filePath], points
/// at. A relative address is resolved against the file's folder, one starting
/// with `/` against the files root; `..` cannot climb above that root.
({MarkdownLinkKind kind, String value}) resolveMarkdownLink(
  String filePath,
  String href,
) {
  const other = (kind: MarkdownLinkKind.other, value: '');
  final Uri uri;
  try {
    uri = Uri.parse(href.trim());
  } on FormatException {
    return other;
  }
  if (uri.isScheme('http') || uri.isScheme('https')) {
    return uri.host.isEmpty
        ? other
        : (kind: MarkdownLinkKind.remote, value: uri.toString());
  }
  if (uri.hasScheme || uri.hasAuthority || uri.path.isEmpty) return other;
  final segments = [
    ...filePath.split('/').where((segment) => segment.isNotEmpty),
  ];
  final base = Uri(pathSegments: ['', ...segments]);
  final resolved = Uri.decodeComponent(
    base.resolve(uri.path).path,
  ).replaceFirst(RegExp(r'^/+'), '');
  return resolved.isEmpty
      ? other
      : (kind: MarkdownLinkKind.file, value: resolved);
}
