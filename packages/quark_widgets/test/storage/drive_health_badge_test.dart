import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:quark_widgets/quark_widgets.dart';

import '../support/pump.dart';

/// The health chip names the verdict in words, at both viewports, from the
/// token colors. Unknown does not wear the healthy color.
void main() {
  for (final verdict in DriveHealthVerdict.values) {
    testBothViewports('reads ${verdict.label}', (tester, size) async {
      await pumpAt(
        tester,
        Center(child: DriveHealthBadge(verdict: verdict)),
        size: size,
      );

      expect(find.byKey(const ValueKey('drive_health_badge')), findsOneWidget);
      expect(find.text(verdict.label), findsOneWidget);
      for (final other in DriveHealthVerdict.values) {
        if (other == verdict) continue;
        expect(find.text(other.label), findsNothing);
      }
      expect(tester.takeException(), isNull);
      expectNoClippedText(tester);
    });
  }

  testBothViewports('gives every verdict its own key', (tester, size) async {
    await pumpAt(
      tester,
      Wrap(
        children: [
          for (final verdict in DriveHealthVerdict.values)
            DriveHealthBadge(id: verdict.name, verdict: verdict),
        ],
      ),
      size: size,
    );

    for (final verdict in DriveHealthVerdict.values) {
      expect(
        find.byKey(ValueKey('drive_health_badge_${verdict.name}')),
        findsOneWidget,
      );
      expect(find.text(verdict.label), findsOneWidget);
    }
    expect(tester.takeException(), isNull);
  });

  for (final (label, brightness, tokens) in [
    ('dark', Brightness.dark, QuarkTokens.dark),
    ('light', Brightness.light, QuarkTokens.light),
  ]) {
    testWidgets('$label: the word and the outline come from the tokens', (
      tester,
    ) async {
      await pumpAt(
        tester,
        const Center(
          child: DriveHealthBadge(verdict: DriveHealthVerdict.unknown),
        ),
        brightness: brightness,
      );

      final box = tester.widget<DecoratedBox>(
        find.byKey(const ValueKey('drive_health_badge')),
      );
      final decoration = box.decoration as BoxDecoration;
      final color = driveHealthVerdictColor(tokens, DriveHealthVerdict.unknown);
      expect((decoration.border! as Border).top.color, color);
      expect(tester.widget<Text>(find.text('Unknown')).style?.color, color);
      expect(color, isNot(tokens.success));
      expect(find.text('Healthy'), findsNothing);
    });

    testWidgets('$label: replace soon uses the error color and its word', (
      tester,
    ) async {
      await pumpAt(
        tester,
        const Center(
          child: DriveHealthBadge(verdict: DriveHealthVerdict.replaceSoon),
        ),
        brightness: brightness,
      );

      expect(find.text('Replace soon'), findsOneWidget);
      expect(
        tester.widget<Text>(find.text('Replace soon')).style?.color,
        tokens.error,
      );
    });
  }

  testWidgets('every icon-only button carries a tooltip', (tester) async {
    await pumpAt(
      tester,
      const Center(
        child: DriveHealthBadge(verdict: DriveHealthVerdict.healthy),
      ),
    );
    for (final button in tester.widgetList<IconButton>(
      find.byType(IconButton),
    )) {
      expect(button.tooltip, isNotNull);
    }
  });
}
