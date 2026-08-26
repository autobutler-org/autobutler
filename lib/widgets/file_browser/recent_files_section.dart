import 'package:quark/models/file_node.dart';
import 'package:quark/services/files_service.dart';
import 'package:quark/theme/quark_colors.dart';
import 'package:quark/utils/files_route_path_utils.dart';
import 'package:quark/widgets/core/quark_file_icon.dart';
import 'package:quark/widgets/file_browser/file_browser_view.dart';
import 'package:flutter/material.dart';
import 'package:quark_icons/quark_icons.dart';

/// A horizontally-scrolling strip showing recently uploaded files.
/// Displayed at the root of the file browser (not in search mode).
///
/// Tapping a file chip calls [onOpenFile] for viewable types (images/video/audio),
/// or [onFileMenuAction] with [FileMenuAction.download] for everything else.
/// Tapping the folder badge triggers [onNavigateToFolder] with the parent directory path.
class RecentFilesSection extends StatefulWidget {
  const RecentFilesSection({
    required this.onOpenFile,
    required this.onFileMenuAction,
    required this.onNavigateToFolder,
    super.key,
  });

  final void Function(FileNode) onOpenFile;
  final Future<void> Function(FileNode, FileMenuAction) onFileMenuAction;
  final void Function(String path) onNavigateToFolder;

  @override
  State<RecentFilesSection> createState() => _RecentFilesSectionState();
}

class _RecentFilesSectionState extends State<RecentFilesSection> {
  late Future<List<FileNode>> _future;

  @override
  void initState() {
    super.initState();
    _future = FilesService.getRecentFiles(limit: 20);
  }

  String _parentPath(FileNode node) {
    final path = node.apiPath;
    final slash = path.lastIndexOf('/');
    if (slash <= 0) return '';
    return path.substring(0, slash);
  }

  /// Returns true for any file type that the file browser can route into an editor or
  /// conversion flow rather than downloading directly.
  static bool _opensInApp(String name) {
    final lower = name.toLowerCase();
    return hasSupportedFilesEditorForPath(lower) || lower.endsWith('.csv');
  }

  void _openOrDownload(FileNode file) {
    if (_opensInApp(file.name)) {
      widget.onOpenFile(file);
    } else {
      widget.onFileMenuAction(file, FileMenuAction.download);
    }
  }

  @override
  Widget build(BuildContext context) {
    return FutureBuilder<List<FileNode>>(
      future: _future,
      builder: (context, snapshot) {
        // Don't show section at all while loading or on error or if empty.
        if (!snapshot.hasData || snapshot.hasError) {
          return const SizedBox.shrink();
        }
        final files = snapshot.data!.where((f) => !f.isDir).toList();
        if (files.isEmpty) return const SizedBox.shrink();

        final colorScheme = Theme.of(context).colorScheme;
        return Container(
          decoration: BoxDecoration(
            border: Border(bottom: BorderSide(color: colorScheme.outline)),
          ),
          padding: const EdgeInsets.fromLTRB(16, 10, 16, 12),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            mainAxisSize: MainAxisSize.min,
            children: [
              Row(
                children: [
                  Icon(
                    QuarkIcons.schedule_rounded,
                    size: 14,
                    color: colorScheme.onSurface.withValues(alpha: 0.4),
                  ),
                  const SizedBox(width: 6),
                  Text(
                    'Recently uploaded',
                    style: TextStyle(
                      fontSize: 12,
                      fontWeight: FontWeight.w500,
                      color: colorScheme.onSurface.withValues(alpha: 0.4),
                    ),
                  ),
                ],
              ),
              const SizedBox(height: 8),
              SizedBox(
                height: 72,
                child: ListView.separated(
                  scrollDirection: Axis.horizontal,
                  itemCount: files.length,
                  separatorBuilder: (_, _) => const SizedBox(width: 8),
                  itemBuilder: (context, index) {
                    final file = files[index];
                    return _RecentFileChip(
                      file: file,
                      onTap: () => _openOrDownload(file),
                      onFolderTap: () =>
                          widget.onNavigateToFolder(_parentPath(file)),
                    );
                  },
                ),
              ),
            ],
          ),
        );
      },
    );
  }
}

class _RecentFileChip extends StatelessWidget {
  const _RecentFileChip({
    required this.file,
    required this.onTap,
    required this.onFolderTap,
  });

  final FileNode file;
  final VoidCallback onTap;
  final VoidCallback onFolderTap;

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;
    return MouseRegion(
      cursor: SystemMouseCursors.click,
      child: Material(
        color: colorScheme.surfaceContainerHighest,
        shape: RoundedRectangleBorder(
          side: BorderSide(color: colorScheme.outline),
          borderRadius: BorderRadius.circular(QuarkColors.radiusMd),
        ),
        clipBehavior: Clip.antiAlias,
        child: InkWell(
          onTap: onTap,
          child: Padding(
            padding: const EdgeInsets.fromLTRB(10, 8, 6, 8),
            child: Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                QuarkFileIcon(
                  node: file,
                  size: 20,
                  color: colorScheme.onSurfaceVariant,
                ),
                const SizedBox(width: 8),
                ConstrainedBox(
                  constraints: const BoxConstraints(maxWidth: 140),
                  child: Column(
                    mainAxisSize: MainAxisSize.min,
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        file.name,
                        maxLines: 1,
                        overflow: TextOverflow.ellipsis,
                        style: TextStyle(
                          fontSize: 13,
                          color: colorScheme.onSurface,
                        ),
                      ),
                      if (file.deviceName.isNotEmpty)
                        Text(
                          file.deviceName,
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                          style: TextStyle(
                            fontSize: 11,
                            color: colorScheme.onSurface.withValues(alpha: 0.4),
                          ),
                        ),
                    ],
                  ),
                ),
                const SizedBox(width: 6),
                // Folder navigate badge
                MouseRegion(
                  cursor: SystemMouseCursors.click,
                  child: Tooltip(
                    message: 'Go to folder',
                    child: InkWell(
                      onTap: onFolderTap,
                      borderRadius: BorderRadius.circular(4),
                      child: Padding(
                        padding: const EdgeInsets.all(4),
                        child: Icon(
                          QuarkIcons.folder_open_rounded,
                          size: 14,
                          color: colorScheme.onSurface.withValues(alpha: 0.4),
                        ),
                      ),
                    ),
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
