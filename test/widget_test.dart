import 'package:autobutler/pages/auth_gate.dart';
import 'package:autobutler/pages/file_browser_page.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';

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
    expect(find.text('Cirrus'), findsOneWidget);
    expect(find.byIcon(Icons.menu), findsOneWidget);
  });

  testWidgets('FileBrowserPage renders core UI elements', (
    WidgetTester tester,
  ) async {
    await tester.pumpWidget(const MaterialApp(home: FileBrowserPage()));
    await tester.pump();

    expect(find.byIcon(Icons.menu), findsOneWidget);
    expect(find.text('Cirrus'), findsOneWidget);
  });
}
