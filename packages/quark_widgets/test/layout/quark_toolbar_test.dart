// The action row that refuses to overflow.
//
// The whole point is the narrow viewport, so every case runs at 360 pixels
// with more buttons than fit, and asserts that nothing threw and that the
// buttons are all still there and still tappable.
import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:quark_widgets/quark_widgets.dart';

import '../support/pump.dart';

/// Six wide buttons: comfortably more than a 360 pixel phone fits on one line.
List<Widget> actionsThatDoNotFit(void Function(String) onTap) => [
  for (final label in const [
    'Select all',
    'Deselect all',
    'Download',
    'Move to folder',
    'Add to album',
    'Delete forever',
  ])
    FilledButton(
      key: ValueKey('toolbar_$label'),
      onPressed: () => onTap(label),
      child: Text(label),
    ),
];

void main() {
  for (final overflow in QuarkToolbarOverflow.values) {
    testBothViewports('${overflow.name}: renders every action', (
      tester,
      size,
    ) async {
      await pumpAt(
        tester,
        QuarkToolbar(actions: actionsThatDoNotFit((_) {}), overflow: overflow),
        size: size,
      );

      expect(tester.takeException(), isNull);
      for (final label in const ['Select all', 'Delete forever']) {
        expect(find.byKey(ValueKey('toolbar_$label')), findsOneWidget);
      }
    });

    testBothViewports('${overflow.name}: reports the action that was tapped', (
      tester,
      size,
    ) async {
      final tapped = <String>[];
      await pumpAt(
        tester,
        QuarkToolbar(actions: actionsThatDoNotFit(tapped.add)),
        size: size,
      );

      await tester.tap(
        find.byKey(const ValueKey('toolbar_Select all')),
        warnIfMissed: false,
      );
      await tester.pump();

      expect(tapped, ['Select all']);
    });
  }

  testWidgets('wrap: too many actions run onto another line', (tester) async {
    await pumpAt(
      tester,
      QuarkToolbar(actions: actionsThatDoNotFit((_) {})),
      size: narrowViewport,
    );

    final first = tester.getRect(
      find.byKey(const ValueKey('toolbar_Select all')),
    );
    final last = tester.getRect(
      find.byKey(const ValueKey('toolbar_Delete forever')),
    );
    expect(
      last.top,
      greaterThan(first.top),
      reason: 'the row has to grow taller rather than overflow',
    );
    expect(tester.takeException(), isNull);
  });

  testWidgets('scroll: too many actions stay on one line', (tester) async {
    await pumpAt(
      tester,
      QuarkToolbar(
        actions: actionsThatDoNotFit((_) {}),
        overflow: QuarkToolbarOverflow.scroll,
      ),
      size: narrowViewport,
    );

    final first = tester.getRect(
      find.byKey(const ValueKey('toolbar_Select all')),
    );
    final last = tester.getRect(
      find.byKey(const ValueKey('toolbar_Delete forever')),
    );
    expect(last.top, first.top, reason: 'a fixed-height bar cannot wrap');
    expect(find.byType(SingleChildScrollView), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testBothViewports('renders nothing for no actions', (tester, size) async {
    await pumpAt(tester, const QuarkToolbar(actions: []), size: size);

    expect(tester.takeException(), isNull);
    expect(find.byType(FilledButton), findsNothing);
  });

  for (final (label, brightness, tokens) in [
    ('dark', Brightness.dark, QuarkTokens.dark),
    ('light', Brightness.light, QuarkTokens.light),
  ]) {
    testWidgets('$label: the gaps come from the tokens', (tester) async {
      await pumpAt(
        tester,
        QuarkToolbar(actions: actionsThatDoNotFit((_) {})),
        brightness: brightness,
      );

      expect(tester.takeException(), isNull);
      final wrap = tester.widget<Wrap>(find.byType(Wrap));
      expect(wrap.spacing, tokens.spacingSm);
      expect(wrap.runSpacing, tokens.spacingXs);
    });
  }

  testBothViewports('survives a single very long label', (tester, size) async {
    await pumpAt(
      tester,
      QuarkToolbar(
        actions: [
          FilledButton(
            onPressed: () {},
            child: Text('Move every selected file ' * 10),
          ),
        ],
      ),
      size: size,
    );

    expect(tester.takeException(), isNull);
  });
}
