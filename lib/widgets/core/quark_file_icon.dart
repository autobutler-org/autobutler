import 'package:quark/models/file_node.dart';
import 'package:quark_icons/quark_icons.dart';
import 'package:flutter/material.dart';

class QuarkFileIcon extends StatelessWidget {
  const QuarkFileIcon({
    required this.node,
    this.size = 20.0,
    this.color,
    super.key,
  });

  final FileNode node;
  final double size;
  final Color? color;

  static IconData iconForNode(FileNode node) {
    if (node.isDir) return QuarkIcons.folder_outlined;
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
      return QuarkIcons.image_outlined;
    }

    if (lower.endsWith('.mp4') ||
        lower.endsWith('.mov') ||
        lower.endsWith('.mkv') ||
        lower.endsWith('.avi') ||
        lower.endsWith('.webm') ||
        lower.endsWith('.m4v') ||
        lower.endsWith('.wmv') ||
        lower.endsWith('.flv')) {
      return QuarkIcons.video_file_outlined;
    }

    if (lower.endsWith('.mp3') ||
        lower.endsWith('.wav') ||
        lower.endsWith('.flac') ||
        lower.endsWith('.aac') ||
        lower.endsWith('.ogg') ||
        lower.endsWith('.m4a') ||
        lower.endsWith('.wma') ||
        lower.endsWith('.opus')) {
      return QuarkIcons.audio_file_outlined;
    }

    if (lower.endsWith('.pdf')) return QuarkIcons.picture_as_pdf_outlined;

    if (lower.endsWith('.zip') ||
        lower.endsWith('.tar') ||
        lower.endsWith('.gz') ||
        lower.endsWith('.bz2') ||
        lower.endsWith('.xz') ||
        lower.endsWith('.7z') ||
        lower.endsWith('.rar') ||
        lower.endsWith('.zst')) {
      return QuarkIcons.archive_outlined;
    }

    // Quark native formats
    if (lower.endsWith('.abdoc')) return QuarkIcons.edit_document;
    if (lower.endsWith('.absheet')) return QuarkIcons.table_chart;

    if (lower.endsWith('.doc') ||
        lower.endsWith('.docx') ||
        lower.endsWith('.odt') ||
        lower.endsWith('.rtf')) {
      return QuarkIcons.description_outlined;
    }

    if (lower.endsWith('.xls') ||
        lower.endsWith('.xlsx') ||
        lower.endsWith('.ods') ||
        lower.endsWith('.csv')) {
      return QuarkIcons.table_chart_outlined;
    }

    if (lower.endsWith('.ppt') ||
        lower.endsWith('.pptx') ||
        lower.endsWith('.odp')) {
      return QuarkIcons.slideshow_outlined;
    }

    return QuarkIcons.insert_drive_file_outlined;
  }

  @override
  Widget build(BuildContext context) {
    return Icon(iconForNode(node), size: size, color: color);
  }
}
