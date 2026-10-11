import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:quark/services/app_settings.dart';
import 'package:quark/services/authenticated_service.dart';
import 'package:quark/services/storage_service.dart';
import 'package:quark/widgets/storage_devices/credentials_dialog.dart';
import 'package:quark/widgets/storage_devices/snapshot_backup_dialog.dart';
import 'package:quark/widgets/system/storage_tab.dart';
import 'package:shared_preferences/shared_preferences.dart';

/// #3102: the Storage tab's credentials and snapshot backup dialogs own their
/// text controllers, so closing one and letting it animate out never rebuilds
/// a field with a disposed controller.
void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  const secureStorage = MethodChannel(
    'plugins.it_nomads.com/flutter_secure_storage',
  );

  setUpAll(() {
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(secureStorage, (_) async => null);
  });

  tearDownAll(() {
    TestDefaultBinaryMessengerBinding.instance.defaultBinaryMessenger
        .setMockMethodCallHandler(secureStorage, null);
  });

  setUp(() async {
    SharedPreferences.setMockInitialValues({
      'hosts': jsonEncode([
        {'name': 'Home', 'hostAddress': 'http://quark.local'},
      ]),
      'activeHostIndex': 0,
    });
    await AppSettings.instance.load();
    AppSettings.instance.isAdmin.value = true;
    StorageService.invalidateDeviceCache();
    resetSharedHttpClient();
    sharedHttpClientFactory = () => MockClient((request) async {
      if (request.url.path == '/api/v0/storage/devices/status') {
        return http.Response(
          jsonEncode({
            'devices': [
              {
                'name': 'Backups',
                'serial': 'USB2',
                'mountPoint': '/mnt/backups',
                'role': 'snapshot-backup',
                'isEnabled': true,
              },
            ],
          }),
          200,
        );
      }
      return http.Response(jsonEncode({'error': ''}), 404);
    });
  });

  tearDown(() {
    AppSettings.instance.isAdmin.value = false;
    resetSharedHttpClient();
    sharedHttpClientFactory = buildLocalTrustHttpClient;
  });

  /// Pumps the tab and opens the credentials dialog from "Back Up".
  Future<void> openCredentials(WidgetTester tester) async {
    tester.view.physicalSize = const Size(1280, 800);
    tester.view.devicePixelRatio = 1.0;
    addTearDown(tester.view.reset);
    await tester.pumpWidget(
      const MaterialApp(home: Scaffold(body: StorageTab())),
    );
    await tester.pumpAndSettle();
    await tester.tap(find.text('Back Up'));
    await tester.pumpAndSettle();
  }

  /// Taps [label] and pumps the dialog's exit animation frame by frame, which
  /// is where a field rebuilding with a disposed controller would throw.
  Future<void> closeWith(WidgetTester tester, String label) async {
    await tester.tap(find.text(label));
    await tester.pump();
    await tester.pump(const Duration(milliseconds: 50));
    expect(tester.takeException(), isNull);
    await tester.pumpAndSettle();
    expect(tester.takeException(), isNull);
  }

  testWidgets('the credentials dialog closes without a disposed controller', (
    tester,
  ) async {
    await openCredentials(tester);
    await tester.enterText(find.widgetWithText(TextField, 'Username'), 'grace');
    await tester.enterText(find.widgetWithText(TextField, 'Password'), 'pw');

    await closeWith(tester, 'Cancel');

    expect(find.byType(CredentialsDialog), findsNothing);
    expect(find.byType(SnapshotBackupDialog), findsNothing);
  });

  testWidgets('the backup dialog closes without a disposed controller', (
    tester,
  ) async {
    await openCredentials(tester);
    await tester.enterText(find.widgetWithText(TextField, 'Username'), 'grace');
    await tester.enterText(find.widgetWithText(TextField, 'Password'), 'pw');
    await closeWith(tester, 'Continue');
    expect(find.byType(SnapshotBackupDialog), findsOneWidget);
    await tester.enterText(
      find.widgetWithText(TextField, 'Recovery password (optional)'),
      'recovery-pw',
    );

    await closeWith(tester, 'Cancel');

    expect(find.byType(SnapshotBackupDialog), findsNothing);
  });
}
