import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:quark/pages/file_browser_page.dart';

void main() {
  // Full mock-client-based flow testing (login page, setup page) is deferred
  // to #691 once DI is in place. The auth gate itself lives in `authRedirect`
  // (lib/router.dart) and is covered by test/router_test.dart.

  testWidgets('FileBrowserPage renders core UI elements', (
    WidgetTester tester,
  ) async {
    await tester.pumpWidget(const MaterialApp(home: FileBrowserPage()));
    await tester.pump();

    expect(find.byIcon(Icons.storage_rounded), findsOneWidget);
    expect(find.text('Files'), findsOneWidget);
  });
}
