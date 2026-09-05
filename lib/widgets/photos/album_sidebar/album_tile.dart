import 'package:flutter/material.dart';
import 'package:quark/models/photo_album.dart';
import 'package:quark_icons/quark_icons.dart';
import 'package:quark_widgets/quark_widgets.dart';

/// One row of the album sidebar, mapping an app [PhotoAlbum] onto the
/// package's tree tile.
class AlbumTile extends StatelessWidget {
  final PhotoAlbum album;
  final int? selectedAlbumId;
  final Set<int> expandedIds;
  final ValueChanged<AlbumItem> onSelected;
  final ValueChanged<int> onToggleExpanded;
  final ValueChanged<AlbumItem> onLongPress;

  const AlbumTile({
    super.key,
    required this.album,
    required this.selectedAlbumId,
    required this.expandedIds,
    required this.onSelected,
    required this.onToggleExpanded,
    required this.onLongPress,
  });

  /// The package's view of [album], with its subtree mapped too.
  static AlbumItem _toItem(PhotoAlbum album) => AlbumItem(
    id: album.id,
    name: album.name,
    parentId: album.parentId,
    itemCount: album.itemCount,
    isSystem: album.isSystemAlbum,
    isFavorites: album.isFavorites,
    children: album.children.map(_toItem).toList(),
  );

  @override
  Widget build(BuildContext context) {
    return AlbumTreeTile(
      album: _toItem(album),
      selectedAlbumId: selectedAlbumId,
      expandedIds: expandedIds,
      onSelected: onSelected,
      onToggleExpanded: onToggleExpanded,
      // System albums are the Quark's, not the user's, so they get no rename
      // or delete menu.
      onLongPress: album.isSystemAlbum ? null : onLongPress,
      systemIcon: album.isSystemAlbum
          ? (album.isFavorites
                ? QuarkIcons.star_rounded
                : QuarkIcons.pending_actions_outlined)
          : null,
    );
  }
}
