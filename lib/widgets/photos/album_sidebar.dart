import 'package:flutter/material.dart';
import 'package:quark/models/photo_album.dart';
import 'package:quark/services/album_service.dart';
import 'package:quark/theme/quark_colors.dart';
import 'package:quark/widgets/photos/album_tree_tile.dart';
import 'package:quark_icons/quark_icons.dart';

class AlbumSidebar extends StatefulWidget {
  const AlbumSidebar({
    required this.selectedAlbumId,
    required this.onAlbumSelected,
    super.key,
  });

  final int? selectedAlbumId;
  final void Function(PhotoAlbum? album) onAlbumSelected;

  @override
  State<AlbumSidebar> createState() => AlbumSidebarState();
}

class AlbumSidebarState extends State<AlbumSidebar> {
  List<PhotoAlbum> _albums = [];
  bool _loading = true;

  @override
  void initState() {
    super.initState();
    _load();
  }

  /// Reload the album list from the server. Call this after external changes
  /// (e.g. a photo was added to an album from the image viewer).
  Future<void> reload() => _load();

  Future<void> _load() async {
    try {
      final albums = await AlbumService.listAlbums(tree: true);
      if (!mounted) return;
      setState(() {
        _albums = albums;
        _loading = false;
      });
    } catch (_) {
      if (!mounted) return;
      setState(() => _loading = false);
    }
  }

  Future<void> _createAlbum({int? parentId}) async {
    final name = await _promptAlbumName(context, title: 'New album');
    if (name == null || name.isEmpty) return;
    try {
      await AlbumService.createAlbum(name, parentId: parentId);
      await _load();
    } catch (_) {
      if (!mounted) return;
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(const SnackBar(content: Text('Failed to create album')));
    }
  }

  Future<void> _renameAlbum(PhotoAlbum album) async {
    final name = await _promptAlbumName(
      context,
      title: 'Rename album',
      initial: album.name,
    );
    if (name == null || name.isEmpty || name == album.name) return;
    try {
      await AlbumService.renameAlbum(album.id, name);
      await _load();
    } catch (_) {
      if (!mounted) return;
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(const SnackBar(content: Text('Failed to rename album')));
    }
  }

  Future<void> _deleteAlbum(PhotoAlbum album) async {
    final confirmed = await showDialog<bool>(
      context: context,
      builder: (ctx) => AlertDialog(
        title: const Text('Delete album?'),
        content: Text(
          'Delete "${album.name}"? Photos will not be deleted from disk. '
          'Sub-albums will also be deleted.',
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(ctx).pop(false),
            child: const Text('Cancel'),
          ),
          FilledButton(
            onPressed: () => Navigator.of(ctx).pop(true),
            child: const Text('Delete'),
          ),
        ],
      ),
    );
    if (confirmed != true) return;
    try {
      await AlbumService.deleteAlbum(album.id);
      if (widget.selectedAlbumId == album.id) {
        widget.onAlbumSelected(null);
      }
      await _load();
    } catch (_) {
      if (!mounted) return;
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(const SnackBar(content: Text('Failed to delete album')));
    }
  }

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;

    final systemAlbums =
        (_albums.where((a) => a.isSystemAlbum).toList()..sort((a, b) {
              if (a.isFavorites) return -1;
              if (b.isFavorites) return 1;
              return 0;
            }))
            .toList();
    final userAlbums = _albums.where((a) => !a.isSystemAlbum).toList();
    final allAlbums = [...systemAlbums, ...userAlbums];

    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        // ── Header ──────────────────────────────────────────────────────────
        Padding(
          padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
          child: Row(
            children: [
              Text(
                'Albums',
                style: TextStyle(
                  fontSize: 11,
                  fontWeight: FontWeight.w600,
                  color: colorScheme.onSurface.withValues(alpha: 0.5),
                  letterSpacing: 0.8,
                ),
              ),
              const Spacer(),
              IconButton(
                icon: const Icon(QuarkIcons.add_rounded, size: 16),
                padding: EdgeInsets.zero,
                constraints: const BoxConstraints(minWidth: 24, minHeight: 24),
                tooltip: 'New album',
                onPressed: _createAlbum,
              ),
            ],
          ),
        ),
        // ── Album list (scrollable) ──────────────────────────────────────────
        if (_loading)
          const Padding(
            padding: EdgeInsets.symmetric(horizontal: 12),
            child: LinearProgressIndicator(),
          )
        else if (_albums.isEmpty)
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
            child: Text(
              'No albums yet',
              style: TextStyle(
                fontSize: 13,
                color: colorScheme.onSurface.withValues(alpha: 0.4),
              ),
            ),
          )
        else
          Expanded(
            child: ListView.builder(
              padding: EdgeInsets.zero,
              itemCount: allAlbums.length,
              itemBuilder: (context, index) =>
                  _buildAlbumTile(context, allAlbums[index]),
            ),
          ),
        // ── Footer divider ───────────────────────────────────────────────────
        const SizedBox(height: 8),
        Divider(height: 1, color: colorScheme.outline.withValues(alpha: 0.5)),
      ],
    );
  }

  Widget _buildAlbumTile(BuildContext context, PhotoAlbum album) {
    if (album.isSystemAlbum) {
      // System albums: no long-press menu, distinct icon
      return AlbumTreeTile(
        album: album,
        selectedAlbumId: widget.selectedAlbumId,
        onSelected: widget.onAlbumSelected,
        systemIcon: album.isFavorites
            ? QuarkIcons.star_rounded
            : QuarkIcons.pending_actions_outlined,
      );
    }
    return GestureDetector(
      onLongPress: () => _showAlbumContextMenu(context, album),
      child: AlbumTreeTile(
        album: album,
        selectedAlbumId: widget.selectedAlbumId,
        onSelected: widget.onAlbumSelected,
      ),
    );
  }

  void _showAlbumContextMenu(BuildContext context, PhotoAlbum album) {
    showModalBottomSheet<void>(
      context: context,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(QuarkColors.radiusLg),
      ),
      builder: (ctx) => SafeArea(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            ListTile(
              leading: const Icon(QuarkIcons.edit_outlined),
              title: const Text('Rename'),
              onTap: () {
                Navigator.of(ctx).pop();
                _renameAlbum(album);
              },
            ),
            ListTile(
              leading: const Icon(QuarkIcons.create_new_folder_outlined),
              title: const Text('New sub-album'),
              onTap: () {
                Navigator.of(ctx).pop();
                _createAlbum(parentId: album.id);
              },
            ),
            ListTile(
              leading: Icon(
                QuarkIcons.delete_outline,
                color: Theme.of(ctx).colorScheme.error,
              ),
              title: Text(
                'Delete',
                style: TextStyle(color: Theme.of(ctx).colorScheme.error),
              ),
              onTap: () {
                Navigator.of(ctx).pop();
                _deleteAlbum(album);
              },
            ),
          ],
        ),
      ),
    );
  }
}

Future<String?> _promptAlbumName(
  BuildContext context, {
  required String title,
  String initial = '',
}) async {
  final controller = TextEditingController(text: initial);
  return showDialog<String>(
    context: context,
    builder: (ctx) => AlertDialog(
      title: Text(title),
      content: TextField(
        controller: controller,
        autofocus: true,
        decoration: const InputDecoration(hintText: 'Album name'),
        textInputAction: TextInputAction.done,
        onSubmitted: (_) => Navigator.of(ctx).pop(controller.text.trim()),
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.of(ctx).pop(),
          child: const Text('Cancel'),
        ),
        FilledButton(
          onPressed: () => Navigator.of(ctx).pop(controller.text.trim()),
          child: const Text('Save'),
        ),
      ],
    ),
  );
}
