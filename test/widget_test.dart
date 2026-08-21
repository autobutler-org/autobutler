import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:quark/pages/auth_gate.dart';
import 'package:quark/pages/file_browser_page.dart';

void main() {
  // AuthGate skips authentication entirely when no host is configured —
  // it goes straight to the child widget. Full mock-client-based flow
  // testing (login page, setup page) is deferred to #691 once DI is in place.

  testWidgets('AuthGate passes through to child when no host is configured', (
    WidgetTester tester,
  ) async {
    // AppSettings has no host by default in test environment (no SharedPreferences data).
    // AuthGate should skip auth and render the child immediately.
    await tester.pumpWidget(
      const MaterialApp(home: AuthGate(child: FileBrowserPage())),
    );
    // Let initState/_check() complete synchronously (no host = no async work).
    await tester.pump();

    // Child renders — file browser is visible, not a login screen.
    // The top bar shows 'Files' as the brand label (restyled from 'Cirrus').
    expect(find.text('Files'), findsOneWidget);
    expect(find.byIcon(Icons.storage_rounded), findsOneWidget);
  });

  testWidgets('FileBrowserPage renders core UI elements', (
    WidgetTester tester,
  ) async {
    await tester.pumpWidget(const MaterialApp(home: FileBrowserPage()));
    await tester.pump();

    expect(find.byIcon(Icons.storage_rounded), findsOneWidget);
    expect(find.text('Files'), findsOneWidget);
  });
}
