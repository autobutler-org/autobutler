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
  return normalized == 'abdoc' || normalized == 'absheet';
}

/// Backend file types with no in-app viewer yet.
///
/// These open in `GenericFileViewerPage` — download plus "Open with…" — rather
/// than falling through to the "No supported editor" dead end. Named document
/// types are included deliberately: without them a `.pdf` ends up worse off
/// than an unclassified file, which reaches that page as `generic` (#1184).
bool usesGenericFileViewer(String fileType) {
  const noInAppViewer = {'generic', 'pdf', 'docx', 'slideshow', 'epub'};
  final normalized = fileType.trim().toLowerCase();
  return normalized.isEmpty || noInAppViewer.contains(normalized);
}
