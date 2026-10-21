import 'package:quark/utils/file_kind.dart';

/// Whether the file called [name] opens in the read-only document viewer
/// (#2987): a `.docx`, `.docm`, `.odt` or `.rtf`, read on the Quark.
bool opensInDocumentViewer(String name) =>
    fileKindForName(name) == FileKind.docx;

/// The address a link in a document may open, or null for one the viewer
/// ignores.
///
/// Only `http` and `https` with a host and `mailto` are followed, the
/// Markdown viewer's policy: `javascript:`, `file:`, `data:`, a relative
/// path and anything unparseable do nothing.
Uri? documentLinkUri(String href) {
  final Uri uri;
  try {
    uri = Uri.parse(href.trim());
  } on FormatException {
    return null;
  }
  if (uri.isScheme('http') || uri.isScheme('https')) {
    return uri.host.isEmpty ? null : uri;
  }
  return uri.isScheme('mailto') && uri.path.isNotEmpty ? uri : null;
}

/// Where "Open as Quark doc" writes the document at [path]: beside it, under
/// the same name with its extension swapped for `.qdoc`, as the Quark names
/// it.
String qdocPathFor(String path) {
  final slash = path.lastIndexOf('/');
  final dot = path.lastIndexOf('.');
  return '${dot > slash + 1 ? path.substring(0, dot) : path}.qdoc';
}
