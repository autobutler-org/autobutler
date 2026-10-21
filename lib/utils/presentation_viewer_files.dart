import 'package:quark/utils/file_kind.dart';

/// Whether the file called [name] opens in the read-only presentation viewer
/// (#3268): a `.pptx` or `.pptm`, read on the Quark.
///
/// A `.ppt` is a presentation too, but the legacy binary format has no
/// reader, so it stays on the generic viewer.
bool opensInPresentationViewer(String name) =>
    fileKindForName(name) == FileKind.slideshow &&
    fileExtension(name) != '.ppt';
