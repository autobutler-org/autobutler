import 'package:quark/utils/file_kind.dart';
import 'package:quark/utils/markdown_viewer_files.dart';

/// Files with no extension that are still source, by their whole lowercase
/// name.
const _codeFileNames = {'dockerfile', 'makefile'};

/// Whether the file called [name] opens in the read-only code viewer
/// (#2991) and not the plaintext editor: source and config files, logs and
/// plain text.
///
/// The rule is one for every text file: it opens in a viewer, with Edit a tap
/// away (#3186). Markdown's viewer is its rendered one (#2987).
bool opensInCodeViewer(String name) {
  final base = name.substring(name.lastIndexOf('/') + 1).toLowerCase();
  return switch (fileKindForName(base)) {
    FileKind.code => true,
    FileKind.text => !opensInMarkdownViewer(base),
    _ => _codeFileNames.contains(base),
  };
}
