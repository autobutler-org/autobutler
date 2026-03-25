import 'package:autobutler/models/cirrus_file_node.dart';
import 'package:flutter/material.dart';

/// Canonical file-type icon widget for AutoButler.
///
/// Consolidates the icon-mapping logic that was previously duplicated in
/// [FileBrowserView] and [RecentFilesSection] into a single source of truth.
class AutobutlerFileIcon extends StatelessWidget {
  const AutobutlerFileIcon({required this.node, this.size = 20, super.key});

  final CirrusFileNode node;
  final double size;

  @override
  Widget build(BuildContext context) {
    return Icon(iconForNode(node), size: size);
  }

  /// Returns the canonical [IconData] for the given [CirrusFileNode].
  ///
  /// This is exposed as a static method so callers that need just the icon data
  /// (e.g. for a leading icon in a ListTile) can use it without wrapping in the
  /// widget.
  static IconData iconForNode(CirrusFileNode node) {
    if (node.isDir) return Icons.folder_outlined;

    final lower = node.name.toLowerCase();

    // Images
    if (lower.endsWith('.jpg') ||
        lower.endsWith('.jpeg') ||
        lower.endsWith('.png') ||
        lower.endsWith('.gif') ||
        lower.endsWith('.webp')) {
      return Icons.image_outlined;
    }

    // Archives
    if (lower.endsWith('.zip') ||
        lower.endsWith('.tar') ||
        lower.endsWith('.gz') ||
        lower.endsWith('.7z') ||
        lower.endsWith('.rar')) {
      return Icons.archive_outlined;
    }

    // PDF
    if (lower.endsWith('.pdf')) return Icons.picture_as_pdf_outlined;

    // Video
    if (lower.endsWith('.mp4') ||
        lower.endsWith('.mov') ||
        lower.endsWith('.mkv') ||
        lower.endsWith('.avi') ||
        lower.endsWith('.webm')) {
      return Icons.video_file_outlined;
    }

    // Audio
    if (lower.endsWith('.mp3') ||
        lower.endsWith('.wav') ||
        lower.endsWith('.flac') ||
        lower.endsWith('.aac') ||
        lower.endsWith('.ogg')) {
      return Icons.audio_file_outlined;
    }

    return Icons.insert_drive_file_outlined;
  }
}
