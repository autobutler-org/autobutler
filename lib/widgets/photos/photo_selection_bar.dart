import 'package:quark/models/photo_album.dart';
import 'package:quark/services/album_service.dart';
import 'package:quark/theme/quark_colors.dart';
import 'package:flutter/material.dart';
import 'package:quark_icons/quark_icons.dart';

/// Sticky bottom bar shown during photo selection mode.
/// Shows count + "Add to Album" button.
class PhotoSelectionBar extends StatelessWidget {
  const PhotoSelectionBar({
    required this.selectedCount,
    required this.onAddToAlbum,
    required this.onCancel,
    super.key,
  });

  final int selectedCount;
  final VoidCallback onAddToAlbum;
  final VoidCallback onCancel;

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;
    return SafeArea(
      top: false,
      child: Container(
        decoration: BoxDecoration(
          color: colorScheme.surface,
          border: Border(top: BorderSide(color: colorScheme.outline)),
          boxShadow: [
            BoxShadow(
              color: Colors.black.withValues(alpha: 0.2),
              blurRadius: 8,
              offset: const Offset(0, -2),
            ),
          ],
        ),
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
        child: Row(
          children: [
            TextButton(onPressed: onCancel, child: const Text('Cancel')),
            const Spacer(),
            Text(
              '$selectedCount ${selectedCount == 1 ? 'photo' : 'photos'} selected',
              style: TextStyle(
                fontSize: 13,
                color: colorScheme.onSurface.withValues(alpha: 0.6),
              ),
            ),
            const Spacer(),
            FilledButton.icon(
              onPressed: selectedCount > 0 ? onAddToAlbum : null,
              icon: const Icon(QuarkIcons.photo_album_outlined, size: 16),
              label: const Text('Add to Album'),
            ),
          ],
        ),
      ),
    );
  }
}

/// Bottom sheet for picking an album to add selected photos to.
/// Shows album list; on tap calls [onAlbumPicked].
class AlbumPickerSheet extends StatefulWidget {
  const AlbumPickerSheet({required this.selectedCount, super.key});

  final int selectedCount;

  static Future<PhotoAlbum?> show(
    BuildContext context, {
    required int selectedCount,
  }) {
    return showModalBottomSheet<PhotoAlbum>(
      context: context,
      isScrollControlled: true,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(QuarkColors.radiusLg),
      ),
      builder: (_) => AlbumPickerSheet(selectedCount: selectedCount),
    );
  }

  @override
  State<AlbumPickerSheet> createState() => _AlbumPickerSheetState();
}

class _AlbumPickerSheetState extends State<AlbumPickerSheet> {
  List<PhotoAlbum> _albums = [];
  bool _loading = true;
  bool _error = false;

  @override
  void initState() {
    super.initState();
    _load();
  }

  Future<void> _load() async {
    setState(() {
      _loading = true;
      _error = false;
    });
    try {
      final albums = await AlbumService.listAlbums(tree: true);
      if (!mounted) return;
      setState(() {
        _albums = albums;
        _loading = false;
      });
    } catch (_) {
      if (!mounted) return;
      setState(() {
        _loading = false;
        _error = true;
      });
    }
  }

  @override
  Widget build(BuildContext context) {
    return DraggableScrollableSheet(
      expand: false,
      initialChildSize: 0.5,
      minChildSize: 0.3,
      maxChildSize: 0.85,
      builder: (ctx, sc) => Column(
        children: [
          const SizedBox(height: 8),
          Container(
            width: 40,
            height: 4,
            decoration: BoxDecoration(
              color: Theme.of(context).colorScheme.outline,
              borderRadius: BorderRadius.circular(2),
            ),
          ),
          const SizedBox(height: 12),
          Text(
            'Add ${widget.selectedCount} ${widget.selectedCount == 1 ? 'photo' : 'photos'} to...',
            style: const TextStyle(fontSize: 16, fontWeight: FontWeight.w600),
          ),
          const SizedBox(height: 4),
          Expanded(
            child: _loading
                ? const Center(child: CircularProgressIndicator())
                : _error
                ? Center(
                    child: Column(
                      mainAxisSize: MainAxisSize.min,
                      children: [
                        const Text('Failed to load albums'),
                        const SizedBox(height: 8),
                        TextButton(
                          onPressed: _load,
                          child: const Text('Retry'),
                        ),
                      ],
                    ),
                  )
                : _albums.isEmpty
                ? const Center(
                    child: Text('No albums — create one in the Photos view'),
                  )
                : ListView(
                    controller: sc,
                    children: _buildAlbumList(_albums, 0),
                  ),
          ),
          const SizedBox(height: 8),
        ],
      ),
    );
  }

  List<Widget> _buildAlbumList(List<PhotoAlbum> albums, int depth) {
    final widgets = <Widget>[];
    for (final album in albums) {
      widgets.add(
        ListTile(
          contentPadding: EdgeInsets.only(left: 16.0 + depth * 16.0, right: 16),
          leading: const Icon(QuarkIcons.photo_album_outlined),
          title: Text(album.name),
          subtitle: Text('${album.itemCount} photos'),
          onTap: () => Navigator.of(context).pop(album),
        ),
      );
      if (album.children.isNotEmpty) {
        widgets.addAll(_buildAlbumList(album.children, depth + 1));
      }
    }
    return widgets;
  }
}
