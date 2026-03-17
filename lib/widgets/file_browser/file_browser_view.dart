import 'package:autobutler/models/cirrus_file_node.dart';
import 'package:autobutler/services/cirrus_service.dart';
import 'package:autobutler/utils/file_browser_path_utils.dart';
import 'package:desktop_drop/desktop_drop.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';

enum FileMenuAction { download, moveRename, delete }

class FileBrowserView extends StatelessWidget {
  const FileBrowserView({
    required this.filesFuture,
    required this.onFileMenuAction,
    required this.onOpenDirectory,
    required this.isGridView,
    required this.currentPath,
    this.onDropToFolder,
    this.onFolderDragEnter,
    this.onFolderDragExit,
    this.showFileSizeAndMenu = true,
    this.isSearchMode = false,
    this.onNavigateToFolder,
    super.key,
  });

  final Future<List<CirrusFileNode>> filesFuture;
  final Future<void> Function(CirrusFileNode, FileMenuAction) onFileMenuAction;
  final void Function(CirrusFileNode) onOpenDirectory;
  final bool isGridView;
  final String currentPath;
  final Future<void> Function(List<DropItem> droppedItems, String targetPath)?
  onDropToFolder;
  final VoidCallback? onFolderDragEnter;
  final VoidCallback? onFolderDragExit;
  final bool showFileSizeAndMenu;
  final bool isSearchMode;
  final void Function(CirrusFileNode)? onNavigateToFolder;

  Widget _buildFolderDropWrapper({
    required CirrusFileNode item,
    required Widget child,
  }) {
    if (!kIsWeb || onDropToFolder == null || !item.isDir) {
      return child;
    }

    return _FolderDropTarget(
      targetPath: normalizePath(joinPath(currentPath, item.name)),
      onDropToFolder: onDropToFolder!,
      onFolderDragEnter: onFolderDragEnter,
      onFolderDragExit: onFolderDragExit,
      child: child,
    );
  }

  void _dispatchMenuAction(
    BuildContext context,
    CirrusFileNode item,
    FileMenuAction action,
  ) {
    Future<void>.delayed(Duration.zero, () async {
      // Check context is still valid before proceeding
      if (!context.mounted) {
        return;
      }
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
          final screenWidth = MediaQuery.sizeOf(context).width;
          const horizontalPadding = 16.0;
          const crossAxisSpacing = 8.0;
          const minTileWidth = 180.0;
          final usableWidth = screenWidth - horizontalPadding;
          final calculatedCount =
              ((usableWidth + crossAxisSpacing) /
                      (minTileWidth + crossAxisSpacing))
                  .floor();
          final crossAxisCount = calculatedCount < 1 ? 1 : calculatedCount;

          return GridView.builder(
            padding: const EdgeInsets.all(8),
            itemCount: files.length,
            gridDelegate: SliverGridDelegateWithFixedCrossAxisCount(
              // Number of tiles per row
              crossAxisCount: crossAxisCount,
              // How tall each tile is relative to its width (1.0 = square)
              childAspectRatio: 1.1,
              crossAxisSpacing: crossAxisSpacing,
              mainAxisSpacing: 8,
            ),
            itemBuilder: (context, index) {
              final item = files[index];
              return _buildFolderDropWrapper(
                item: item,
                child: Card(
                  clipBehavior: Clip.hardEdge,
                  child: InkWell(
                    onTap: () => onOpenDirectory(item),
                    child: Padding(
                      padding: const EdgeInsets.all(8),
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          // If the node is an image, show a thumbnail from the backend thumbnails endpoint;
                          // otherwise show a representative icon.
                          (() {
                            final lower = item.name.toLowerCase();
                            final isImage =
                                lower.endsWith('.jpg') ||
                                lower.endsWith('.jpeg') ||
                                lower.endsWith('.png') ||
                                lower.endsWith('.gif') ||
                                lower.endsWith('.webp');
                            if (isImage) {
                              final url = CirrusService.constructThumbnailUrl(
                                item.apiPath,
                                serial: item.deviceSerial,
                              );
                              return SizedBox(
                                height: 96,
                                width: double.infinity,
                                child: Image.network(
                                  url.toString(),
                                  fit: BoxFit.cover,
                                  loadingBuilder: (context, child, progress) {
                                    if (progress == null) return child;
                                    return Container(color: Colors.grey[300]);
                                  },
                                  errorBuilder: (context, error, stack) =>
                                      Container(color: Colors.grey[300]),
                                ),
                              );
                            }
                            return Center(
                              child: Icon(_iconForNode(item), size: 48),
                            );
                          })(),
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
                              if (showFileSizeAndMenu)
                                Flexible(
                                  child: Text(
                                    _formatSize(item.size, item.isDir),
                                    maxLines: 1,
                                    overflow: TextOverflow.ellipsis,
                                  ),
                                ),
                              if (showFileSizeAndMenu)
                                PopupMenuButton<FileMenuAction>(
                                  icon: const Icon(Icons.more_vert),
                                  itemBuilder: (context) => [
                                    PopupMenuItem<FileMenuAction>(
                                      value: FileMenuAction.download,
                                      onTap: () => _dispatchMenuAction(
                                        context,
                                        item,
                                        FileMenuAction.download,
                                      ),
                                      child: Text('Download'),
                                    ),
                                    PopupMenuItem<FileMenuAction>(
                                      value: FileMenuAction.moveRename,
                                      onTap: () => _dispatchMenuAction(
                                        context,
                                        item,
                                        FileMenuAction.moveRename,
                                      ),
                                      child: Text('Move/Rename'),
                                    ),
                                    PopupMenuItem<FileMenuAction>(
                                      value: FileMenuAction.delete,
                                      onTap: () => _dispatchMenuAction(
                                        context,
                                        item,
                                        FileMenuAction.delete,
                                      ),
                                      child: Text('Delete'),
                                    ),
                                    if (isSearchMode &&
                                        onNavigateToFolder != null)
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
            return _buildFolderDropWrapper(
              item: item,
              child: Material(
                color: Colors.transparent,
                child: ListTile(
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
                      if (showFileSizeAndMenu)
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
                  trailing: showFileSizeAndMenu
                      ? PopupMenuButton<FileMenuAction>(
                          icon: const Icon(Icons.more_vert),
                          itemBuilder: (context) => [
                            PopupMenuItem<FileMenuAction>(
                              value: FileMenuAction.download,
                              onTap: () => _dispatchMenuAction(
                                context,
                                item,
                                FileMenuAction.download,
                              ),
                              child: Text('Download'),
                            ),
                            PopupMenuItem<FileMenuAction>(
                              value: FileMenuAction.moveRename,
                              onTap: () => _dispatchMenuAction(
                                context,
                                item,
                                FileMenuAction.moveRename,
                              ),
                              child: Text('Move/Rename'),
                            ),
                            PopupMenuItem<FileMenuAction>(
                              value: FileMenuAction.delete,
                              onTap: () => _dispatchMenuAction(
                                context,
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
                        )
                      : null,
                  onTap: () => onOpenDirectory(item),
                ),
              ),
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

class _FolderDropTarget extends StatefulWidget {
  const _FolderDropTarget({
    required this.targetPath,
    required this.onDropToFolder,
    this.onFolderDragEnter,
    this.onFolderDragExit,
    required this.child,
  });

  final String targetPath;
  final Future<void> Function(List<DropItem> droppedItems, String targetPath)
  onDropToFolder;
  final VoidCallback? onFolderDragEnter;
  final VoidCallback? onFolderDragExit;
  final Widget child;

  @override
  State<_FolderDropTarget> createState() => _FolderDropTargetState();
}

class _FolderDropTargetState extends State<_FolderDropTarget> {
  bool _isDragOver = false;

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;
    return DropTarget(
      onDragEntered: (_) {
        if (!mounted) {
          return;
        }
        setState(() {
          _isDragOver = true;
        });
        widget.onFolderDragEnter?.call();
      },
      onDragExited: (_) {
        if (!mounted) {
          return;
        }
        setState(() {
          _isDragOver = false;
        });
        widget.onFolderDragExit?.call();
      },
      onDragDone: (details) async {
        if (mounted) {
          setState(() {
            _isDragOver = false;
          });
        }
        widget.onFolderDragExit?.call();
        await widget.onDropToFolder(details.files, widget.targetPath);
      },
      child: AnimatedContainer(
        duration: const Duration(milliseconds: 120),
        decoration: BoxDecoration(
          borderRadius: BorderRadius.circular(8),
          border: Border.all(
            color: _isDragOver ? colorScheme.primary : Colors.transparent,
            width: 1.5,
          ),
          color: _isDragOver
              ? colorScheme.primaryContainer.withValues(alpha: 0.35)
              : null,
        ),
        child: widget.child,
      ),
    );
  }
}
