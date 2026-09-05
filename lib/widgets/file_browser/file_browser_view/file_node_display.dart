import 'package:quark/models/file_node.dart';

bool _isImageFile(FileNode node) {
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

/// Extensions the server classifies as a video; kept in sync with
/// `storageutil.DetermineFileTypeFromPath` so the two agree on what the
/// thumbnail endpoint will accept.
const _videoExtensions = <String>{
  '.mp4',
  '.m4v',
  '.webm',
  '.ogv',
  '.avi',
  '.mov',
  '.mkv',
  '.wmv',
  '.flv',
  '.3gp',
  '.3g2',
  '.mpeg',
  '.mpg',
  '.ts',
};

bool _isVideoFile(FileNode node) {
  if (node.isDir) return false;
  if (node.fileType == 'video') return true;
  final lower = node.name.toLowerCase();
  final dot = lower.lastIndexOf('.');
  return dot >= 0 && _videoExtensions.contains(lower.substring(dot));
}

/// Whether the server can render a thumbnail for this node. Videos go
/// through ffmpeg frame extraction on the backend and come back as JPEG,
/// so they use the same thumbnail URL as images.
bool hasServerThumbnail(FileNode node) =>
    _isImageFile(node) || _isVideoFile(node);

bool isArchiveNode(FileNode node) {
  if (node.isDir) return false;
  return node.fileType == 'archive';
}

String formatFileSize(int bytes, bool isDir, {int compressedSize = 0}) {
  if (isDir) return '--';
  final sizeStr = _formatBytes(bytes);
  if (compressedSize > 0 && compressedSize != bytes) {
    return '${_formatBytes(compressedSize)} → $sizeStr';
  }
  return sizeStr;
}

String _formatBytes(int bytes) {
  if (bytes < 1024) return '$bytes B';
  if (bytes < 1024 * 1024) return '${(bytes / 1024).toStringAsFixed(1)} KB';
  if (bytes < 1024 * 1024 * 1024) {
    return '${(bytes / (1024 * 1024)).toStringAsFixed(1)} MB';
  }
  return '${(bytes / (1024 * 1024 * 1024)).toStringAsFixed(1)} GB';
}
