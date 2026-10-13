import 'package:flutter_test/flutter_test.dart';
import 'package:quark/controllers/video_playback_controller.dart';
import 'package:quark/models/playback_position.dart';

/// Resume position and watched state for one video (#2985), against fake
/// service calls and a clock the test moves.
void main() {
  const total = Duration(seconds: 100);
  Duration at(int seconds) => Duration(seconds: seconds);

  late DateTime now;
  late List<Duration> saved;
  late List<bool> watchedCalls;
  late PlaybackPosition stored;
  late PlaybackPosition saveReply;
  late bool saveFails;

  VideoPlaybackController build() => VideoPlaybackController(
    relPath: 'Movies/clip.mp4',
    serial: 'USB1',
    now: () => now,
    load: (relPath, {serial}) async => stored,
    save: (relPath, {serial, required position, required duration}) async {
      saved.add(position);
      if (saveFails) throw Exception('offline');
      return saveReply;
    },
    setWatched: (relPath, {serial, required watched}) async {
      watchedCalls.add(watched);
      return PlaybackPosition(watched: watched);
    },
  );

  setUp(() {
    now = DateTime.utc(2026, 10, 11);
    saved = [];
    watchedCalls = [];
    stored = const PlaybackPosition();
    saveReply = const PlaybackPosition();
    saveFails = false;
  });

  test('load returns the stored position to resume at', () async {
    stored = PlaybackPosition(position: at(42), duration: total);
    final controller = build();

    expect(await controller.load(), at(42));
    expect(controller.watched, isFalse);
  });

  test('load returns null when nothing is stored', () async {
    expect(await build().load(), isNull);
  });

  test('load returns null for a watched video', () async {
    stored = PlaybackPosition(position: at(42), duration: total, watched: true);
    final controller = build();

    expect(await controller.load(), isNull);
    expect(controller.watched, isTrue);
  });

  test('load returns null when the request fails', () async {
    final controller = VideoPlaybackController(
      relPath: 'a.mp4',
      load: (relPath, {serial}) async => throw Exception('offline'),
    );

    expect(await controller.load(), isNull);
  });

  test('a near-end save the Quark answers as watched flips watched', () async {
    saveReply = PlaybackPosition(duration: total, watched: true);
    final controller = build();
    var notified = 0;
    controller.addListener(() => notified++);

    await controller.report(at(97), total, isPlaying: true);
    await controller.report(at(97), total, isPlaying: false);

    expect(saved, [at(97)]);
    expect(controller.watched, isTrue);
    expect(notified, 1);
  });

  test('toggleWatched from watched marks unwatched and clears', () async {
    stored = PlaybackPosition(duration: total, watched: true);
    final controller = build();
    await controller.load();

    await controller.toggleWatched();

    expect(watchedCalls, [false]);
    expect(controller.watched, isFalse);

    await controller.toggleWatched();
    expect(watchedCalls, [false, true]);
    expect(controller.watched, isTrue);
  });

  test('marking unwatched saves the current position again', () async {
    final controller = build();
    await controller.report(at(30), total, isPlaying: true);
    await controller.report(at(30), total, isPlaying: false);
    await controller.toggleWatched();
    await controller.toggleWatched();

    await controller.flush();

    expect(saved, [at(30), at(30)]);
  });

  test('toggleWatched rethrows a failure and leaves watched alone', () async {
    final controller = VideoPlaybackController(
      relPath: 'a.mp4',
      setWatched: (relPath, {serial, required watched}) async =>
          throw Exception('offline'),
    );

    await expectLater(controller.toggleWatched(), throwsException);
    expect(controller.watched, isFalse);
  });

  test('saves are throttled while playing and a pause saves at once', () async {
    final controller = build();

    await controller.report(at(1), total, isPlaying: true);
    now = now.add(const Duration(seconds: 9));
    await controller.report(at(9), total, isPlaying: true);
    expect(saved, isEmpty);

    now = now.add(const Duration(seconds: 1));
    await controller.report(at(10), total, isPlaying: true);
    expect(saved, [at(10)]);

    now = now.add(const Duration(seconds: 3));
    await controller.report(at(13), total, isPlaying: true);
    expect(saved, [at(10)]);

    await controller.report(at(13), total, isPlaying: false);
    expect(saved, [at(10), at(13)]);

    // Still paused at the same place: nothing new to save.
    now = now.add(const Duration(minutes: 1));
    await controller.report(at(13), total, isPlaying: false);
    await controller.flush();
    expect(saved, [at(10), at(13)]);
  });

  test('flush saves a position reached by seeking while paused', () async {
    final controller = build();
    await controller.report(at(50), total, isPlaying: false);

    await controller.flush();

    expect(saved, [at(50)]);
  });

  test('nothing is saved while the duration is unknown', () async {
    final controller = build();

    await controller.report(at(5), Duration.zero, isPlaying: true);
    await controller.report(at(5), Duration.zero, isPlaying: false);
    await controller.flush();

    expect(saved, isEmpty);
  });

  test('a failing save does not throw', () async {
    saveFails = true;
    final controller = build();

    await controller.report(at(5), total, isPlaying: true);
    await controller.report(at(5), total, isPlaying: false);

    expect(saved, [at(5)]);
    expect(controller.watched, isFalse);
  });

  test('a reply that lands after dispose is dropped', () async {
    saveReply = PlaybackPosition(duration: total, watched: true);
    final controller = build();
    await controller.report(at(97), total, isPlaying: false);

    final flushed = controller.flush();
    controller.dispose();

    await flushed;
    expect(saved, [at(97)]);
  });
}
