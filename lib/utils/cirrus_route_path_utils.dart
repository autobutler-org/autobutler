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
  return normalized.endsWith('.abdoc') || normalized.endsWith('.absheet');
}

bool hasSupportedCirrusEditorForType(String fileType) {
  final normalized = fileType.trim().toLowerCase();
  return normalized == 'abdoc' ||
      normalized == 'absheet' ||
      normalized == 'video' ||
      normalized == 'image';
}
