import 'package:autobutler/models/cirrus_file_node.dart';
import 'package:autobutler/services/cirrus_service.dart';
import 'package:autobutler/theme/autobutler_colors.dart';
import 'package:flutter/material.dart';

/// A horizontally-scrolling strip showing recently uploaded files.
/// Displayed at the root of the file browser (not in search mode).
///
/// Tapping a file chip triggers [onOpenFile]; tapping the folder badge
/// triggers [onNavigateToFolder] with the parent directory path.
class RecentFilesSection extends StatefulWidget {
  const RecentFilesSection({
    required this.onOpenFile,
    required this.onNavigateToFolder,
    super.key,
  });

  final void Function(CirrusFileNode) onOpenFile;
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
    // apiPath is like "/photos/IMG_001.jpg" — strip the filename.
    final path = node.apiPath;
    final slash = path.lastIndexOf('/');
    if (slash <= 0) return '';
    return path.substring(0, slash);
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

        return Container(
          decoration: const BoxDecoration(
            border: Border(bottom: BorderSide(color: AutobutlerColors.border)),
          ),
          padding: const EdgeInsets.fromLTRB(16, 10, 16, 12),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            mainAxisSize: MainAxisSize.min,
            children: [
              Row(
                children: [
                  const Icon(
                    Icons.schedule_rounded,
                    size: 14,
                    color: AutobutlerColors.mutedForeground,
                  ),
                  const SizedBox(width: 6),
                  const Text(
                    'Recently uploaded',
                    style: TextStyle(
                      fontSize: 12,
                      fontWeight: FontWeight.w500,
                      color: AutobutlerColors.mutedForeground,
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
                  itemBuilder: (context, index) => _RecentFileChip(
                    file: files[index],
                    onTap: () => widget.onOpenFile(files[index]),
                    onFolderTap: () =>
                        widget.onNavigateToFolder(_parentPath(files[index])),
                  ),
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
    return MouseRegion(
      cursor: SystemMouseCursors.click,
      child: Material(
        color: AutobutlerColors.input,
        shape: RoundedRectangleBorder(
          side: const BorderSide(color: AutobutlerColors.border),
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
                Icon(
                  _iconForNode(file),
                  size: 20,
                  color: AutobutlerColors.secondaryForeground,
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
                        style: const TextStyle(
                          fontSize: 13,
                          color: AutobutlerColors.cardForeground,
                        ),
                      ),
                      if (file.deviceName.isNotEmpty)
                        Text(
                          file.deviceName,
                          maxLines: 1,
                          overflow: TextOverflow.ellipsis,
                          style: const TextStyle(
                            fontSize: 11,
                            color: AutobutlerColors.mutedForeground,
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
                      child: const Padding(
                        padding: EdgeInsets.all(4),
                        child: Icon(
                          Icons.folder_open_rounded,
                          size: 14,
                          color: AutobutlerColors.mutedForeground,
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

  static IconData _iconForNode(CirrusFileNode node) {
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
        lower.endsWith('.mkv')) {
      return Icons.video_file_outlined;
    }
    if (lower.endsWith('.mp3') ||
        lower.endsWith('.wav') ||
        lower.endsWith('.flac')) {
      return Icons.audio_file_outlined;
    }
    return Icons.insert_drive_file_outlined;
  }
}
