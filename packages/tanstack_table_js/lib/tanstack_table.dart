library flutter_tanstack_table;

import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:flutter/widgets.dart';

// Web-only implementation. On non-web platforms this widget displays a placeholder.

export 'src/js_interop.dart';

class TanstackTable extends StatefulWidget {
  final List<Map<String, dynamic>> columns;
  final List<Map<String, dynamic>> data;
  final double? height;
  final double? width;

  const TanstackTable({
    Key? key,
    required this.columns,
    required this.data,
    this.height,
    this.width,
  }) : super(key: key);

  @override
  State<TanstackTable> createState() => _TanstackTableState();
}

class _TanstackTableState extends State<TanstackTable> {
  late final String _viewId;
  bool _registered = false;

  @override
  void initState() {
    super.initState();
    _viewId = 'tanstack_table_${DateTime.now().millisecondsSinceEpoch}_${uniqueIdCounter++}';
  }

  static int uniqueIdCounter = 0;

  @override
  Widget build(BuildContext context) {
    if (!kIsWeb) {
      return const Center(child: Text('TanstackTable is only supported on Flutter web'));
    }

    // Register a platform view and attach the JS renderer.
    // Use a SizedBox to control size if provided.
    final container = SizedBox(
      height: widget.height,
      width: widget.width,
      child: HtmlElementView(viewType: _viewId),
    );

    // Registration must occur once.
    if (!_registered) {
      // ignore: undefined_prefixed_name
      // Register view factory for web
      // `ui.platformViewRegistry` is available only on web
      // Using a runtime import avoids analyzer errors on non-web platforms
      registerViewFactory(_viewId);
      // Delay a tick so the element exists before JS call
      scheduleMicrotask(() {
        // Use the JS interop helper to create the table
        createTableInJs(_viewId, widget.data, widget.columns);
      });
      _registered = true;
    } else {
      // update data on rebuild
      scheduleMicrotask(() {
        updateTableInJs(_viewId, widget.data, widget.columns);
      });
    }

    return container;
  }
}

// The web-specific registration function uses dart:ui and dart:html.

// Placeholders to avoid import errors on non-web platforms. The real implementations
// live in src/js_interop.dart and are conditional on web.

// ignore: non_constant_identifier_names
void registerViewFactory(String viewId) {
  // implementation provided in src/js_interop.dart
  throw UnsupportedError('registerViewFactory is only supported on web');
}

// ignore: non_constant_identifier_names
void createTableInJs(String viewId, List<Map<String, dynamic>> data, List<Map<String, dynamic>> columns) {
  throw UnsupportedError('createTableInJs is only supported on web');
}

// ignore: non_constant_identifier_names
void updateTableInJs(String viewId, List<Map<String, dynamic>> data, List<Map<String, dynamic>> columns) {
  throw UnsupportedError('updateTableInJs is only supported on web');
}
