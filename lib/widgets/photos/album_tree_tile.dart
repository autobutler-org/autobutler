import 'package:autobutler/models/photo_album.dart';
import 'package:autobutler/theme/autobutler_colors.dart';
import 'package:flutter/material.dart';
import 'package:autobutler_icons/autobutler_icons.dart';

typedef AlbumSelectedCallback = void Function(PhotoAlbum? album);

class AlbumTreeTile extends StatefulWidget {
  const AlbumTreeTile({
    required this.album,
    required this.selectedAlbumId,
    required this.onSelected,
    this.depth = 0,
    this.systemIcon,
    super.key,
  });

  final PhotoAlbum album;
  final int? selectedAlbumId;
  final AlbumSelectedCallback onSelected;
  final int depth;

  /// Override icon for system albums (e.g. star for Favorites).
  final IconData? systemIcon;

  @override
  State<AlbumTreeTile> createState() => _AlbumTreeTileState();
}

class _AlbumTreeTileState extends State<AlbumTreeTile> {
  bool _expanded = false;

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;
    final isSelected = widget.selectedAlbumId == widget.album.id;
    final hasChildren = widget.album.children.isNotEmpty;
    final indent = widget.depth * 16.0;

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      mainAxisSize: MainAxisSize.min,
      children: [
        InkWell(
          onTap: () => widget.onSelected(widget.album),
          borderRadius: BorderRadius.circular(AutobutlerColors.radiusMd),
          child: Container(
            decoration: BoxDecoration(
              color: isSelected
                  ? colorScheme.primary.withValues(alpha: 0.12)
                  : Colors.transparent,
              borderRadius: BorderRadius.circular(AutobutlerColors.radiusMd),
            ),
            padding: EdgeInsets.only(
              left: 8 + indent,
              right: 8,
              top: 6,
              bottom: 6,
            ),
            child: Row(
              children: [
                if (hasChildren)
                  GestureDetector(
                    onTap: () => setState(() => _expanded = !_expanded),
                    child: Icon(
                      _expanded
                          ? AutobutlerIcons.expand_more_rounded
                          : AutobutlerIcons.chevron_right_rounded,
                      size: 16,
                      color: colorScheme.onSurfaceVariant,
                    ),
                  )
                else
                  const SizedBox(width: 16),
                const SizedBox(width: 4),
                Icon(
                  widget.systemIcon ?? AutobutlerIcons.photo_album_outlined,
                  size: 16,
                  color: isSelected
                      ? colorScheme.primary
                      : colorScheme.onSurfaceVariant,
                ),
                const SizedBox(width: 8),
                Expanded(
                  child: Text(
                    widget.album.name,
                    style: TextStyle(
                      fontSize: 13,
                      fontWeight: isSelected
                          ? FontWeight.w600
                          : FontWeight.normal,
                      color: isSelected
                          ? colorScheme.primary
                          : colorScheme.onSurface,
                    ),
                    overflow: TextOverflow.ellipsis,
                  ),
                ),
                if (widget.album.itemCount > 0)
                  Text(
                    '${widget.album.itemCount}',
                    style: TextStyle(
                      fontSize: 11,
                      color: colorScheme.onSurface.withValues(alpha: 0.4),
                    ),
                  ),
              ],
            ),
          ),
        ),
        if (_expanded && hasChildren)
          ...widget.album.children.map(
            (child) => AlbumTreeTile(
              album: child,
              selectedAlbumId: widget.selectedAlbumId,
              onSelected: widget.onSelected,
              depth: widget.depth + 1,
            ),
          ),
      ],
    );
  }
}
