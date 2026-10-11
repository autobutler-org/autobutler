import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:quark/services/app_settings.dart';
import 'package:quark/utils/error_text.dart';
import 'package:quark/widgets/host_manager.dart';
import 'package:shared_preferences/shared_preferences.dart';

/// #2721: a saved Quark with a pinned certificate offers Forget certificate,
/// and one that presented a different certificate says so in its row.
void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  const secureStorage = MethodChannel(
    'plugins.it_nomads.com/flutter_secure_storage',
  );
  const home = 'https://home.local';
  const forget = ValueKey('host_action_forget_certificate_0');
  const changed = ValueKey('host_certificate_changed_0');
  const viewports = {'narrow': Size(360, 640), 'wide': Size(1280, 800)};
  final settings = AppSettings.instance;

  setUpAll(() {
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(secureStorage, (_) async => null);
  });

  tearDownAll(() {
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(secureStorage, null);
  });

  Future<void> pumpManager(WidgetTester tester, Size viewport) async {
    tester.view.physicalSize = viewport;
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.reset);
    SharedPreferences.setMockInitialValues({
      'hosts': jsonEncode([
        {'name': 'Home', 'hostAddress': home},
        {'name': 'Cabin', 'hostAddress': 'https://cabin.local'},
      ]),
    });
    await settings.load();
    await tester.pumpWidget(
      const MaterialApp(
        home: Scaffold(body: SingleChildScrollView(child: HostManager())),
      ),
    );
    await tester.pump();
  }

  Future<void> openMenu(WidgetTester tester, int index) async {
    await tester.tap(find.byKey(ValueKey('host_menu_$index')));
    await tester.pumpAndSettle();
  }

  for (final MapEntry(key: name, value: viewport) in viewports.entries) {
    testWidgets('a Quark with no pin has nothing to forget ($name)', (
      tester,
    ) async {
      await pumpManager(tester, viewport);

      await openMenu(tester, 0);

      expect(find.byKey(const ValueKey('host_action_edit_0')), findsOneWidget);
      expect(find.byKey(forget), findsNothing);
    });

    testWidgets('Forget certificate re-trusts a refused Quark ($name)', (
      tester,
    ) async {
      await pumpManager(tester, viewport);
      settings.trustCertificate(home, 'aa11');
      settings.trustCertificate(home, 'bb22');
      await tester.pump();

      expect(find.byKey(changed), findsOneWidget);
      expect(find.text(Errors.certificateChanged), findsOneWidget);
      expect(tester.takeException(), isNull, reason: 'the notice fits');

      // Only the Quark that is pinned offers it.
      await openMenu(tester, 1);
      expect(
        find.byKey(const ValueKey('host_action_forget_certificate_1')),
        findsNothing,
      );
      await tester.tapAt(const Offset(5, 5));
      await tester.pumpAndSettle();

      await openMenu(tester, 0);
      await tester.tap(find.byKey(forget));
      await tester.pumpAndSettle();

      expect(find.byKey(changed), findsNothing);
      expect(settings.hasCertificateFor(home), isFalse);
      expect(settings.trustCertificate(home, 'bb22'), isTrue);
    });
  }
}
