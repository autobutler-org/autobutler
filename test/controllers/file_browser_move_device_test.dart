import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:quark/controllers/file_browser_controller.dart';
import 'package:quark/models/file_node.dart';
import 'package:quark/services/authenticated_service.dart';
import 'package:quark/services/storage_service.dart';
import 'package:quark/widgets/file_browser/file_browser_view.dart';

/// #3193: the Move / Rename dialog could not reliably move a file to another
/// storage device. Each test drives the Files page's move action through the
/// real dialog and reads the `PUT /api/v0/files` body it sends, if any.
void main() {
  Map<String, Object?> device(
    String name, {
    String serial = '',
    bool isEnabled = true,
  }) => {
    'name': name,
    'devicePath': '/dev/$name',
    'mountPoint': '/mnt/$name',
    'isInternal': serial.isEmpty,
    'isEnabled': isEnabled,
    if (serial.isNotEmpty) 'usbInfo': {'serial': serial},
  };

  FileNode node(String path, {String serial = '', bool isDir = false}) =>
      FileNode(
        name: path.split('/').last,
        size: 0,
        isDir: isDir,
        deviceName: '',
        devicePath: '',
        deviceSerial: serial,
        dirPath: path,
      );

  late List<Map<String, Object?>> devices;
  late List<Map<String, dynamic>> moves;

  setUp(() {
    devices = [device('Built-in'), device('Stick', serial: 'USB1')];
    moves = [];
    StorageService.invalidateDeviceCache();
    sharedHttpClientFactory = () => MockClient((request) async {
      if (request.url.path.endsWith('/storage/devices/status')) {
        return http.Response(jsonEncode({'devices': devices}), 200);
      }
      if (request.method == 'PUT') {
        moves.add(jsonDecode(request.body) as Map<String, dynamic>);
        return http.Response('{}', 202);
      }
      return http.Response('[]', 200);
    });
  });

  tearDown(() {
    StorageService.invalidateDeviceCache();
    resetSharedHttpClient();
    sharedHttpClientFactory = buildLocalTrustHttpClient;
  });

  /// Opens the move dialog for [moved] and returns the action's outcome once
  /// the dialog closes.
  Future<Future<FileMenuActionOutcome?>> open(
    WidgetTester tester,
    FileNode moved,
  ) async {
    tester.view.physicalSize = const Size(1280, 1600);
    tester.view.devicePixelRatio = 1.0;
    addTearDown(tester.view.reset);
    late Future<FileMenuActionOutcome?> outcome;
    await tester.pumpWidget(
      MaterialApp(
        home: Builder(
          builder: (context) => ElevatedButton(
            onPressed: () =>
                outcome = const FileBrowserController().handleFileAction(
                  node: moved,
                  action: FileMenuAction.moveRename,
                  context: context,
                ),
            child: const Text('open'),
          ),
        ),
      ),
    );
    await tester.tap(find.text('open'));
    await tester.pumpAndSettle();
    expect(find.text('Move / Rename'), findsOneWidget);
    return outcome;
  }

  Future<void> pickDevice(WidgetTester tester, String label) async {
    await tester.tap(find.byType(DropdownButtonFormField<StorageDevice>));
    await tester.pumpAndSettle();
    await tester.tap(find.text(label).last);
    await tester.pumpAndSettle();
  }

  Future<void> save(WidgetTester tester) async {
    await tester.tap(find.text('Save'));
    await tester.pumpAndSettle();
  }

  testWidgets('the same path on another drive is a move', (tester) async {
    final outcome = await open(tester, node('Docs/a.txt'));
    await pickDevice(tester, 'Stick');
    await save(tester);

    expect((await outcome)?.shouldRefresh, isTrue);
    expect(moves, [
      {
        'oldFilePath': '/Docs/a.txt',
        'newFilePath': '/Docs/a.txt',
        'newDeviceSerial': 'USB1',
      },
    ]);
  });

  testWidgets('a folder moves to the same path on another drive', (
    tester,
  ) async {
    final outcome = await open(tester, node('Docs', isDir: true));
    await pickDevice(tester, 'Stick');
    await save(tester);

    expect((await outcome)?.shouldRefresh, isTrue);
    expect(moves.single['newDeviceSerial'], 'USB1');
  });

  testWidgets('a file on a USB drive moves to built-in storage', (
    tester,
  ) async {
    final outcome = await open(tester, node('Docs/a.txt', serial: 'USB1'));
    await pickDevice(tester, 'Built-in (Internal)');
    await save(tester);

    expect((await outcome)?.shouldRefresh, isTrue);
    expect(moves, [
      {
        'oldFilePath': '/Docs/a.txt',
        'newFilePath': '/Docs/a.txt',
        'oldDeviceSerial': 'USB1',
      },
    ]);
  });

  testWidgets('a rename stays on the drive the file is on', (tester) async {
    // With built-in storage listed first the old default happened to mean
    // "same drive"; two USB drives show the dropdown's real starting point.
    devices = [device('Card', serial: 'USB0'), device('Stick', serial: 'USB1')];
    final outcome = await open(tester, node('Docs/a.txt', serial: 'USB1'));
    await tester.enterText(find.byType(TextField).last, 'b.txt');
    await tester.pump();
    await save(tester);

    expect((await outcome)?.shouldRefresh, isTrue);
    expect(moves, [
      {
        'oldFilePath': '/Docs/a.txt',
        'newFilePath': '/Docs/b.txt',
        'oldDeviceSerial': 'USB1',
        'newDeviceSerial': 'USB1',
      },
    ]);
  });

  testWidgets('nothing is sent when neither path nor drive changes', (
    tester,
  ) async {
    final outcome = await open(tester, node('Docs/a.txt', serial: 'USB1'));
    await save(tester);

    expect(await outcome, isNull);
    expect(moves, isEmpty);
  });

  testWidgets('a drive that is not mounted is not offered', (tester) async {
    devices.add(device('Unplugged', serial: 'USB2', isEnabled: false));
    await open(tester, node('Docs/a.txt'));
    await tester.tap(find.byType(DropdownButtonFormField<StorageDevice>));
    await tester.pumpAndSettle();

    expect(find.text('Stick'), findsWidgets);
    expect(find.text('Unplugged'), findsNothing);
  });

  testWidgets("the file's own drive is offered even when not mounted", (
    tester,
  ) async {
    devices.add(device('Unplugged', serial: 'USB2', isEnabled: false));
    await open(tester, node('Docs/a.txt', serial: 'USB2'));

    expect(find.text('Unplugged'), findsOneWidget);
  });
}
