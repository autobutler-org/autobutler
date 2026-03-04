import 'package:autobutler/models/cirrus_file_node.dart';
import 'package:flutter/material.dart';

enum FileMenuAction { download, moveRename, delete }

class FileBrowserView extends StatelessWidget {
  const FileBrowserView({
    required this.filesFuture,
    required this.onFileMenuAction,
    required this.onOpenDirectory,
    required this.isGridView,
    this.isSearchMode = false,
    this.onNavigateToFolder,
    super.key,
  });

  final Future<List<CirrusFileNode>> filesFuture;
  final Future<void> Function(CirrusFileNode, FileMenuAction) onFileMenuAction;
  final void Function(CirrusFileNode) onOpenDirectory;
  final bool isGridView;
  final bool isSearchMode;
  final void Function(CirrusFileNode)? onNavigateToFolder;

  void _dispatchMenuAction(CirrusFileNode item, FileMenuAction action) {
    Future<void>.delayed(Duration.zero, () async {
      await onFileMenuAction(item, action);
    });
  }

  @override
  Widget build(BuildContext context) {
    final colors = Theme.of(context).colorScheme;

    return FutureBuilder<List<CirrusFileNode>>(
      future: filesFuture,
      builder: (context, snapshot) {
        if (snapshot.connectionState == ConnectionState.waiting) {
          return const Center(child: CircularProgressIndicator());
        }

        if (snapshot.hasError) {
          return Center(
            child: Text(
              'Unable to load files',
              style: Theme.of(context).textTheme.bodyMedium,
            ),
          );
        }

        final files = snapshot.data ?? const <CirrusFileNode>[];
        if (files.isEmpty) {
          return const Center(child: Text('No files found'));
        }

        if (isGridView) {
          return GridView.builder(
            padding: const EdgeInsets.all(8),
            itemCount: files.length,
            gridDelegate: const SliverGridDelegateWithFixedCrossAxisCount(
              // Number of tiles per row
              crossAxisCount: 2,
              // How tall each tile is relative to its width (1.0 = square)
              childAspectRatio: 1.1,
              crossAxisSpacing: 8,
              mainAxisSpacing: 8,
            ),
            itemBuilder: (context, index) {
              final item = files[index];
              return Card(
                clipBehavior: Clip.hardEdge,
                child: InkWell(
                  onTap: () => onOpenDirectory(item),
                  child: Padding(
                    padding: const EdgeInsets.all(8),
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Center(child: Icon(_iconForNode(item), size: 48)),
                        const SizedBox(height: 8),
                        Flexible(
                          child: Text(
                            item.name,
                            maxLines: 1,
                            overflow: TextOverflow.ellipsis,
                          ),
                        ),
                        const Spacer(),
                        Row(
                          mainAxisAlignment: MainAxisAlignment.spaceBetween,
                          children: [
                            Flexible(
                              child: Text(
                                _formatSize(item.size, item.isDir),
                                maxLines: 1,
                                overflow: TextOverflow.ellipsis,
                              ),
                            ),
                            PopupMenuButton<FileMenuAction>(
                              icon: const Icon(Icons.more_vert),
                              itemBuilder: (context) => [
                                PopupMenuItem<FileMenuAction>(
                                  value: FileMenuAction.download,
                                  onTap: () => _dispatchMenuAction(
                                    item,
                                    FileMenuAction.download,
                                  ),
                                  child: Text('Download'),
                                ),
                                PopupMenuItem<FileMenuAction>(
                                  value: FileMenuAction.moveRename,
                                  onTap: () => _dispatchMenuAction(
                                    item,
                                    FileMenuAction.moveRename,
                                  ),
                                  child: Text('Move/Rename'),
                                ),
                                PopupMenuItem<FileMenuAction>(
                                  value: FileMenuAction.delete,
                                  onTap: () => _dispatchMenuAction(
                                    item,
                                    FileMenuAction.delete,
                                  ),
                                  child: Text('Delete'),
                                ),
                                if (isSearchMode && onNavigateToFolder != null)
                                  PopupMenuItem<FileMenuAction>(
                                    // Reuse an existing enum value; action is handled via onNavigateToFolder
                                    value: FileMenuAction.download,
                                    onTap: () => onNavigateToFolder!(item),
                                    child: Text('Navigate to folder'),
                                  ),
                              ],
                            ),
                          ],
                        ),
                      ],
                    ),
                  ),
                ),
              );
            },
          );
        }

        return ListView.separated(
          itemCount: files.length,
          separatorBuilder: (_, _) => Divider(
            height: 1,
            color: colors.outlineVariant.withValues(alpha: 0.5),
          ),
          itemBuilder: (context, index) {
            final item = files[index];
            return ListTile(
              contentPadding: const EdgeInsets.symmetric(
                horizontal: 16,
                vertical: 2,
              ),
              leading: Icon(_iconForNode(item)),
              title: Row(
                children: [
                  Expanded(
                    flex: 5,
                    child: Text(
                      item.name,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                    ),
                  ),
                  Expanded(
                    flex: 2,
                    child: Text(
                      item.deviceName,
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                    ),
                  ),
                  Expanded(
                    flex: 2,
                    child: Text(
                      _formatSize(item.size, item.isDir),
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                    ),
                  ),
                ],
              ),
              trailing: PopupMenuButton<FileMenuAction>(
                icon: const Icon(Icons.more_vert),
                itemBuilder: (context) => [
                  PopupMenuItem<FileMenuAction>(
                    value: FileMenuAction.download,
                    onTap: () =>
                        _dispatchMenuAction(item, FileMenuAction.download),
                    child: Text('Download'),
                  ),
                  PopupMenuItem<FileMenuAction>(
                    value: FileMenuAction.moveRename,
                    onTap: () =>
                        _dispatchMenuAction(item, FileMenuAction.moveRename),
                    child: Text('Move/Rename'),
                  ),
                  PopupMenuItem<FileMenuAction>(
                    value: FileMenuAction.delete,
                    onTap: () =>
                        _dispatchMenuAction(item, FileMenuAction.delete),
                    child: Text('Delete'),
                  ),
                  if (isSearchMode && onNavigateToFolder != null)
                    PopupMenuItem<FileMenuAction>(
                      // Reuse an existing enum value; action is handled via onNavigateToFolder
                      value: FileMenuAction.download,
                      onTap: () => onNavigateToFolder!(item),
                      child: Text('Navigate to folder'),
                    ),
                ],
              ),
              onTap: () => onOpenDirectory(item),
            );
          },
        );
      },
    );
  }

  static IconData _iconForNode(CirrusFileNode node) {
    if (node.isDir) {
      return Icons.folder_outlined;
    }

    final lowerName = node.name.toLowerCase();
    if (lowerName.endsWith('.jpg') ||
        lowerName.endsWith('.jpeg') ||
        lowerName.endsWith('.png') ||
        lowerName.endsWith('.gif') ||
        lowerName.endsWith('.webp')) {
      return Icons.image_outlined;
    }

    if (lowerName.endsWith('.zip') ||
        lowerName.endsWith('.tar') ||
        lowerName.endsWith('.gz') ||
        lowerName.endsWith('.7z')) {
      return Icons.archive_outlined;
    }

    return Icons.insert_drive_file_outlined;
  }

  static String _formatSize(int bytes, bool isDir) {
    if (isDir) {
      return '--';
    }

    if (bytes < 1024) {
      return '$bytes B';
    }
    if (bytes < 1024 * 1024) {
      return '${(bytes / 1024).toStringAsFixed(1)} KB';
    }
    if (bytes < 1024 * 1024 * 1024) {
      return '${(bytes / (1024 * 1024)).toStringAsFixed(1)} MB';
    }
    return '${(bytes / (1024 * 1024 * 1024)).toStringAsFixed(1)} GB';
  }
}
