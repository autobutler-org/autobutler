import 'package:quark/utils/file_kind.dart';

/// Files with no extension that are still source, by their whole lowercase
/// name.
const _codeFileNames = {'dockerfile', 'makefile'};

/// Whether the file called [name] opens in the read-only code viewer
/// (#2991) and not the plaintext editor: source and config files, and logs.
///
/// A log is [FileKind.text], but it is read, not written, and it is the file
/// most likely to be too large to load whole. Plain prose (`.txt`) keeps
/// opening in the editor; Markdown has its own viewer (#2987).
bool opensInCodeViewer(String name) {
  final base = name.substring(name.lastIndexOf('/') + 1).toLowerCase();
  return fileKindForName(base) == FileKind.code ||
      fileExtension(base) == '.log' ||
      _codeFileNames.contains(base);
}
