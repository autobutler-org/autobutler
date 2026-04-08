// Web-only JS interop implementation for flutter_tanstack_table

import 'dart:html' as html;
import 'dart:js' as js;
import 'dart:ui' as ui; // ignore: implementation_imports

// Register a view factory that creates a container div with a predictable id.
void registerViewFactory(String viewId) {
  // ignore: undefined_prefixed_name
  ui.platformViewRegistry.registerViewFactory(viewId, (int viewIdInt) {
    final div = html.DivElement();
    div.id = viewId;
    div.style.width = '100%';
    div.style.height = '100%';
    return div;
  });
}

// Call the JS glue to create the table
void createTableInJs(String elementId, List<Map<String, dynamic>> data, List<Map<String, dynamic>> columns) {
  final win = js.context;
  try {
    win.callMethod('createTanstackTable', [elementId, data, columns]);
  } catch (e) {
    // ignore errors if glue not loaded yet
    // schedule a retry
    Future.delayed(const Duration(milliseconds: 50), () => createTableInJs(elementId, data, columns));
  }
}

void updateTableInJs(String elementId, List<Map<String, dynamic>> data, List<Map<String, dynamic>> columns) {
  final win = js.context;
  try {
    win.callMethod('updateTanstackTable', [elementId, data, columns]);
  } catch (e) {
    Future.delayed(const Duration(milliseconds: 50), () => updateTableInJs(elementId, data, columns));
  }
}
