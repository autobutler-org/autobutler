import 'dart:async';

import 'package:flutter/foundation.dart';
import 'package:http/http.dart' as http;
import 'package:quark/services/file_browser_actions.dart';
import 'package:quark/utils/task_pool.dart';
import 'package:quark/utils/upload_tree_utils.dart';
import 'package:quark/utils/upload_unload_guard.dart';

/// The result of a run of the queue, emitted once it drains.
class UploadBatchResult {
  const UploadBatchResult({
    required this.total,
    required this.failed,
    this.note,
  });

  final int total;
  final int failed;

  /// Something the user should know once it is over — that a cap was hit, say.
  /// Reported at the end rather than as a prompt beforehand: it is information,
  /// not a decision, and nothing should stand between "upload" and uploading.
  final String? note;

  int get succeeded => total - failed;
  bool get hadFailures => failed > 0;
}

/// How many uploads are in flight at once.
///
/// A browser allows about six connections per host on HTTP/1.1, and the file
/// listing and device calls need some of those too — saturating the pool is
/// what made uploads look stalled in the first place. Four keeps the pipe busy
/// while leaving the rest of the app able to talk to the server.
const int kDefaultUploadConcurrency = 4;

/// Sends one file. Swapped out in tests; in the app it is the real upload.
typedef UploadSender =
    Future<void> Function({
      required String currentPath,
      required List<http.MultipartFile> selectedFiles,
      String? serial,
    });

/// Runs uploads independently of whatever is on screen.
///
/// Uploads used to live in the file browser's State, which tied them to the
/// page: navigating to another folder took the progress with it, and every
/// completed file published a server event that the page turned into a full
/// refresh — devices call plus listing call — so a folder upload fired two
/// extra requests per file, all competing with the uploads themselves for the
/// browser's handful of connections per host. The upload appeared to stall.
///
/// Owning the queue here is what makes an upload survive navigation and a
/// refresh: nothing in the widget tree can start, stop, or outlive it. A page
/// reload still ends an upload, because the bytes are read and sent by this
/// page — see [setUploadUnloadGuard], which is the one case that warns.
class UploadManager extends ChangeNotifier {
  UploadManager._() : _concurrency = kDefaultUploadConcurrency;

  static final UploadManager instance = UploadManager._();

  /// For tests: an isolated manager, so one test's queue is not another's.
  @visibleForTesting
  UploadManager.forTesting({
    UploadSender? sender,
    int concurrency = kDefaultUploadConcurrency,
  }) : _sender = sender,
       _concurrency = concurrency;

  UploadSender? _sender;

  late final TaskPool<_QueuedUpload> _pool = TaskPool<_QueuedUpload>(
    concurrency: _concurrency,
    worker: _upload,
  );
  final int _concurrency;

  final StreamController<UploadBatchResult> _results =
      StreamController<UploadBatchResult>.broadcast();

  bool _running = false;
  int _total = 0;
  int _completed = 0;
  int _failed = 0;
  String? _note;

  /// Emits once each time the queue drains.
  Stream<UploadBatchResult> get results => _results.stream;

  bool get isUploading => _running;

  /// Files in the current run, including any queued while it was already going.
  int get total => _total;

  int get completed => _completed;

  /// Adds [uploads] to the queue and starts draining if nothing is running.
  ///
  /// Returns immediately: the caller is a button handler, not the owner of the
  /// work. Enqueuing during a run extends it rather than starting a second one,
  /// so uploads never compete with each other for connections.
  void enqueue({
    required List<PendingUpload> uploads,
    required String uploadPath,
    String? serial,
    String? note,
  }) {
    if (uploads.isEmpty) {
      return;
    }
    if (note != null && note.isNotEmpty) {
      _note = note;
    }

    // Grouped so every file bound for one directory is sent together, and the
    // target path is resolved once per directory rather than once per file.
    for (final group in groupByRelativeDir(uploads).entries) {
      final targetPath = uploadTargetPath(uploadPath, group.key);
      for (final upload in group.value) {
        _pool.add(_QueuedUpload(upload, targetPath, serial));
      }
    }

    _total += uploads.length;
    notifyListeners();

    if (!_running) {
      unawaited(_drain());
    }
  }

  Future<void> _drain() async {
    _running = true;
    setUploadUnloadGuard(active: true);
    notifyListeners();

    try {
      // TaskPool handles the fan-out and picks up anything enqueued mid-run.
      // Failures are counted in _upload rather than thrown, so this does not
      // need to guard against one file taking the batch down.
      await _pool.drain();
    } finally {
      final result = UploadBatchResult(
        total: _total,
        failed: _failed,
        note: _note,
      );
      _running = false;
      _total = 0;
      _completed = 0;
      _failed = 0;
      _note = null;
      setUploadUnloadGuard(active: false);
      notifyListeners();
      if (!_results.isClosed) {
        _results.add(result);
      }
    }
  }

  /// Sends one file. A failure is counted, never rethrown — one unreadable
  /// file must not take the rest of the folder with it.
  Future<void> _upload(_QueuedUpload queued) async {
    try {
      // Built here, as it is sent: a folder upload cannot hold every file's
      // bytes at once, and only [_concurrency] of them are ever live.
      final file = await queued.upload.build();
      if (file == null) {
        _failed++;
      } else {
        await _send(
          currentPath: queued.targetPath,
          selectedFiles: [file],
          serial: queued.serial,
        );
      }
    } catch (e) {
      _failed++;
      debugPrint(
        '[upload_manager.dart] Failed to upload ${queued.upload.name}: $e',
      );
    }
    _completed++;
    notifyListeners();
  }

  Future<void> _send({
    required String currentPath,
    required List<http.MultipartFile> selectedFiles,
    String? serial,
  }) {
    final sender = _sender ?? uploadMultipartFilesToCurrentPath;
    return sender(
      currentPath: currentPath,
      selectedFiles: selectedFiles,
      serial: serial,
    );
  }

  @override
  void dispose() {
    _results.close();
    super.dispose();
  }
}

class _QueuedUpload {
  const _QueuedUpload(this.upload, this.targetPath, this.serial);

  final PendingUpload upload;
  final String targetPath;
  final String? serial;
}
