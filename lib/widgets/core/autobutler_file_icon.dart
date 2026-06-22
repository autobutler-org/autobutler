import 'package:autobutler/models/cirrus_file_node.dart';
import 'package:flutter/material.dart';

class AutobutlerFileIcon extends StatelessWidget {
  const AutobutlerFileIcon({
    required this.node,
    this.size = 20.0,
    this.color,
    super.key,
  });

  final CirrusFileNode node;
  final double size;
  final Color? color;

  static IconData iconForNode(CirrusFileNode node) {
    if (node.isDir) return Icons.folder_outlined;
    final lower = node.name.toLowerCase();

    if (lower.endsWith('.jpg') ||
        lower.endsWith('.jpeg') ||
        lower.endsWith('.png') ||
        lower.endsWith('.gif') ||
        lower.endsWith('.webp') ||
        lower.endsWith('.heic') ||
        lower.endsWith('.heif') ||
        lower.endsWith('.bmp') ||
        lower.endsWith('.tiff') ||
        lower.endsWith('.tif') ||
        lower.endsWith('.raw') ||
        lower.endsWith('.cr2') ||
        lower.endsWith('.cr3') ||
        lower.endsWith('.nef') ||
        lower.endsWith('.arw') ||
        lower.endsWith('.dng') ||
        lower.endsWith('.orf') ||
        lower.endsWith('.rw2')) {
      return Icons.image_outlined;
    }

    if (lower.endsWith('.mp4') ||
        lower.endsWith('.mov') ||
        lower.endsWith('.mkv') ||
        lower.endsWith('.avi') ||
        lower.endsWith('.webm') ||
        lower.endsWith('.m4v') ||
        lower.endsWith('.wmv') ||
        lower.endsWith('.flv')) {
      return Icons.video_file_outlined;
    }

    if (lower.endsWith('.mp3') ||
        lower.endsWith('.wav') ||
        lower.endsWith('.flac') ||
        lower.endsWith('.aac') ||
        lower.endsWith('.ogg') ||
        lower.endsWith('.m4a') ||
        lower.endsWith('.opus')) {
      return Icons.audio_file_outlined;
    }

    if (lower.endsWith('.pdf')) return Icons.picture_as_pdf_outlined;

    if (lower.endsWith('.zip') ||
        lower.endsWith('.tar') ||
        lower.endsWith('.gz') ||
        lower.endsWith('.bz2') ||
        lower.endsWith('.xz') ||
        lower.endsWith('.7z') ||
        lower.endsWith('.rar') ||
        lower.endsWith('.zst')) {
      return Icons.archive_outlined;
    }

    // AutoButler native formats
    if (lower.endsWith('.abdoc')) return Icons.edit_document;
    if (lower.endsWith('.absheet')) return Icons.table_chart;

    if (lower.endsWith('.doc') ||
        lower.endsWith('.docx') ||
        lower.endsWith('.odt') ||
        lower.endsWith('.rtf')) {
      return Icons.description_outlined;
    }

    if (lower.endsWith('.xls') ||
        lower.endsWith('.xlsx') ||
        lower.endsWith('.ods') ||
        lower.endsWith('.csv')) {
      return Icons.table_chart_outlined;
    }

    if (lower.endsWith('.ppt') ||
        lower.endsWith('.pptx') ||
        lower.endsWith('.odp')) {
      return Icons.slideshow_outlined;
    }

    return Icons.insert_drive_file_outlined;
  }

  @override
  Widget build(BuildContext context) {
    return Icon(iconForNode(node), size: size, color: color);
  }
}
