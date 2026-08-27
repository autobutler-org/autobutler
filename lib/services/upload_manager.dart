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
    required this.completed,
    required this.failed,
    this.cancelled = false,
    this.stoppedEarly = false,
    this.firstError,
    this.note,
  });

  /// Files enqueued.
  final int total;

  /// Files actually attempted. Below [total] when the run was cancelled or
  /// gave up.
  final int completed;

  final int failed;

  /// The user asked for it to stop.
  final bool cancelled;

  /// It stopped itself after [kMaxConsecutiveUploadFailures] in a row.
  final bool stoppedEarly;

  /// The first failure's message, so the report can say what went wrong rather
  /// than only how much did.
  final String? firstError;

  /// Something the user should know once it is over — that a cap was hit, say.
  /// Reported at the end rather than as a prompt beforehand: it is information,
  /// not a decision, and nothing should stand between "upload" and uploading.
  final String? note;

  int get succeeded => completed - failed;

  /// Enqueued but never attempted, because the run stopped first.
  int get skipped => total - completed;

  bool get hadFailures => failed > 0;

  /// True when the run did not get through everything it was given.
  bool get endedEarly => cancelled || stoppedEarly;
}

/// How long one attempt at one file may take before it is treated as failed.
///
/// Nothing here can abort the request itself, so this frees the worker rather
/// than the connection. Without it a server that accepts a request and then
/// stops responding — an overloaded Raspberry Pi will — holds a worker
/// forever, and with every worker held the batch never ends and the UI never
/// unlocks. Generous, because a large file over a slow link is not a fault.
const Duration kUploadAttemptTimeout = Duration(minutes: 2);

/// How many times one file is attempted before it counts as failed.
///
/// A small server under load fails intermittently rather than permanently, and
/// the second attempt usually lands.
const int kMaxUploadAttempts = 3;

/// How many files may fail in a row before the batch gives up.
///
/// Past a handful the server is not having a bad moment, it is unwell, and
/// grinding through another two thousand files to prove it wastes the user's
/// time and buries the error.
const int kMaxConsecutiveUploadFailures = 5;

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
  UploadManager._()
    : _concurrency = kDefaultUploadConcurrency,
      _attemptTimeout = kUploadAttemptTimeout,
      _maxAttempts = kMaxUploadAttempts,
      _maxConsecutiveFailures = kMaxConsecutiveUploadFailures;

  static final UploadManager instance = UploadManager._();

  /// For tests: an isolated manager, so one test's queue is not another's.
  @visibleForTesting
  UploadManager.forTesting({
    UploadSender? sender,
    int concurrency = kDefaultUploadConcurrency,
    Duration attemptTimeout = kUploadAttemptTimeout,
    int maxAttempts = kMaxUploadAttempts,
    int maxConsecutiveFailures = kMaxConsecutiveUploadFailures,
  }) : _sender = sender,
       _concurrency = concurrency,
       _attemptTimeout = attemptTimeout,
       _maxAttempts = maxAttempts,
       _maxConsecutiveFailures = maxConsecutiveFailures;

  UploadSender? _sender;

  late final TaskPool<_QueuedUpload> _pool = TaskPool<_QueuedUpload>(
    concurrency: _concurrency,
    worker: _upload,
  );
  final int _concurrency;
  final Duration _attemptTimeout;
  final int _maxAttempts;
  final int _maxConsecutiveFailures;

  final StreamController<UploadBatchResult> _results =
      StreamController<UploadBatchResult>.broadcast();

  bool _running = false;
  int _total = 0;
  int _completed = 0;
  int _failed = 0;
  int _consecutiveFailures = 0;
  bool _cancelled = false;
  bool _stoppedEarly = false;
  String? _firstError;
  String? _note;

  /// Emits once each time the queue drains.
  Stream<UploadBatchResult> get results => _results.stream;

  bool get isUploading => _running;

  /// Files in the current run, including any queued while it was already going.
  int get total => _total;

  int get completed => _completed;

  /// Drops everything not yet started and lets the run finish.
  ///
  /// Files already in flight are not interrupted — nothing here can abort an
  /// in-flight request — so the run ends once they return or time out, which
  /// [kUploadAttemptTimeout] bounds.
  void cancel() {
    if (!_running) {
      return;
    }
    _cancelled = true;
    _pool.clear();
    notifyListeners();
  }

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
        completed: _completed,
        failed: _failed,
        cancelled: _cancelled,
        stoppedEarly: _stoppedEarly,
        firstError: _firstError,
        note: _note,
      );
      _running = false;
      _total = 0;
      _completed = 0;
      _failed = 0;
      _consecutiveFailures = 0;
      _cancelled = false;
      _stoppedEarly = false;
      _firstError = null;
      _note = null;
      setUploadUnloadGuard(active: false);
      notifyListeners();
      if (!_results.isClosed) {
        _results.add(result);
      }
    }
  }

  /// Sends one file, retrying a few times. A failure is counted, never
  /// rethrown — one unreadable file must not take the rest of the folder with
  /// it.
  Future<void> _upload(_QueuedUpload queued) async {
    Object? lastError;

    for (var attempt = 1; attempt <= _maxAttempts; attempt++) {
      try {
        // Rebuilt every attempt: a multipart file's body is a stream that can
        // only be read once, so a retry needs a fresh one. Built here, as it
        // is sent, so only [_concurrency] files are ever live at a time.
        final file = await queued.upload.build();
        if (file == null) {
          _recordFailure(queued.upload.name, 'could not be read');
          return;
        }
        await _send(
          currentPath: queued.targetPath,
          selectedFiles: [file],
          serial: queued.serial,
        ).timeout(
          _attemptTimeout,
          onTimeout: () =>
              throw TimeoutException('no response after $_attemptTimeout'),
        );

        _consecutiveFailures = 0;
        _completed++;
        notifyListeners();
        return;
      } catch (e) {
        lastError = e;
        debugPrint(
          '[upload_manager.dart] ${queued.upload.name} attempt $attempt/'
          '$_maxAttempts failed: $e',
        );
        if (attempt < _maxAttempts && !_cancelled && !_stoppedEarly) {
          // Backing off rather than retrying straight away: an overloaded
          // server needs a moment, and hammering it is what caused this.
          await Future<void>.delayed(Duration(seconds: attempt));
        } else {
          break;
        }
      }
    }

    _recordFailure(queued.upload.name, '$lastError');
  }

  void _recordFailure(String name, String reason) {
    _failed++;
    _completed++;
    _firstError ??= reason;
    _consecutiveFailures++;

    if (_consecutiveFailures >= _maxConsecutiveFailures && !_stoppedEarly) {
      _stoppedEarly = true;
      final dropped = _pool.clear();
      debugPrint(
        '[upload_manager.dart] $_consecutiveFailures uploads failed in a row '
        '($name last); abandoning $dropped still queued',
      );
    }

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
