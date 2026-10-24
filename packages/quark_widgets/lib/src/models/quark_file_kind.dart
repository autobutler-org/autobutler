/// Which glyph `QuarkFileIcon` draws for an entry. The caller decides what a
/// file is and passes the answer in; the package keeps no extension table of
/// its own (#3730).
enum QuarkFileKind {
  /// A folder.
  folder,

  /// A photo or other picture.
  image,

  /// A video.
  video,

  /// A song or other recording.
  audio,

  /// A PDF.
  pdf,

  /// A compressed archive.
  archive,

  /// A Quark document.
  qdoc,

  /// A Quark spreadsheet.
  qsheet,

  /// A Quark presentation.
  qslide,

  /// A word-processor document from another app.
  document,

  /// A spreadsheet from another app.
  spreadsheet,

  /// A presentation from another app.
  presentation,

  /// Anything else.
  generic,
}
