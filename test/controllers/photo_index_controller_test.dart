import 'dart:async';

import 'package:flutter_test/flutter_test.dart';
import 'package:quark/controllers/photo_index_controller.dart';
import 'package:quark/services/events_service.dart';
import 'package:quark/services/photo_index_service.dart';
import 'package:quark/services/storage_service.dart';
import 'package:quark/utils/error_text.dart';

PhotoIndexStatus _status({
  int hashed = 1200,
  bool running = true,
  bool paused = false,
  List<PhotoIndexFailure> failures = const [],
}) => PhotoIndexStatus(
  total: 5000,
  hashed: hashed,
  datesRead: 1100,
  failed: failures.length,
  failures: failures,
  running: running,
  paused: paused,
);

/// The photo library index status on the duplicates page (#3556).
void main() {
  test('loads the status and follows photo_index_changed', () async {
    final events = StreamController<FileEvent>.broadcast();
    addTearDown(events.close);
    var fetches = 0;
    final controller = PhotoIndexController(
      fetch: () async {
        fetches++;
        return _status(hashed: fetches * 1000);
      },
      events: events.stream,
    );
    addTearDown(controller.dispose);

    expect(fetches, 0, reason: 'nothing is fetched before start');
    await controller.start();
    expect(controller.status?.hashed, 1000);
    expect(controller.isLoading, isFalse);

    events.add(const FileEvent(kind: 'upload', path: 'a.jpg'));
    await pumpEventQueue();
    expect(fetches, 1, reason: 'other events are ignored');

    events.add(const FileEvent(kind: 'photo_index_changed', path: ''));
    await pumpEventQueue();
    expect(controller.status?.hashed, 2000);

    events.add(const FileEvent(kind: 'resync', path: ''));
    await pumpEventQueue();
    expect(fetches, 3);
  });

  test('start twice listens once', () async {
    final events = StreamController<FileEvent>.broadcast();
    addTearDown(events.close);
    var fetches = 0;
    final controller = PhotoIndexController(
      fetch: () async {
        fetches++;
        return _status();
      },
      events: events.stream,
    );
    addTearDown(controller.dispose);
    await controller.start();
    await controller.start();
    fetches = 0;
    events.add(const FileEvent(kind: 'photo_index_changed', path: ''));
    await pumpEventQueue();
    expect(fetches, 1);
  });

  test('a 403 hides the status rather than showing an error', () async {
    final controller = PhotoIndexController(
      fetch: () async => throw const ApiException(403, 'index'),
      events: const Stream.empty(),
    );
    addTearDown(controller.dispose);
    await controller.start();
    expect(controller.isHidden, isTrue);
    expect(controller.error, isNull);
  });

  test('another failure is kept for the page to word', () async {
    final controller = PhotoIndexController(
      fetch: () async => throw const ApiException(500, 'index'),
      events: const Stream.empty(),
    );
    addTearDown(controller.dispose);
    await controller.start();
    expect(controller.isHidden, isFalse);
    expect(controller.error, isA<ApiException>());
    expect(controller.errorAction, 'load the photo index status');
  });

  test('pause and resume take the status they answer with', () async {
    final calls = <String>[];
    final controller = PhotoIndexController(
      fetch: () async => _status(),
      pause: () async {
        calls.add('pause');
        return _status(paused: true);
      },
      resume: () async {
        calls.add('resume');
        return _status();
      },
      events: const Stream.empty(),
    );
    addTearDown(controller.dispose);
    await controller.start();

    final pausing = controller.pause();
    expect(controller.isChanging, isTrue);
    await pausing;
    expect(controller.isChanging, isFalse);
    expect(controller.status?.paused, isTrue);

    await controller.resume();
    expect(controller.status?.paused, isFalse);
    expect(calls, ['pause', 'resume']);
  });

  test('a failed pause keeps the status and names the action', () async {
    final controller = PhotoIndexController(
      fetch: () async => _status(),
      pause: () async => throw const ApiException(500, 'pause'),
      events: const Stream.empty(),
    );
    addTearDown(controller.dispose);
    await controller.start();
    await controller.pause();
    expect(controller.status, isNotNull);
    expect(controller.error, isA<ApiException>());
    expect(controller.errorAction, 'pause indexing');

    // The next good load clears it.
    await controller.load();
    expect(controller.error, isNull);
  });

  test('maps failures to plain rows, naming drives by name', () async {
    final controller = PhotoIndexController(
      fetch: () async => _status(
        failures: const [
          PhotoIndexFailure(
            serial: '',
            relPath: 'users/ana/photos/2024/broken.jpg',
            reason: PhotoIndexFailureReason.undecodable,
          ),
          PhotoIndexFailure(
            serial: 'usb-1',
            relPath: 'Camera/IMG_1.CR2',
            reason: PhotoIndexFailureReason.missingConverter,
          ),
        ],
      ),
      listDevices: () async => [
        StorageDevice.fromJson({
          'name': 'Backup drive',
          'usbInfo': {'serial': 'usb-1'},
        }),
      ],
      events: const Stream.empty(),
    );
    addTearDown(controller.dispose);
    await controller.start();

    final rows = controller.failures;
    expect(rows.first.id, ':users/ana/photos/2024/broken.jpg');
    expect(rows.first.name, 'broken.jpg');
    expect(rows.first.location, 'Photos › 2024');
    expect(
      rows.first.reason,
      Errors.photoIndexFailure(PhotoIndexFailureReason.undecodable),
    );
    expect(rows.last.location, 'Backup drive · Camera');

    expect(controller.showFailures, isFalse);
    controller.toggleFailures();
    expect(controller.showFailures, isTrue);
  });

  test('a slow load cannot overwrite a newer one', () async {
    final first = Completer<PhotoIndexStatus>();
    var call = 0;
    final controller = PhotoIndexController(
      fetch: () {
        call++;
        return call == 1 ? first.future : Future.value(_status(hashed: 9));
      },
      events: const Stream.empty(),
    );
    addTearDown(controller.dispose);
    final slow = controller.load();
    await controller.load();
    first.complete(_status(hashed: 1));
    await slow;
    expect(controller.status?.hashed, 9);
  });
}
