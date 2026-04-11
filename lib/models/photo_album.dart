class PhotoAlbum {
  final int id;
  final String name;
  final int? parentId;
  final DateTime createdAt;
  final DateTime updatedAt;
  final int itemCount;
  final List<PhotoAlbum> children;

  const PhotoAlbum({
    required this.id,
    required this.name,
    this.parentId,
    required this.createdAt,
    required this.updatedAt,
    required this.itemCount,
    this.children = const [],
  });

  factory PhotoAlbum.fromJson(Map<String, dynamic> json) {
    return PhotoAlbum(
      id: json['id'] as int,
      name: json['name'] as String,
      parentId: json['parentId'] as int?,
      createdAt: DateTime.parse(json['createdAt'] as String),
      updatedAt: DateTime.parse(json['updatedAt'] as String),
      itemCount: json['itemCount'] as int? ?? 0,
      children:
          (json['children'] as List<dynamic>?)
              ?.map((c) => PhotoAlbum.fromJson(c as Map<String, dynamic>))
              .toList() ??
          [],
    );
  }
}

class PhotoAlbumItem {
  final int id;
  final int albumId;
  final String deviceSerial;
  final String relPath;
  final DateTime addedAt;

  const PhotoAlbumItem({
    required this.id,
    required this.albumId,
    required this.deviceSerial,
    required this.relPath,
    required this.addedAt,
  });

  factory PhotoAlbumItem.fromJson(Map<String, dynamic> json) {
    return PhotoAlbumItem(
      id: json['id'] as int,
      albumId: json['albumId'] as int,
      deviceSerial: json['deviceSerial'] as String,
      relPath: json['relPath'] as String,
      addedAt: DateTime.parse(json['addedAt'] as String),
    );
  }
}
