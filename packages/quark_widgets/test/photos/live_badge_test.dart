import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:quark_widgets/quark_widgets.dart';

import '../support/pump.dart';

void main() {
  testBothViewports('reads LIVE', (tester, size) async {
    await pumpAt(tester, const LiveBadge(), size: size);

    expect(find.text('LIVE'), findsOneWidget);
  });

  testWidgets('keeps its scrim and white text in the light theme', (
    tester,
  ) async {
    // It is drawn on a photograph, not on a surface, so following the theme
    // into light mode would make it disappear against a bright picture.
    await pumpAt(
      tester,
      const LiveBadge(),
      size: narrowViewport,
      brightness: Brightness.light,
    );

    final text = tester.widget<Text>(find.text('LIVE'));
    expect(text.style?.color, Colors.white);

    final container = tester.widget<Container>(find.byType(Container));
    expect((container.decoration! as BoxDecoration).color, Colors.black54);
  });

  testWidgets('stays small enough for a thumbnail corner', (tester) async {
    await pumpAt(
      tester,
      const Align(alignment: Alignment.topLeft, child: LiveBadge()),
      size: narrowViewport,
    );

    expect(tester.getSize(find.byType(LiveBadge)).height, lessThan(24));
  });
}
