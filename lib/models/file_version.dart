/// One saved version of a file, as `/api/v0/versions` describes it.
class FileVersion {
  const FileVersion({
    required this.id,
    required this.createdAt,
    required this.size,
    this.sha256 = '',
    this.isNamed = false,
    this.label = '',
    this.authorId = 0,
  });

  /// What the Quark calls the version; only meaningful beside the file's path.
  final String id;

  /// When the version was saved.
  final DateTime createdAt;

  /// The version's size in bytes.
  final int size;

  /// The hex SHA-256 of the version's content.
  final String sha256;

  /// Whether the version's kind is `named`, saved on purpose, rather than
  /// `auto`, saved by the Quark before a change.
  final bool isNamed;

  /// The version's name, empty when it has none.
  final String label;

  /// The account that saved the version, 0 when the Quark did.
  final int authorId;

  factory FileVersion.fromJson(Map<String, dynamic> json) => FileVersion(
    id: json['id'] as String? ?? '',
    createdAt:
        DateTime.tryParse(json['createdAt'] as String? ?? '') ??
        DateTime.fromMillisecondsSinceEpoch(0, isUtc: true),
    size: (json['size'] as num?)?.toInt() ?? 0,
    sha256: json['sha256'] as String? ?? '',
    isNamed: json['kind'] == 'named',
    label: json['label'] as String? ?? '',
    authorId: (json['authorId'] as num?)?.toInt() ?? 0,
  );
}
