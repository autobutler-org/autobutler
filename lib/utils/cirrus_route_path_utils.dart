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
