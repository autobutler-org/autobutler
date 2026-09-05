import 'package:flutter/material.dart';
import 'package:quark/models/photo_album.dart';
import 'package:quark/widgets/photos/album_sidebar/album_tile.dart';
import 'package:quark_widgets/quark_widgets.dart';

/// The album list, expanding to fill the parent or shrink-wrapping to its
/// own height depending on whether the parent bounds it (#1599).
class AlbumList extends StatelessWidget {
  final List<PhotoAlbum> albums;
  final bool shrinkWrap;
  final int? selectedAlbumId;
  final Set<int> expandedIds;
  final ValueChanged<AlbumItem> onSelected;
  final ValueChanged<int> onToggleExpanded;
  final ValueChanged<AlbumItem> onLongPress;

  const AlbumList({
    super.key,
    required this.albums,
    required this.shrinkWrap,
    required this.selectedAlbumId,
    required this.expandedIds,
    required this.onSelected,
    required this.onToggleExpanded,
    required this.onLongPress,
  });

  @override
  Widget build(BuildContext context) {
    final list = ListView.builder(
      padding: EdgeInsets.zero,
      shrinkWrap: shrinkWrap,
      // Already inside an outer scroll view when shrink-wrapped; a nested
      // scrollable on the same axis would fight it for drag gestures.
      physics: shrinkWrap ? const NeverScrollableScrollPhysics() : null,
      itemCount: albums.length,
      itemBuilder: (context, index) => AlbumTile(
        album: albums[index],
        selectedAlbumId: selectedAlbumId,
        expandedIds: expandedIds,
        onSelected: onSelected,
        onToggleExpanded: onToggleExpanded,
        onLongPress: onLongPress,
      ),
    );
    return shrinkWrap ? list : Expanded(child: list);
  }
}
