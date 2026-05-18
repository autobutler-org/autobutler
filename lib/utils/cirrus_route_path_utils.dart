String cirrusRouteDisplayPath(String path) {
  final trimmed = path.trim();
  if (trimmed.isEmpty || trimmed == '/') {
    return '/cirrus';
  }
  return trimmed.startsWith('/cirrus') ? trimmed : '/cirrus$trimmed';
}

bool isLikelyFilePath(String path) {
  final trimmed = path.trim();
  if (trimmed.isEmpty || trimmed.endsWith('/')) {
    return false;
  }

  final normalized = trimmed.startsWith('/') ? trimmed.substring(1) : trimmed;
  final segments = normalized.split('/');
  if (segments.isEmpty) {
    return false;
  }

  final lastSegment = segments.last;
  final dotIndex = lastSegment.lastIndexOf('.');
  return dotIndex > 0 && dotIndex < lastSegment.length - 1;
}

bool hasSupportedCirrusEditorForPath(String path) {
  final normalized = path.trim().toLowerCase();
  if (normalized.endsWith('.abdoc') || normalized.endsWith('.absheet')) {
    return true;
  }
  const imageExts = [
    '.jpg',
    '.jpeg',
    '.png',
    '.gif',
    '.webp',
    '.heic',
    '.heif',
  ];
  const videoExts = [
    '.mp4',
    '.m4v',
    '.webm',
    '.ogg',
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
  ];
  for (final ext in [...imageExts, ...videoExts]) {
    if (normalized.endsWith(ext)) return true;
  }
  return false;
}

bool hasSupportedCirrusEditorForType(String fileType) {
  final normalized = fileType.trim().toLowerCase();
  return normalized == 'abdoc' ||
      normalized == 'absheet' ||
      normalized == 'video' ||
      normalized == 'image';
}
