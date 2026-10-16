import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:quark_widgets/quark_widgets.dart';

import '../support/pump.dart';

/// The compact protection chip. It reads the level it is given, at both
/// viewports, from the token colors.
void main() {
  for (final level in DataProtectionLevel.values) {
    testBothViewports('reads ${level.label}', (tester, size) async {
      await pumpAt(
        tester,
        Center(child: DataProtectionBadge(level: level)),
        size: size,
      );

      expect(
        find.byKey(const ValueKey('data_protection_badge')),
        findsOneWidget,
      );
      expect(find.text(level.label), findsOneWidget);
      for (final other in DataProtectionLevel.values) {
        if (other == level) continue;
        expect(find.text(other.label), findsNothing);
      }
      expect(tester.takeException(), isNull);
    });
  }

  testBothViewports('survives a long tree of badges', (tester, size) async {
    await pumpAt(
      tester,
      Wrap(
        children: [
          for (final level in DataProtectionLevel.values)
            DataProtectionBadge(id: level.name, level: level),
        ],
      ),
      size: size,
    );

    for (final level in DataProtectionLevel.values) {
      expect(
        find.byKey(ValueKey('data_protection_badge_${level.name}')),
        findsOneWidget,
      );
    }
    expect(tester.takeException(), isNull);
  });

  for (final (label, brightness, tokens) in [
    ('dark', Brightness.dark, QuarkTokens.dark),
    ('light', Brightness.light, QuarkTokens.light),
  ]) {
    testWidgets('$label: the outline comes from the tokens', (tester) async {
      await pumpAt(
        tester,
        const Center(
          child: DataProtectionBadge(level: DataProtectionLevel.protected),
        ),
        brightness: brightness,
      );

      final box = tester.widget<DecoratedBox>(
        find.byKey(const ValueKey('data_protection_badge')),
      );
      final decoration = box.decoration as BoxDecoration;
      expect(
        (decoration.border! as Border).top.color,
        dataProtectionLevelColor(tokens, DataProtectionLevel.protected),
      );
      expect(
        tester.widget<Text>(find.text('Protected')).style?.color,
        dataProtectionLevelColor(tokens, DataProtectionLevel.protected),
      );
    });
  }
}
