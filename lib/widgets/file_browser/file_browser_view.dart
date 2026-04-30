import 'package:autobutler/models/cirrus_file_node.dart';
import 'package:autobutler/services/cirrus_service.dart';
import 'package:autobutler/utils/file_browser_path_utils.dart';
import 'package:autobutler/utils/safe_set_state_mixin.dart';
import 'package:autobutler/widgets/core/autobutler_file_icon.dart';
import 'package:autobutler/widgets/core/empty_state_widget.dart';
import 'package:cached_network_image/cached_network_image.dart';
import 'package:desktop_drop/desktop_drop.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter/material.dart';
import 'package:shimmer/shimmer.dart';

enum FileMenuAction {
  download,
  moveRename,
  delete,
  navigateToFolder,
  extractHere,
}

enum SortColumn { name, type, size, device }

enum SortDirection { asc, desc }

class FileBrowserView extends StatefulWidget {
  const FileBrowserView({
    required this.filesFuture,
    required this.onFileMenuAction,
    required this.onOpenDirectory,
    required this.isGridView,
    required this.currentPath,
    this.initialData,
    this.isUnifiedView = true,
    this.onDropToFolder,
    this.onFolderDragEnter,
    this.onFolderDragExit,
    this.scrollController,
    this.showFileSizeAndMenu = true,
    this.isSearchMode = false,
    this.onNavigateToFolder,
    this.inArchive = false,
    this.isInitialLoad = false,
    this.errorBuilder,
    super.key,
  });

  final Future<List<CirrusFileNode>> filesFuture;
  final List<CirrusFileNode>? initialData;
  final Future<void> Function(CirrusFileNode, FileMenuAction) onFileMenuAction;
  final void Function(CirrusFileNode) onOpenDirectory;
  final bool isGridView;

  /// When true (default), files from all devices are shown merged.
  /// When false, files are grouped by device with a section header per device.
  final bool isUnifiedView;
  final String currentPath;
  final Future<void> Function(List<DropItem> droppedItems, String targetPath)?
  onDropToFolder;
  final VoidCallback? onFolderDragEnter;
  final VoidCallback? onFolderDragExit;
  final ScrollController? scrollController;
  final bool showFileSizeAndMenu;
  final bool isSearchMode;
  final void Function(CirrusFileNode)? onNavigateToFolder;

  /// When true, we are browsing inside an archive — only download is available
  /// for files (no move/rename/delete).
  final bool inArchive;

  /// When true, show a spinner unconditionally — used during the initial page
  /// load before any data has been fetched. Without this, the pre-resolved
  /// empty default future would immediately show "No files yet" instead of
  /// a spinner.
  final bool isInitialLoad;
  final Widget Function(BuildContext context, Object error)? errorBuilder;

  @override
  State<FileBrowserView> createState() => _FileBrowserViewState();
}

class _FileBrowserViewState extends State<FileBrowserView> {
  SortColumn _sortColumn = SortColumn.name;
  SortDirection _sortDirection = SortDirection.asc;
  final Set<String> _extractingPaths = {};

  void _toggleSort(SortColumn column) {
    setState(() {
      if (_sortColumn == column) {
        _sortDirection = _sortDirection == SortDirection.asc
            ? SortDirection.desc
            : SortDirection.asc;
      } else {
        _sortColumn = column;
        _sortDirection = SortDirection.asc;
      }
    });
  }

  List<CirrusFileNode> _sorted(List<CirrusFileNode> files) {
    final sorted = List<CirrusFileNode>.from(files);
    sorted.sort((a, b) {
      // Directories always first.
      final dirCmp = (b.isDir ? 1 : 0) - (a.isDir ? 1 : 0);
      if (dirCmp != 0) return dirCmp;

      int cmp;
      switch (_sortColumn) {
        case SortColumn.name:
          cmp = a.name.toLowerCase().compareTo(b.name.toLowerCase());
        case SortColumn.type:
          cmp = _fileType(a).compareTo(_fileType(b));
        case SortColumn.size:
          cmp = a.size.compareTo(b.size);
        case SortColumn.device:
          cmp = a.deviceName.toLowerCase().compareTo(
            b.deviceName.toLowerCase(),
          );
      }
      return _sortDirection == SortDirection.asc ? cmp : -cmp;
    });
    return sorted;
  }

  Widget _buildFolderDropWrapper({
    required CirrusFileNode item,
    required Widget child,
  }) {
    if (!kIsWeb || widget.onDropToFolder == null || !item.isDir) {
      return child;
    }
    return _FolderDropTarget(
      targetPath: normalizePath(joinPath(widget.currentPath, item.name)),
      onDropToFolder: widget.onDropToFolder!,
      onFolderDragEnter: widget.onFolderDragEnter,
      onFolderDragExit: widget.onFolderDragExit,
      child: child,
    );
  }

  void _dispatchMenuAction(
    BuildContext context,
    CirrusFileNode item,
    FileMenuAction action,
  ) {
    Future<void>.delayed(Duration.zero, () async {
      if (!context.mounted) return;
      if (action == FileMenuAction.extractHere) {
        if (_extractingPaths.contains(item.apiPath)) return;
        if (mounted) setState(() => _extractingPaths.add(item.apiPath));
        try {
          await widget.onFileMenuAction(item, action);
        } finally {
          if (mounted) setState(() => _extractingPaths.remove(item.apiPath));
        }
      } else {
        await widget.onFileMenuAction(item, action);
      }
    });
  }

  // ── Sort header ──────────────────────────────────────────────────────────

  Widget _buildSortHeader() {
    final colorScheme = Theme.of(context).colorScheme;
    return Container(
      color: colorScheme.secondary,
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
      child: Row(
        children: [
          // Leading icon placeholder
          const SizedBox(width: 40),
          _headerCell('Name', SortColumn.name, flex: 5),
          _headerCell('Device', SortColumn.device, flex: 2),
          if (widget.showFileSizeAndMenu)
            _headerCell('Size', SortColumn.size, flex: 2),
          // Trailing menu placeholder
          if (widget.showFileSizeAndMenu) const SizedBox(width: 48),
        ],
      ),
    );
  }

  Widget _buildGridSortHeader() {
    final colorScheme = Theme.of(context).colorScheme;
    return Container(
      color: colorScheme.secondary,
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 8),
      child: Row(
        children: [
          _headerCell('Name', SortColumn.name),
          _headerCell('Type', SortColumn.type),
          _headerCell('Size', SortColumn.size),
          _headerCell('Device', SortColumn.device),
        ],
      ),
    );
  }

  Widget _headerCell(String label, SortColumn column, {int flex = 1}) {
    final colorScheme = Theme.of(context).colorScheme;
    final isActive = _sortColumn == column;
    return Expanded(
      flex: flex,
      child: MouseRegion(
        cursor: SystemMouseCursors.click,
        child: GestureDetector(
          onTap: () => _toggleSort(column),
          child: Row(
            mainAxisSize: MainAxisSize.min,
            children: [
              Text(
                label,
                style: TextStyle(
                  fontSize: 12,
                  fontWeight: FontWeight.w500,
                  color: isActive
                      ? colorScheme.onSurface
                      : colorScheme.onSurfaceVariant,
                ),
              ),
              if (isActive) ...[
                const SizedBox(width: 4),
                Icon(
                  _sortDirection == SortDirection.asc
                      ? Icons.arrow_upward_rounded
                      : Icons.arrow_downward_rounded,
                  size: 12,
                  color: colorScheme.onSurface,
                ),
              ],
            ],
          ),
        ),
      ),
    );
  }

  // ── List tile ─────────────────────────────────────────────────────────────

  Widget _buildListTile(BuildContext context, CirrusFileNode item) {
    final colors = Theme.of(context).colorScheme;
    return Material(
      color: Colors.transparent,
      child: ListTile(
        contentPadding: const EdgeInsets.symmetric(horizontal: 16, vertical: 2),
        leading: _buildListLeading(item),
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
                style: TextStyle(color: colors.onSurfaceVariant),
              ),
            ),
            if (widget.showFileSizeAndMenu)
              Expanded(
                flex: 2,
                child: Text(
                  _formatSize(item.size, item.isDir),
                  maxLines: 1,
                  overflow: TextOverflow.ellipsis,
                  style: TextStyle(color: colors.onSurfaceVariant),
                ),
              ),
          ],
        ),
        trailing: widget.showFileSizeAndMenu
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
                    child: const Text('Download'),
                  ),
                  if (!widget.inArchive)
                    PopupMenuItem<FileMenuAction>(
                      value: FileMenuAction.moveRename,
                      onTap: () => _dispatchMenuAction(
                        context,
                        item,
                        FileMenuAction.moveRename,
                      ),
                      child: const Text('Move/Rename'),
                    ),
                  if (!widget.inArchive)
                    PopupMenuItem<FileMenuAction>(
                      value: FileMenuAction.delete,
                      onTap: () => _dispatchMenuAction(
                        context,
                        item,
                        FileMenuAction.delete,
                      ),
                      child: const Text('Delete'),
                    ),
                  if (!widget.inArchive && _isArchive(item))
                    PopupMenuItem<FileMenuAction>(
                      value: FileMenuAction.extractHere,
                      enabled: !_extractingPaths.contains(item.apiPath),
                      onTap: () => _dispatchMenuAction(
                        context,
                        item,
                        FileMenuAction.extractHere,
                      ),
                      child: _extractingPaths.contains(item.apiPath)
                          ? const Row(
                              children: [
                                SizedBox(
                                  width: 16,
                                  height: 16,
                                  child: CircularProgressIndicator(
                                    strokeWidth: 2,
                                  ),
                                ),
                                SizedBox(width: 8),
                                Text('Extracting...'),
                              ],
                            )
                          : const Text('Extract here'),
                    ),
                  if (widget.isSearchMode && widget.onNavigateToFolder != null)
                    PopupMenuItem<FileMenuAction>(
                      value: FileMenuAction.navigateToFolder,
                      onTap: () => widget.onNavigateToFolder!(item),
                      child: const Text('Navigate to folder'),
                    ),
                ],
              )
            : null,
        onTap: () => widget.onOpenDirectory(item),
      ),
    );
  }

  // ── Build ─────────────────────────────────────────────────────────────────

  @override
  Widget build(BuildContext context) {
    final colors = Theme.of(context).colorScheme;

    return FutureBuilder<List<CirrusFileNode>>(
      future: widget.filesFuture,
      initialData: widget.initialData,
      builder: (context, snapshot) {
        if (widget.isInitialLoad ||
            (snapshot.connectionState == ConnectionState.waiting &&
                !snapshot.hasData)) {
          return const Center(child: CircularProgressIndicator());
        }

        if (snapshot.hasError) {
          final error = snapshot.error!;
          if (widget.errorBuilder != null) {
            return widget.errorBuilder!(context, error);
          }
          return Center(
            child: Text(
              'Unable to load files',
              style: Theme.of(context).textTheme.bodyMedium,
            ),
          );
        }

        final raw = snapshot.data ?? const <CirrusFileNode>[];
        if (raw.isEmpty) {
          return const EmptyStateWidget(
            icon: Icons.folder_open_outlined,
            headline: 'No files yet',
            subtext:
                'Upload files using the button above, or drag and drop here.',
          );
        }

        final files = _sorted(raw);

        // ── Segmented view ────────────────────────────────────────────────
        if (!widget.isUnifiedView && !widget.isSearchMode) {
          final groups = <String, List<CirrusFileNode>>{};
          for (final f in files) {
            final key = f.deviceName.isNotEmpty
                ? f.deviceName
                : 'Unknown Device';
            groups.putIfAbsent(key, () => []).add(f);
          }
          return Column(
            children: [
              _buildSortHeader(),
              Expanded(
                child: ListView(
                  controller: widget.scrollController,
                  children: [
                    for (final entry in groups.entries)
                      ExpansionTile(
                        initiallyExpanded: true,
                        leading: const Icon(Icons.storage_rounded),
                        title: Text(entry.key),
                        subtitle: Text(
                          '${entry.value.length} '
                          'item${entry.value.length == 1 ? '' : 's'}',
                        ),
                        children: [
                          for (final item in entry.value)
                            _buildFolderDropWrapper(
                              item: item,
                              child: _buildListTile(context, item),
                            ),
                        ],
                      ),
                  ],
                ),
              ),
            ],
          );
        }

        // ── Grid view ─────────────────────────────────────────────────────
        if (widget.isGridView) {
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

          return Column(
            children: [
              _buildGridSortHeader(),
              Expanded(
                child: GridView.builder(
                  controller: widget.scrollController,
                  padding: const EdgeInsets.all(8),
                  itemCount: files.length,
                  gridDelegate: SliverGridDelegateWithFixedCrossAxisCount(
                    crossAxisCount: crossAxisCount,
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
                          onTap: () => widget.onOpenDirectory(item),
                          child: Padding(
                            padding: const EdgeInsets.all(8),
                            child: Column(
                              crossAxisAlignment: CrossAxisAlignment.start,
                              children: [
                                if (_isImageFile(item))
                                  SizedBox(
                                    height: 96,
                                    width: double.infinity,
                                    child: CachedNetworkImage(
                                      imageUrl:
                                          CirrusService.constructThumbnailUrl(
                                            item.apiPath,
                                            serial: item.deviceSerial,
                                          ).toString(),
                                      fit: BoxFit.cover,
                                      placeholder: (context, url) =>
                                          Shimmer.fromColors(
                                            baseColor: Colors.grey[800]!,
                                            highlightColor: Colors.grey[700]!,
                                            child: Container(
                                              color: Colors.grey[800],
                                            ),
                                          ),
                                      errorWidget: (context, url, error) =>
                                          Center(
                                            child: AutobutlerFileIcon(
                                              node: item,
                                              size: 48,
                                            ),
                                          ),
                                    ),
                                  )
                                else
                                  Center(
                                    child: AutobutlerFileIcon(
                                      node: item,
                                      size: 48,
                                    ),
                                  ),
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
                                  mainAxisAlignment:
                                      MainAxisAlignment.spaceBetween,
                                  children: [
                                    if (widget.showFileSizeAndMenu)
                                      Flexible(
                                        child: Text(
                                          _formatSize(item.size, item.isDir),
                                          maxLines: 1,
                                          overflow: TextOverflow.ellipsis,
                                        ),
                                      ),
                                    if (widget.showFileSizeAndMenu)
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
                                            child: const Text('Download'),
                                          ),
                                          if (!widget.inArchive)
                                            PopupMenuItem<FileMenuAction>(
                                              value: FileMenuAction.moveRename,
                                              onTap: () => _dispatchMenuAction(
                                                context,
                                                item,
                                                FileMenuAction.moveRename,
                                              ),
                                              child: const Text('Move/Rename'),
                                            ),
                                          if (!widget.inArchive)
                                            PopupMenuItem<FileMenuAction>(
                                              value: FileMenuAction.delete,
                                              onTap: () => _dispatchMenuAction(
                                                context,
                                                item,
                                                FileMenuAction.delete,
                                              ),
                                              child: const Text('Delete'),
                                            ),
                                          if (!widget.inArchive &&
                                              _isArchive(item))
                                            PopupMenuItem<FileMenuAction>(
                                              value: FileMenuAction.extractHere,
                                              enabled: !_extractingPaths
                                                  .contains(item.apiPath),
                                              onTap: () => _dispatchMenuAction(
                                                context,
                                                item,
                                                FileMenuAction.extractHere,
                                              ),
                                              child:
                                                  _extractingPaths.contains(
                                                    item.apiPath,
                                                  )
                                                  ? const Row(
                                                      children: [
                                                        SizedBox(
                                                          width: 16,
                                                          height: 16,
                                                          child:
                                                              CircularProgressIndicator(
                                                                strokeWidth: 2,
                                                              ),
                                                        ),
                                                        SizedBox(width: 8),
                                                        Text('Extracting...'),
                                                      ],
                                                    )
                                                  : const Text('Extract here'),
                                            ),
                                          if (widget.isSearchMode &&
                                              widget.onNavigateToFolder != null)
                                            PopupMenuItem<FileMenuAction>(
                                              value: FileMenuAction
                                                  .navigateToFolder,
                                              onTap: () =>
                                                  widget.onNavigateToFolder!(
                                                    item,
                                                  ),
                                              child: const Text(
                                                'Navigate to folder',
                                              ),
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
                ),
              ),
            ],
          );
        }

        // ── List view ─────────────────────────────────────────────────────
        return Column(
          children: [
            _buildSortHeader(),
            Expanded(
              child: ListView.separated(
                controller: widget.scrollController,
                itemCount: files.length,
                separatorBuilder: (_, _) => Divider(
                  height: 1,
                  color: colors.outlineVariant.withValues(alpha: 0.5),
                ),
                itemBuilder: (context, index) {
                  final item = files[index];
                  return _buildFolderDropWrapper(
                    item: item,
                    child: _buildListTile(context, item),
                  );
                },
              ),
            ),
          ],
        );
      },
    );
  }

  static bool _isImageFile(CirrusFileNode node) {
    if (node.isDir) return false;
    final lower = node.name.toLowerCase();
    return lower.endsWith('.jpg') ||
        lower.endsWith('.jpeg') ||
        lower.endsWith('.png') ||
        lower.endsWith('.gif') ||
        lower.endsWith('.webp') ||
        lower.endsWith('.heic') ||
        lower.endsWith('.heif');
  }

  Widget _buildListLeading(CirrusFileNode item) {
    if (!_isImageFile(item)) {
      return AutobutlerFileIcon(node: item);
    }

    return ClipRRect(
      borderRadius: BorderRadius.circular(4),
      child: SizedBox(
        width: 40,
        height: 40,
        child: CachedNetworkImage(
          imageUrl: CirrusService.constructThumbnailUrl(
            item.apiPath,
            serial: item.deviceSerial,
            size: 'sm',
          ).toString(),
          fit: BoxFit.cover,
          placeholder: (context, url) => Shimmer.fromColors(
            baseColor: Colors.grey[800]!,
            highlightColor: Colors.grey[700]!,
            child: Container(color: Colors.grey[800]),
          ),
          errorWidget: (context, url, error) => AutobutlerFileIcon(node: item),
        ),
      ),
    );
  }

  static bool _isArchive(CirrusFileNode node) {
    if (node.isDir) return false;
    return node.fileType == 'archive';
  }

  static String _fileType(CirrusFileNode node) {
    if (node.isDir) return '';
    final dot = node.name.lastIndexOf('.');
    if (dot < 0) return 'file';
    return node.name.substring(dot + 1).toLowerCase();
  }

  static String _formatSize(int bytes, bool isDir) {
    if (isDir) return '--';
    if (bytes < 1024) return '$bytes B';
    if (bytes < 1024 * 1024) return '${(bytes / 1024).toStringAsFixed(1)} KB';
    if (bytes < 1024 * 1024 * 1024) {
      return '${(bytes / (1024 * 1024)).toStringAsFixed(1)} MB';
    }
    return '${(bytes / (1024 * 1024 * 1024)).toStringAsFixed(1)} GB';
  }
}

// ── Folder drop target ────────────────────────────────────────────────────────

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

class _FolderDropTargetState extends State<_FolderDropTarget>
    with SafeSetStateMixin {
  bool _isDragOver = false;

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;
    return DropTarget(
      onDragEntered: (_) {
        if (!mounted) return;
        setStateSafely(() => _isDragOver = true);
        widget.onFolderDragEnter?.call();
      },
      onDragExited: (_) {
        if (!mounted) return;
        setStateSafely(() => _isDragOver = false);
        widget.onFolderDragExit?.call();
      },
      onDragDone: (details) async {
        if (mounted) setStateSafely(() => _isDragOver = false);
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
