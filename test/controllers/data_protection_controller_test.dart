import 'dart:async';

import 'package:flutter_test/flutter_test.dart';
import 'package:quark/controllers/data_protection_controller.dart';
import 'package:quark/services/data_protection_service.dart';
import 'package:quark/services/events_service.dart';
import 'package:quark/utils/error_text.dart';
import 'package:quark_widgets/quark_widgets.dart';

DataProtectionStatus _status({
  String level = 'protected',
  String sentence =
      'Your files are on 3 drives. Keep one backup drive somewhere else.',
  DataProtectionAction action = DataProtectionAction.keepElsewhere,
  List<DataProtectionDriveInfo> drives = const [],
}) {
  return DataProtectionStatus(
    level: level,
    sentence: sentence,
    action: action,
    drives: drives,
  );
}

/// The protection card's controller. The level is the one the fake Quark
/// reported. A drive list that disagrees does not change it.
void main() {
  test('loads the reported level and does not recount the drives', () async {
    final controller = DataProtectionController(
      fetch: () async => _status(
        drives: const [
          DataProtectionDriveInfo(
            serial: 'usb-1',
            name: 'Backup',
            counts: false,
            reason: 'No completed backup yet.',
          ),
        ],
      ),
      events: const Stream.empty(),
    );
    addTearDown(controller.dispose);

    await controller.load();

    expect(controller.level, DataProtectionLevel.protected);
    expect(controller.drives, hasLength(1));
    expect(controller.drives.single.reason, 'No completed backup yet.');
    expect(controller.error, isNull);
    expect(controller.isLoading, isFalse);
  });

  test('a failed first load keeps the card empty', () async {
    final controller = DataProtectionController(
      fetch: () async => throw const ApiException(500, 'check the backup'),
      events: const Stream.empty(),
    );
    addTearDown(controller.dispose);

    await controller.load();

    expect(controller.status, isNull);
    expect(controller.level, isNull);
    expect(controller.error, isA<ApiException>());
    expect(controller.isLoading, isFalse);
  });

  test('a failed refresh keeps the level already shown', () async {
    var fail = false;
    final controller = DataProtectionController(
      fetch: () async {
        if (fail) throw const ApiException(500, 'check the backup');
        return _status(
          level: 'basic',
          action: DataProtectionAction.startBackup,
        );
      },
      events: const Stream.empty(),
    );
    addTearDown(controller.dispose);

    await controller.load();
    fail = true;
    await controller.load();

    expect(controller.level, DataProtectionLevel.basic);
    expect(controller.error, isA<ApiException>());
    expect(controller.isLoading, isFalse);
  });

  test('a slow first load does not overwrite a newer one', () async {
    final first = Completer<DataProtectionStatus>();
    final second = Completer<DataProtectionStatus>();
    var calls = 0;
    final controller = DataProtectionController(
      fetch: () {
        calls++;
        return calls == 1 ? first.future : second.future;
      },
      events: const Stream.empty(),
    );
    addTearDown(controller.dispose);

    final older = controller.load();
    final newer = controller.load();
    second.complete(_status(level: 'protected'));
    await newer;
    first.complete(_status(level: 'unprotected'));
    await older;

    expect(controller.level, DataProtectionLevel.protected);
  });

  test('backup_completed loads again and backup_progress does not', () async {
    var calls = 0;
    final events = StreamController<FileEvent>.broadcast(sync: true);
    addTearDown(events.close);
    final controller = DataProtectionController(
      fetch: () async {
        calls++;
        return _status(level: calls == 1 ? 'basic' : 'protected');
      },
      events: events.stream,
    );
    addTearDown(controller.dispose);

    await controller.load();
    events.add(const FileEvent(kind: 'backup_progress', path: ''));
    await Future<void>.delayed(Duration.zero);
    expect(calls, 1);

    events.add(const FileEvent(kind: 'backup_completed', path: ''));
    await Future<void>.delayed(Duration.zero);
    expect(calls, 2);
    expect(controller.level, DataProtectionLevel.protected);

    events.add(const FileEvent(kind: 'backup_failed', path: ''));
    await Future<void>.delayed(Duration.zero);
    expect(calls, 3);
  });

  test('dispose during an in-flight load does not throw', () async {
    final gate = Completer<DataProtectionStatus>();
    final controller = DataProtectionController(
      fetch: () => gate.future,
      events: const Stream.empty(),
    );

    final pending = controller.load();
    controller.dispose();
    gate.complete(_status());
    await pending;
  });

  test('fromJson reads the protection body and rejects an unknown level', () {
    final status = DataProtectionStatus.fromJson({
      'level': 'basic',
      'sentence':
          'Your files are on 2 drives. Add a second backup drive and keep it somewhere else.',
      'nextStep': 'start-backup',
      'drives': [
        {
          'serial': 'usb-1',
          'name': 'Backup',
          'counts': true,
          'reason': '',
          'lastBackedUp': '2026-10-01T15:04:05Z',
        },
        {
          'serial': 'usb-2',
          'name': 'Second',
          'counts': false,
          'reason': 'No completed backup yet.',
          'lastBackedUp': null,
        },
      ],
    });

    expect(status.level, 'basic');
    expect(status.action, DataProtectionAction.startBackup);
    expect(status.drives.first.reason, isNull);
    expect(
      status.drives.first.lastBackedUp,
      DateTime.utc(2026, 10, 1, 15, 4, 5),
    );
    expect(status.backupTargetSerial, 'usb-2');

    expect(
      () => DataProtectionStatus.fromJson({
        'level': 'superb',
        'sentence': 'nope',
        'nextStep': 'start-backup',
        'drives': [],
      }),
      throwsA(
        isA<FormatException>().having(
          (error) => error.message,
          'message',
          'Invalid protection status',
        ),
      ),
    );
  });

  test('backupTargetSerial skips a drive a backup cannot make count', () {
    final status = DataProtectionStatus(
      level: 'basic',
      sentence: 'Your files are on 2 drives.',
      action: DataProtectionAction.startBackup,
      drives: const [
        DataProtectionDriveInfo(
          serial: 'same',
          name: 'Partition',
          counts: false,
          reason: DataProtectionStatus.reasonSameDisk,
        ),
        DataProtectionDriveInfo(
          serial: 'stale',
          name: 'Office',
          counts: false,
          reason: DataProtectionStatus.reasonStale,
        ),
      ],
    );

    expect(status.backupTargetSerial, 'stale');
  });
}
