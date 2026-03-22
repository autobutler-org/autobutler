import 'package:autobutler/models/cirrus_file_node.dart';
import 'package:autobutler/theme/autobutler_colors.dart';
import 'package:autobutler/widgets/file_browser/file_browser_view.dart';
import 'package:flutter/material.dart';

enum SortColumn { name, type, size, modified }

enum SortDirection { asc, desc }

class FileListTable extends StatefulWidget {
  const FileListTable({
    required this.files,
    required this.onOpenDirectory,
    required this.onFileMenuAction,
    this.onNavigateToFolder,
    this.isSearchMode = false,
    this.scrollController,
    super.key,
  });

  final List<CirrusFileNode> files;
  final void Function(CirrusFileNode) onOpenDirectory;
  final Future<void> Function(CirrusFileNode, FileMenuAction) onFileMenuAction;
  final void Function(CirrusFileNode)? onNavigateToFolder;
  final bool isSearchMode;
  final ScrollController? scrollController;

  @override
  State<FileListTable> createState() => _FileListTableState();
}

class _FileListTableState extends State<FileListTable> {
  SortColumn _sortColumn = SortColumn.name;
  SortDirection _sortDirection = SortDirection.asc;

  List<CirrusFileNode> get _sortedFiles {
    final sorted = List<CirrusFileNode>.from(widget.files);
    sorted.sort((a, b) {
      final dirCompare = (b.isDir ? 1 : 0) - (a.isDir ? 1 : 0);
      if (dirCompare != 0) return dirCompare;

      int compare;
      switch (_sortColumn) {
        case SortColumn.name:
          compare = a.name.toLowerCase().compareTo(b.name.toLowerCase());
        case SortColumn.type:
          compare = _fileType(a).compareTo(_fileType(b));
        case SortColumn.size:
          compare = a.size.compareTo(b.size);
        case SortColumn.modified:
          compare = a.name.compareTo(b.name);
      }
      return _sortDirection == SortDirection.asc ? compare : -compare;
    });
    return sorted;
  }

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

  @override
  Widget build(BuildContext context) {
    final files = _sortedFiles;
    return Container(
      decoration: BoxDecoration(
        color: AutobutlerColors.card,
        border: Border.all(color: AutobutlerColors.border),
        borderRadius: BorderRadius.circular(AutobutlerColors.radiusLg),
      ),
      clipBehavior: Clip.antiAlias,
      child: Column(
        children: [
          _buildHeader(),
          Expanded(
            child: ListView.builder(
              controller: widget.scrollController,
              itemCount: files.length,
              itemBuilder: (context, index) =>
                  _buildRow(context, files[index], index == files.length - 1),
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildHeader() {
    return Container(
      color: AutobutlerColors.sidebar,
      padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
      child: Row(
        children: [
          const SizedBox(width: 32),
          _headerCell('Name', SortColumn.name, flex: 4),
          _headerCell('Type', SortColumn.type, flex: 2),
          _headerCell('Size', SortColumn.size, flex: 2),
          _headerCell('Device', SortColumn.modified, flex: 2),
          const SizedBox(width: 80),
        ],
      ),
    );
  }

  Widget _headerCell(String label, SortColumn column, {int flex = 1}) {
    final isActive = _sortColumn == column;
    return Expanded(
      flex: flex,
      child: GestureDetector(
        onTap: () => _toggleSort(column),
        child: Row(
          children: [
            Text(
              label,
              style: TextStyle(
                fontSize: 12,
                fontWeight: FontWeight.w500,
                color: isActive
                    ? AutobutlerColors.foreground
                    : AutobutlerColors.secondaryForeground,
              ),
            ),
            if (isActive) ...[
              const SizedBox(width: 4),
              Icon(
                _sortDirection == SortDirection.asc
                    ? Icons.arrow_upward_rounded
                    : Icons.arrow_downward_rounded,
                size: 12,
                color: AutobutlerColors.foreground,
              ),
            ],
          ],
        ),
      ),
    );
  }

  Widget _buildRow(BuildContext context, CirrusFileNode file, bool isLast) {
    return Container(
      decoration: BoxDecoration(
        border: isLast
            ? null
            : const Border(
                bottom: BorderSide(color: AutobutlerColors.border, width: 0.5),
              ),
      ),
      child: Material(
        color: Colors.transparent,
        child: InkWell(
          onTap: () => widget.onOpenDirectory(file),
          hoverColor: AutobutlerColors.sidebar.withValues(alpha: 0.5),
          child: Padding(
            padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
            child: Row(
              children: [
                SizedBox(
                  width: 32,
                  child: Icon(
                    _iconForNode(file),
                    size: 18,
                    color: file.isDir
                        ? AutobutlerColors.primary
                        : AutobutlerColors.secondaryForeground,
                  ),
                ),
                Expanded(
                  flex: 4,
                  child: Text(
                    file.name,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: const TextStyle(
                      fontSize: 14,
                      color: AutobutlerColors.cardForeground,
                    ),
                  ),
                ),
                Expanded(
                  flex: 2,
                  child: Text(
                    file.isDir ? 'Folder' : _fileType(file),
                    style: const TextStyle(
                      fontSize: 14,
                      color: AutobutlerColors.secondaryForeground,
                    ),
                  ),
                ),
                Expanded(
                  flex: 2,
                  child: Text(
                    file.isDir ? '—' : _formatSize(file.size),
                    style: const TextStyle(
                      fontSize: 14,
                      color: AutobutlerColors.secondaryForeground,
                    ),
                  ),
                ),
                Expanded(
                  flex: 2,
                  child: Text(
                    file.deviceName,
                    maxLines: 1,
                    overflow: TextOverflow.ellipsis,
                    style: const TextStyle(
                      fontSize: 14,
                      color: AutobutlerColors.secondaryForeground,
                    ),
                  ),
                ),
                SizedBox(
                  width: 80,
                  child: Row(
                    mainAxisAlignment: MainAxisAlignment.end,
                    children: [
                      if (file.isDir)
                        _actionBadge(
                          context,
                          icon: Icons.arrow_forward_rounded,
                          label: 'Open',
                          onTap: () => widget.onOpenDirectory(file),
                        )
                      else
                        _actionBadge(
                          context,
                          icon: Icons.more_horiz_rounded,
                          onTap: () => _showMenu(context, file),
                        ),
                    ],
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }

  Widget _actionBadge(
    BuildContext context, {
    required IconData icon,
    String? label,
    required VoidCallback onTap,
  }) {
    return GestureDetector(
      onTap: onTap,
      child: Container(
        padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
        decoration: BoxDecoration(
          color: AutobutlerColors.input,
          border: Border.all(color: AutobutlerColors.border),
          borderRadius: BorderRadius.circular(AutobutlerColors.radiusLg),
        ),
        child: Row(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(icon, size: 12, color: AutobutlerColors.secondaryForeground),
            if (label != null) ...[
              const SizedBox(width: 4),
              Text(
                label,
                style: const TextStyle(
                  fontSize: 12,
                  color: AutobutlerColors.secondaryForeground,
                ),
              ),
            ],
          ],
        ),
      ),
    );
  }

  void _showMenu(BuildContext context, CirrusFileNode file) {
    showMenu<FileMenuAction>(
      context: context,
      position: RelativeRect.fill,
      items: [
        const PopupMenuItem(
          value: FileMenuAction.download,
          child: Text('Download'),
        ),
        const PopupMenuItem(
          value: FileMenuAction.moveRename,
          child: Text('Move/Rename'),
        ),
        const PopupMenuItem(
          value: FileMenuAction.delete,
          child: Text('Delete'),
        ),
        if (widget.isSearchMode && widget.onNavigateToFolder != null)
          const PopupMenuItem(
            value: FileMenuAction.navigateToFolder,
            child: Text('Navigate to folder'),
          ),
      ],
    ).then((action) {
      if (action != null) {
        if (action == FileMenuAction.navigateToFolder &&
            widget.onNavigateToFolder != null) {
          widget.onNavigateToFolder!(file);
        } else {
          widget.onFileMenuAction(file, action);
        }
      }
    });
  }

  static String _fileType(CirrusFileNode node) {
    if (node.isDir) return 'Folder';
    final dot = node.name.lastIndexOf('.');
    if (dot < 0) return 'File';
    return node.name.substring(dot + 1).toUpperCase();
  }

  static String _formatSize(int bytes) {
    if (bytes < 1024) return '$bytes B';
    if (bytes < 1024 * 1024) return '${(bytes / 1024).toStringAsFixed(1)} KB';
    if (bytes < 1024 * 1024 * 1024) {
      return '${(bytes / (1024 * 1024)).toStringAsFixed(1)} MB';
    }
    return '${(bytes / (1024 * 1024 * 1024)).toStringAsFixed(1)} GB';
  }

  static IconData _iconForNode(CirrusFileNode node) {
    if (node.isDir) return Icons.folder_outlined;
    final lower = node.name.toLowerCase();
    if (lower.endsWith('.jpg') ||
        lower.endsWith('.jpeg') ||
        lower.endsWith('.png') ||
        lower.endsWith('.gif') ||
        lower.endsWith('.webp')) {
      return Icons.image_outlined;
    }
    if (lower.endsWith('.zip') ||
        lower.endsWith('.tar') ||
        lower.endsWith('.gz') ||
        lower.endsWith('.7z')) {
      return Icons.archive_outlined;
    }
    if (lower.endsWith('.pdf')) return Icons.picture_as_pdf_outlined;
    if (lower.endsWith('.mp4') ||
        lower.endsWith('.mov') ||
        lower.endsWith('.mkv') ||
        lower.endsWith('.webm')) {
      return Icons.video_file_outlined;
    }
    if (lower.endsWith('.mp3') ||
        lower.endsWith('.wav') ||
        lower.endsWith('.flac')) {
      return Icons.audio_file_outlined;
    }
    if (lower.endsWith('.md') ||
        lower.endsWith('.txt') ||
        lower.endsWith('.doc') ||
        lower.endsWith('.docx')) {
      return Icons.description_outlined;
    }
    if (lower.endsWith('.js') ||
        lower.endsWith('.ts') ||
        lower.endsWith('.dart') ||
        lower.endsWith('.go') ||
        lower.endsWith('.py')) {
      return Icons.code_outlined;
    }
    return Icons.insert_drive_file_outlined;
  }
}
