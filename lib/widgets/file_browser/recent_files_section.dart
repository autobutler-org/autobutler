import 'package:autobutler/models/cirrus_file_node.dart';
import 'package:autobutler/services/cirrus_service.dart';
import 'package:autobutler/theme/autobutler_colors.dart';
import 'package:autobutler/widgets/core/autobutler_file_icon.dart';
import 'package:autobutler/widgets/file_browser/file_browser_view.dart';
import 'package:flutter/material.dart';
import 'package:autobutler/pages/document_editor_page.dart';

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

  final void Function(CirrusFileNode) onOpenFile;
  final Future<void> Function(CirrusFileNode, FileMenuAction) onFileMenuAction;
  final void Function(String path) onNavigateToFolder;

  @override
  State<RecentFilesSection> createState() => _RecentFilesSectionState();
}

class _RecentFilesSectionState extends State<RecentFilesSection> {
  late Future<List<CirrusFileNode>> _future;

  @override
  void initState() {
    super.initState();
    _future = CirrusService.getRecentFiles(limit: 20);
  }

  String _parentPath(CirrusFileNode node) {
    final path = node.apiPath;
    final slash = path.lastIndexOf('/');
    if (slash <= 0) return '';
    return path.substring(0, slash);
  }

  static bool _isViewable(String name) {
    final lower = name.toLowerCase();
    return lower.endsWith('.jpg') ||
        lower.endsWith('.jpeg') ||
        lower.endsWith('.png') ||
        lower.endsWith('.gif') ||
        lower.endsWith('.webp') ||
        lower.endsWith('.mp4') ||
        lower.endsWith('.mov') ||
        lower.endsWith('.mkv') ||
        lower.endsWith('.webm') ||
        lower.endsWith('.avi') ||
        lower.endsWith('.mp3') ||
        lower.endsWith('.wav') ||
        lower.endsWith('.m4a') ||
        lower.endsWith('.aac');
  }

  void _openOrDownload(CirrusFileNode file) {
    if (_isViewable(file.name)) {
      widget.onOpenFile(file);
    } else {
      widget.onFileMenuAction(file, FileMenuAction.download);
    }
  }

  @override
  Widget build(BuildContext context) {
    return FutureBuilder<List<CirrusFileNode>>(
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
                    Icons.schedule_rounded,
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
                      onTap: () {
                        if (file.name.endsWith('.abdoc')) {
                          Navigator.of(context).push(
                            MaterialPageRoute(
                              builder: (_) => DocumentEditorPage(
                                filePath: file.apiPath,
                                deviceSerial: file.deviceSerial,
                              ),
                            ),
                          );
                          return;
                        }
                        _openOrDownload(file);
                      },
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

  final CirrusFileNode file;
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
          borderRadius: BorderRadius.circular(AutobutlerColors.radiusMd),
        ),
        clipBehavior: Clip.antiAlias,
        child: InkWell(
          onTap: onTap,
          child: Padding(
            padding: const EdgeInsets.fromLTRB(10, 8, 6, 8),
            child: Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                AutobutlerFileIcon(
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
                          Icons.folder_open_rounded,
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
