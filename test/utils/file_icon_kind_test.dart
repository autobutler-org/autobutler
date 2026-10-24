import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:quark/utils/file_kind.dart';
import 'package:quark_icons/quark_icons.dart';
import 'package:quark_widgets/quark_widgets.dart';

/// #3730: a file's icon follows its [FileKind], so it can no longer show the
/// generic glyph for a file that opens in a real viewer.
void main() {
  IconData glyphFor(FileKind kind) => switch (kind) {
    FileKind.qdoc => QuarkIcons.edit_document,
    FileKind.qsheet => QuarkIcons.table_chart,
    FileKind.qslide => QuarkIcons.slideshow,
    FileKind.image || FileKind.svg => QuarkIcons.image_outlined,
    FileKind.video => QuarkIcons.video_file_outlined,
    FileKind.audio => QuarkIcons.audio_file_outlined,
    FileKind.pdf => QuarkIcons.picture_as_pdf_outlined,
    FileKind.docx => QuarkIcons.description_outlined,
    FileKind.xlsx || FileKind.csv => QuarkIcons.table_chart_outlined,
    FileKind.slideshow => QuarkIcons.slideshow_outlined,
    FileKind.archive => QuarkIcons.archive_outlined,
    FileKind.text ||
    FileKind.code ||
    FileKind.epub ||
    FileKind.generic => QuarkIcons.insert_drive_file_outlined,
  };

  test('every known extension gets the glyph for its kind', () {
    for (final ext in knownFileExtensions) {
      final name = 'file$ext';
      expect(
        QuarkFileIcon.iconFor(fileIconKindForName(name)),
        glyphFor(fileKindForName(name)),
        reason: ext,
      );
    }
  });

  test('a folder is a folder whatever its name', () {
    expect(
      fileIconKindForName('backup.zip', isDir: true),
      QuarkFileKind.folder,
    );
  });
}
