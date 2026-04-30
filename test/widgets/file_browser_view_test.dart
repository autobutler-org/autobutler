import 'dart:async';

import 'package:autobutler/models/cirrus_file_node.dart';
import 'package:autobutler/widgets/file_browser/file_browser_view.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  Future<void> pumpFileBrowserView(
    WidgetTester tester, {
    required Future<List<CirrusFileNode>> filesFuture,
    bool isInitialLoad = false,
    Widget Function(BuildContext context, Object error)? errorBuilder,
    WidgetBuilder? loadingBuilder,
  }) {
    return tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: FileBrowserView(
            filesFuture: filesFuture,
            isInitialLoad: isInitialLoad,
            currentPath: '/Documents',
            onFileMenuAction: (_, __) async {},
            onOpenDirectory: (_) {},
            isGridView: false,
            errorBuilder: errorBuilder,
            loadingBuilder: loadingBuilder,
          ),
        ),
      ),
    );
  }

  testWidgets('uses the custom loading builder during initial loads', (
    WidgetTester tester,
  ) async {
    await pumpFileBrowserView(
      tester,
      filesFuture: Future.value(const <CirrusFileNode>[]),
      isInitialLoad: true,
      loadingBuilder: (_) => const Text('Opening /cirrus/Documents'),
    );

    expect(find.text('Opening /cirrus/Documents'), findsOneWidget);
    expect(find.byType(CircularProgressIndicator), findsNothing);
  });

  testWidgets('uses the custom error builder for failed folder loads', (
    WidgetTester tester,
  ) async {
    final completer = Completer<List<CirrusFileNode>>();
    await pumpFileBrowserView(
      tester,
      filesFuture: completer.future,
      errorBuilder: (_, error) => Text('Route error: $error'),
    );
    completer.completeError(Exception('folder not found'));
    await tester.pumpAndSettle();

    expect(
      find.text('Route error: Exception: folder not found'),
      findsOneWidget,
    );
    expect(find.text('Unable to load files'), findsNothing);
  });
}
