/// One saved version of a file, as a `FileVersionsPanel` lists it.
///
/// The caller formats what a person reads: [sizeLabel] and [author] arrive as
/// text, so the package needs neither a byte formatter nor a user lookup.
class FileVersionItem {
  /// Creates the entry for the version [id].
  const FileVersionItem({
    required this.id,
    required this.createdAt,
    required this.sizeLabel,
    this.author = '',
    this.label = '',
    this.isNamed = false,
  });

  /// What the panel's callbacks hand back, and the suffix of the row's keys.
  /// Unique within one file's history.
  final String id;

  /// When the version was saved.
  final DateTime createdAt;

  /// The version's size, already formatted: `12.4 KB`.
  final String sizeLabel;

  /// Who saved it, empty when nobody could be named.
  final String author;

  /// The name the version was given, empty when it has none.
  final String label;

  /// Whether somebody saved this version on purpose, rather than the Quark
  /// saving it automatically before a change.
  final bool isNamed;
}
