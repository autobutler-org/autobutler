import 'package:autobutler/pages/auth_gate.dart';
import 'package:autobutler/pages/file_browser_page.dart';
import 'package:autobutler/services/app_settings.dart';
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';

Widget _testApp(Widget home) {
  return MaterialApp(
    title: 'Autobutler',
    theme: ThemeData(
      colorScheme: ColorScheme.fromSeed(
        seedColor: Colors.blue,
        brightness: Brightness.light,
      ),
      useMaterial3: true,
    ),
    home: home,
  );
}

void main() {
  setUp(() async {
    // Provide a clean in-memory SharedPreferences for every test.
    SharedPreferences.setMockInitialValues({});
    await AppSettings.instance.load();
  });

  testWidgets('renders file browser screen', (WidgetTester tester) async {
    // Pump FileBrowserPage directly to verify the file browser renders.
    // AuthGate is tested separately below.
    await tester.pumpWidget(_testApp(const FileBrowserPage()));
    await tester.pumpAndSettle();

    expect(find.text('Cirrus'), findsOneWidget);
    expect(find.text('Name'), findsOneWidget);
    expect(find.text('Device'), findsOneWidget);
    expect(find.text('Size'), findsOneWidget);
  });

  testWidgets('AuthGate passes through when no host is configured', (
    WidgetTester tester,
  ) async {
    // With no host configured (default after setMockInitialValues({})),
    // AuthGate should skip the auth check and render its child immediately.
    await tester.pumpWidget(_testApp(const AuthGate(child: FileBrowserPage())));
    // First pump: loading spinner
    // After settle: no host → authenticated → child renders
    await tester.pumpAndSettle();

    expect(find.text('Cirrus'), findsOneWidget);
  });
}
