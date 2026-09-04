import 'package:flutter/material.dart';
import 'package:quark/models/photo_album.dart';
import 'package:quark/services/album_service.dart';
import 'package:quark_widgets/quark_widgets.dart';
import 'package:quark_icons/quark_icons.dart';
import 'package:quark/utils/error_text.dart';

class AddToAlbumSheet extends StatefulWidget {
  const AddToAlbumSheet({
    required this.deviceSerial,
    required this.relPath,
    super.key,
  });

  final String deviceSerial;
  final String relPath;

  static Future<void> show(
    BuildContext context, {
    required String deviceSerial,
    required String relPath,
  }) {
    return showModalBottomSheet<void>(
      context: context,
      isScrollControlled: true,
      shape: RoundedRectangleBorder(
        borderRadius: BorderRadius.circular(QuarkColors.radiusLg),
      ),
      builder: (_) =>
          AddToAlbumSheet(deviceSerial: deviceSerial, relPath: relPath),
    );
  }

  @override
  State<AddToAlbumSheet> createState() => _AddToAlbumSheetState();
}

class _AddToAlbumSheetState extends State<AddToAlbumSheet> {
  List<PhotoAlbum> _albums = [];
  final Set<int> _inAlbums = {};
  bool _loading = true;

  @override
  void initState() {
    super.initState();
    _load();
  }

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

  Future<void> _toggle(PhotoAlbum album) async {
    final inAlbum = _inAlbums.contains(album.id);
    try {
      if (inAlbum) {
        await AlbumService.removePhotoFromAlbum(
          album.id,
          deviceSerial: widget.deviceSerial,
          relPath: widget.relPath,
        );
        setState(() => _inAlbums.remove(album.id));
      } else {
        await AlbumService.addPhotoToAlbum(
          album.id,
          deviceSerial: widget.deviceSerial,
          relPath: widget.relPath,
        );
        setState(() => _inAlbums.add(album.id));
      }
    } catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(context).showSnackBar(
        SnackBar(content: Text(Errors.message(e, 'update the album'))),
      );
    }
  }

  @override
  Widget build(BuildContext context) {
    return DraggableScrollableSheet(
      expand: false,
      initialChildSize: 0.5,
      minChildSize: 0.3,
      maxChildSize: 0.85,
      builder: (ctx, scrollController) => Column(
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
          const Text(
            'Add to album',
            style: TextStyle(fontSize: 16, fontWeight: FontWeight.w600),
          ),
          const SizedBox(height: 8),
          Expanded(
            child: _loading
                ? const Center(child: CircularProgressIndicator())
                : _albums.isEmpty
                ? const Center(child: Text('No albums — create one first'))
                : ListView(
                    controller: scrollController,
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
      final inAlbum = _inAlbums.contains(album.id);
      widgets.add(
        ListTile(
          contentPadding: EdgeInsets.only(left: 16.0 + depth * 16.0, right: 16),
          leading: Icon(
            inAlbum
                ? QuarkIcons.check_circle_rounded
                : QuarkIcons.photo_album_outlined,
            color: inAlbum
                ? Theme.of(context).colorScheme.primary
                : Theme.of(context).colorScheme.onSurfaceVariant,
          ),
          title: Text(album.name),
          subtitle: Text('${album.itemCount} photos'),
          onTap: () => _toggle(album),
        ),
      );
      if (album.children.isNotEmpty) {
        widgets.addAll(_buildAlbumList(album.children, depth + 1));
      }
    }
    return widgets;
  }
}
