import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:quark_widgets/quark_widgets.dart';

import '../support/pump.dart';

void main() {
  testBothViewports('offers dark while the app is light', (tester, size) async {
    final chosen = <ThemeMode>[];
    await pumpAt(
      tester,
      ThemeToggleButton(mode: ThemeMode.light, onChanged: chosen.add),
      size: size,
    );

    expect(find.byIcon(Icons.dark_mode), findsOneWidget);
    expect(find.byTooltip('Switch to dark mode'), findsOneWidget);

    await tester.tap(find.byKey(const ValueKey('theme_toggle')));
    await tester.pump();

    expect(chosen, [ThemeMode.dark]);
  });

  testBothViewports('offers light while the app is dark', (tester, size) async {
    final chosen = <ThemeMode>[];
    await pumpAt(
      tester,
      ThemeToggleButton(mode: ThemeMode.dark, onChanged: chosen.add),
      size: size,
    );

    expect(find.byIcon(Icons.light_mode), findsOneWidget);

    await tester.tap(find.byKey(const ValueKey('theme_toggle')));
    await tester.pump();

    expect(chosen, [ThemeMode.light]);
  });

  testWidgets('commits to light from the system setting', (tester) async {
    final chosen = <ThemeMode>[];
    await pumpAt(
      tester,
      ThemeToggleButton(mode: ThemeMode.system, onChanged: chosen.add),
      size: narrowViewport,
    );

    expect(find.byIcon(Icons.brightness_auto), findsOneWidget);

    await tester.tap(find.byKey(const ValueKey('theme_toggle')));
    await tester.pump();

    expect(chosen, [ThemeMode.light]);
  });
}
