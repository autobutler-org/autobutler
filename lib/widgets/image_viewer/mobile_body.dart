import 'package:flutter/material.dart';
import 'package:quark/models/photo_metadata.dart';
import 'package:quark/widgets/image_viewer/metadata_drawer.dart';

/// The photo viewer at phone widths: the photo, with the metadata drawer
/// dragged up over it.
class MobileBody extends StatelessWidget {
  final Widget photoArea;
  final bool sidebarOpen;
  final DraggableScrollableController drawerController;
  final String name;
  final PhotoMetadata? metadata;
  final bool loading;
  final void Function(AlbumRef album) onAlbumTap;

  const MobileBody({
    super.key,
    required this.photoArea,
    required this.sidebarOpen,
    required this.drawerController,
    required this.name,
    required this.metadata,
    required this.loading,
    required this.onAlbumTap,
  });

  @override
  Widget build(BuildContext context) {
    return Stack(
      children: [
        photoArea,
        if (sidebarOpen)
          DraggableScrollableSheet(
            controller: drawerController,
            initialChildSize: 0.28,
            minChildSize: 0.08,
            maxChildSize: 0.85,
            snap: true,
            snapSizes: const [0.08, 0.28, 0.85],
            builder: (context, scrollController) => MetadataDrawer(
              scrollController: scrollController,
              name: name,
              metadata: metadata,
              loading: loading,
              onAlbumTap: onAlbumTap,
            ),
          ),
      ],
    );
  }
}
