import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:quark_icons/quark_icons.dart';
import 'package:quark_widgets/quark_widgets.dart';

import '../support/pump.dart';

void main() {
  Widget page({List<Widget> actions = const []}) => Scaffold(
    appBar: QuarkAppBar(
      label: 'Photos',
      icon: QuarkIcons.photo_library_outlined,
      actions: actions,
    ),
    drawer: const QuarkDrawer(activeSection: QuarkDrawerSection.photos),
    body: const SizedBox.shrink(),
  );

  testBothViewports('leads with the brand button and no title', (
    tester,
    size,
  ) async {
    await pumpAt(tester, page(), size: size, scaffold: false);

    expect(find.byType(QuarkBrandButton), findsOneWidget);
    expect(find.text('Photos'), findsOneWidget);
    expect(tester.widget<AppBar>(find.byType(AppBar)).title, isNull);
  });

  testBothViewports('opens the drawer from the brand button', (
    tester,
    size,
  ) async {
    await pumpAt(tester, page(), size: size, scaffold: false);

    await tester.tap(find.byKey(const ValueKey('brand_button')));
    await tester.pumpAndSettle();

    expect(find.byType(QuarkDrawer), findsOneWidget);
  });

  testBothViewports('renders the actions it is given, in order', (
    tester,
    size,
  ) async {
    await pumpAt(
      tester,
      page(actions: const [Text('first'), Text('second')]),
      size: size,
      scaffold: false,
    );

    expect(
      tester.getRect(find.text('first')).left,
      lessThan(tester.getRect(find.text('second')).left),
    );
  });

  testWidgets('reserves enough leading width for the brand button', (
    tester,
  ) async {
    await pumpAt(tester, page(), size: narrowViewport, scaffold: false);

    final bar = tester.widget<AppBar>(find.byType(AppBar));
    expect(
      bar.leadingWidth,
      greaterThanOrEqualTo(QuarkBrandButton.preferredWidth),
    );
    expect(tester.takeException(), isNull);
  });
}
