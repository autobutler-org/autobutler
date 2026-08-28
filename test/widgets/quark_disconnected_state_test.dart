import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:quark/services/app_settings.dart';
import 'package:quark/widgets/core/quark_disconnected_state.dart';

/// #1637: away from the Quark's network the app used to show whatever
/// connection error came back. These pin the replacement — the plain
/// statement, the troubleshooting steps, and the fact that neither shape ever
/// renders an exception.
void main() {
  final settings = AppSettings.instance;

  Future<void> clearHosts() async {
    while (settings.hosts.isNotEmpty) {
      await settings.removeHost(settings.hosts.length - 1);
    }
  }

  setUp(clearHosts);
  tearDown(clearHosts);

  Future<void> pump(WidgetTester tester, Widget child) async {
    tester.view.physicalSize = const Size(1200, 2400);
    tester.view.devicePixelRatio = 1.0;
    addTearDown(tester.view.reset);
    await tester.pumpWidget(MaterialApp(home: Scaffold(body: child)));
    await tester.pumpAndSettle();
  }

  group('QuarkDisconnectedView', () {
    testWidgets('states plainly that the app is not connected', (tester) async {
      await pump(tester, const QuarkDisconnectedView());

      expect(find.text(quarkDisconnectedHeadline), findsOneWidget);
      expect(find.text(quarkDisconnectedBody), findsOneWidget);
    });

    testWidgets('lists every troubleshooting step', (tester) async {
      await pump(tester, const QuarkDisconnectedView());

      for (final step in quarkTroubleshootingSteps) {
        expect(find.text(step), findsOneWidget);
      }
    });

    testWidgets('shows the address it could not reach', (tester) async {
      await settings.addHost(
        HostEntry(name: 'Home', hostAddress: 'https://quark.local'),
      );

      await pump(tester, const QuarkDisconnectedView());

      expect(find.text('https://quark.local'), findsOneWidget);
    });

    testWidgets('offers no actions unless the caller supplies them', (
      tester,
    ) async {
      await pump(tester, const QuarkDisconnectedView());

      // "Check the address" must not appear on a page that cannot go
      // anywhere useful — a button leading nowhere is worse than none.
      expect(find.text('Try again'), findsNothing);
      expect(find.text('Check the address'), findsNothing);
    });

    testWidgets('retries through the callback the page supplied', (
      tester,
    ) async {
      var retries = 0;
      await pump(tester, QuarkDisconnectedView(onRetry: () => retries++));

      await tester.tap(find.text('Try again'));
      await tester.pumpAndSettle();

      expect(retries, 1);
    });

    testWidgets('routes to host management through its callback', (
      tester,
    ) async {
      var opened = 0;
      await pump(tester, QuarkDisconnectedView(onManageHosts: () => opened++));

      await tester.tap(find.text('Check the address'));
      await tester.pumpAndSettle();

      expect(opened, 1);
    });
  });

  group('QuarkDisconnectedBanner', () {
    testWidgets('carries the same message in the compact shape', (
      tester,
    ) async {
      await pump(tester, const QuarkDisconnectedBanner());

      expect(find.text(quarkDisconnectedHeadline), findsOneWidget);
      expect(find.text(quarkDisconnectedBody), findsOneWidget);
    });

    testWidgets('does not send the user to Settings they are already on', (
      tester,
    ) async {
      await pump(tester, const QuarkDisconnectedBanner());

      expect(
        find.text('Confirm the Quark address is correct.'),
        findsOneWidget,
      );
      expect(
        find.text('Confirm the Quark address in Settings is correct.'),
        findsNothing,
      );
    });

    testWidgets('gives no directional guidance', (tester) async {
      await pump(tester, const QuarkDisconnectedBanner());

      // Login renders the host card above this banner and Settings renders
      // host management below it, so any on-screen direction is wrong on one
      // of them. Naming a destination is fine; pointing is not.
      for (final step in quarkTroubleshootingStepsInPlace) {
        expect(step, isNot(contains('below')));
        expect(step, isNot(contains('above')));
      }
    });

    testWidgets('retries through the callback the page supplied', (
      tester,
    ) async {
      var retries = 0;
      await pump(tester, QuarkDisconnectedBanner(onRetry: () => retries++));

      await tester.tap(find.text('Try again'));
      await tester.pumpAndSettle();

      expect(retries, 1);
    });
  });
}
