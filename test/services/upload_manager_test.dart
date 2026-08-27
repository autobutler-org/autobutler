import 'dart:async';
import 'dart:typed_data';

import 'package:flutter_test/flutter_test.dart';
import 'package:http/http.dart' as http;
import 'package:quark/services/upload_manager.dart';
import 'package:quark/utils/upload_tree_utils.dart';

/// An upload must outlive the folder it was started from.
///
/// It used to run inside the file browser's State, so navigating away took the
/// progress with it, and every uploaded file published a server event that the
/// page turned into a full refresh — two extra requests per file, competing
/// with the uploads for the few connections a browser allows per host.
void main() {
  PendingUpload upload(String name, {String relativeDir = ''}) {
    return PendingUpload(
      relativeDir: relativeDir,
      name: name,
      build: () async =>
          http.MultipartFile.fromBytes('files', _bytes, filename: name),
    );
  }

  test('sends every queued file to its own directory', () async {
    final sent = <String>[];
    final manager = UploadManager.forTesting(
      sender: ({required currentPath, required selectedFiles, serial}) async {
        sent.add('$currentPath/${selectedFiles.single.filename}');
      },
    );

    final done = manager.results.first;
    manager.enqueue(
      uploads: [
        upload('a.jpg', relativeDir: 'photos/2024'),
        upload('cover.png', relativeDir: 'photos'),
      ],
      uploadPath: '/vacation',
    );

    await done;

    expect(sent, ['/vacation/photos/2024/a.jpg', '/vacation/photos/cover.png']);
  });

  test('keeps running after every listener has gone away', () async {
    // The navigation case: the page that started the upload is disposed and
    // stops listening. The upload is not the page's to cancel.
    final completers = <Completer<void>>[];
    final manager = UploadManager.forTesting(
      sender: ({required currentPath, required selectedFiles, serial}) {
        final completer = Completer<void>();
        completers.add(completer);
        return completer.future;
      },
    );

    var notifications = 0;
    void listener() => notifications++;
    manager.addListener(listener);

    final done = manager.results.first;
    manager.enqueue(
      uploads: [upload('a.txt'), upload('b.txt'), upload('c.txt')],
      uploadPath: '/docs',
    );

    await pumpEventQueue();
    completers.first.complete();
    await pumpEventQueue();

    // Page disposed mid-upload.
    manager.removeListener(listener);
    final seenBeforeLeaving = notifications;

    // Drain the rest with nobody watching. The sender appends to `completers`
    // as each new file starts, so take a snapshot each pass rather than
    // iterating a list that is still growing.
    while (completers.length < 3 || completers.any((c) => !c.isCompleted)) {
      for (final completer in List.of(completers)) {
        if (!completer.isCompleted) completer.complete();
      }
      await pumpEventQueue();
    }

    final result = await done;

    expect(result.total, 3, reason: 'all three files were sent');
    expect(result.failed, 0);
    expect(completers, hasLength(3));
    expect(
      notifications,
      seenBeforeLeaving,
      reason: 'a detached page stops hearing, it does not stop the upload',
    );
  });

  test('reports progress as files complete', () async {
    final seen = <int>[];
    final manager = UploadManager.forTesting(
      sender: ({required currentPath, required selectedFiles, serial}) async {},
    );
    manager.addListener(() {
      if (manager.isUploading) seen.add(manager.completed);
    });

    final done = manager.results.first;
    manager.enqueue(
      uploads: [upload('a.txt'), upload('b.txt')],
      uploadPath: '',
    );
    await done;

    expect(seen, containsAllInOrder([1, 2]));
  });

  test('extends the run instead of starting a second one', () async {
    final completers = <Completer<void>>[];
    final manager = UploadManager.forTesting(
      sender: ({required currentPath, required selectedFiles, serial}) {
        final completer = Completer<void>();
        completers.add(completer);
        return completer.future;
      },
    );

    final done = manager.results.first;
    manager.enqueue(uploads: [upload('a.txt')], uploadPath: '');
    await pumpEventQueue();
    manager.enqueue(uploads: [upload('b.txt')], uploadPath: '');

    expect(manager.total, 2, reason: 'one run covering both enqueues');

    while (completers.length < 2) {
      for (final c in completers) {
        if (!c.isCompleted) c.complete();
      }
      await pumpEventQueue();
    }
    for (final c in completers) {
      if (!c.isCompleted) c.complete();
    }

    final result = await done;
    expect(result.total, 2);
    expect(manager.isUploading, isFalse);
  });

  test(
    'a file that fails does not take the rest of the batch with it',
    () async {
      var call = 0;
      final manager = UploadManager.forTesting(
        sender: ({required currentPath, required selectedFiles, serial}) async {
          call++;
          if (call == 1) throw Exception('network down');
        },
      );

      final done = manager.results.first;
      manager.enqueue(
        uploads: [upload('a.txt'), upload('b.txt'), upload('c.txt')],
        uploadPath: '',
      );
      final result = await done;

      expect(call, 3);
      expect(result.failed, 1);
      expect(result.succeeded, 2);
    },
  );

  test('carries a cap note through to the end', () async {
    final manager = UploadManager.forTesting(
      sender: ({required currentPath, required selectedFiles, serial}) async {},
    );

    final done = manager.results.first;
    manager.enqueue(
      uploads: [upload('a.txt')],
      uploadPath: '',
      note: 'Stopped at the first 2000 files',
    );
    final result = await done;

    // Reported once it is over, never as a prompt beforehand.
    expect(result.note, 'Stopped at the first 2000 files');
  });

  group('concurrency', () {
    test('sends several at once instead of one at a time', () async {
      var inFlight = 0;
      var peak = 0;
      final gate = Completer<void>();
      final manager = UploadManager.forTesting(
        concurrency: 3,
        sender: ({required currentPath, required selectedFiles, serial}) async {
          inFlight++;
          peak = peak > inFlight ? peak : inFlight;
          await gate.future;
          inFlight--;
        },
      );

      final done = manager.results.first;
      manager.enqueue(
        uploads: List.generate(9, (i) => upload('f$i.txt')),
        uploadPath: '',
      );

      await pumpEventQueue();
      expect(peak, 3, reason: 'the whole pool is working, not one worker');

      gate.complete();
      final result = await done;
      expect(result.total, 9);
      expect(result.failed, 0);
    });

    test('never exceeds the pool size', () async {
      var inFlight = 0;
      var peak = 0;
      final gates = <Completer<void>>[];
      final manager = UploadManager.forTesting(
        concurrency: 2,
        sender: ({required currentPath, required selectedFiles, serial}) async {
          inFlight++;
          peak = peak > inFlight ? peak : inFlight;
          final gate = Completer<void>();
          gates.add(gate);
          await gate.future;
          inFlight--;
        },
      );

      final done = manager.results.first;
      manager.enqueue(
        uploads: List.generate(8, (i) => upload('f$i.txt')),
        uploadPath: '',
      );

      // Release them a few at a time; the cap must hold throughout, not just
      // at the start.
      while (peak < 2 || gates.any((g) => !g.isCompleted)) {
        for (final gate in List.of(gates)) {
          if (!gate.isCompleted) gate.complete();
        }
        await pumpEventQueue();
      }

      final result = await done;
      expect(result.total, 8);
      expect(peak, lessThanOrEqualTo(2), reason: 'bounded, not unbounded');
      expect(gates, hasLength(8), reason: 'every file was still sent');
    });

    test('one slow file does not hold the pool behind it', () async {
      // The reason for a shared queue rather than fixed batches: a batch runs
      // at the speed of its slowest member.
      final slow = Completer<void>();
      final finished = <String>[];
      final manager = UploadManager.forTesting(
        concurrency: 2,
        sender: ({required currentPath, required selectedFiles, serial}) async {
          final name = selectedFiles.single.filename!;
          if (name == 'slow.txt') {
            await slow.future;
          }
          finished.add(name);
        },
      );

      final done = manager.results.first;
      manager.enqueue(
        uploads: [
          upload('slow.txt'),
          upload('a.txt'),
          upload('b.txt'),
          upload('c.txt'),
        ],
        uploadPath: '',
      );

      await pumpEventQueue();
      expect(
        finished,
        containsAll(['a.txt', 'b.txt', 'c.txt']),
        reason: 'the other worker kept going past the stuck file',
      );

      slow.complete();
      final result = await done;
      expect(result.total, 4);
      expect(result.failed, 0);
    });

    test('a failure in one worker does not stop the others', () async {
      final manager = UploadManager.forTesting(
        concurrency: 3,
        sender: ({required currentPath, required selectedFiles, serial}) async {
          if (selectedFiles.single.filename == 'bad.txt') {
            throw Exception('network down');
          }
        },
      );

      final done = manager.results.first;
      manager.enqueue(
        uploads: [
          upload('a.txt'),
          upload('bad.txt'),
          upload('b.txt'),
          upload('c.txt'),
        ],
        uploadPath: '',
      );
      final result = await done;

      expect(result.total, 4);
      expect(result.failed, 1);
      expect(result.succeeded, 3);
    });
  });
}

final _bytes = Uint8List.fromList([1, 2, 3]);
