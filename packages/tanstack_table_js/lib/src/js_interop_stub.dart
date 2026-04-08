// Stub implementation for non-web platforms.

void registerViewFactory(String viewId) {
  throw UnsupportedError('registerViewFactory is only supported on web');
}

void createTableInJs(String elementId, List<Map<String, dynamic>> data, List<Map<String, dynamic>> columns) {
  throw UnsupportedError('createTableInJs is only supported on web');
}

void updateTableInJs(String elementId, List<Map<String, dynamic>> data, List<Map<String, dynamic>> columns) {
  throw UnsupportedError('updateTableInJs is only supported on web');
}
