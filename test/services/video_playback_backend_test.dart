import 'package:flutter/foundation.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:quark/services/video_playback_backend.dart';

void main() {
  group('installVideoPlaybackBackend', () {
    test('initializes media_kit on Linux', () {
      var calls = 0;
      installVideoPlaybackBackend(
        isWeb: false,
        platform: TargetPlatform.linux,
        initialize: () => calls++,
      );
      expect(calls, 1);
    });

    for (final platform in TargetPlatform.values.where(
      (p) => p != TargetPlatform.linux,
    )) {
      test('leaves ${platform.name} on its own backend', () {
        var calls = 0;
        installVideoPlaybackBackend(
          isWeb: false,
          platform: platform,
          initialize: () => calls++,
        );
        expect(calls, 0);
      });
    }

    test('leaves the web on its own backend, whatever the host OS', () {
      var calls = 0;
      installVideoPlaybackBackend(
        isWeb: true,
        platform: TargetPlatform.linux,
        initialize: () => calls++,
      );
      expect(calls, 0);
    });

    test('a libmpv that fails to load does not stop the app launching', () {
      expect(
        () => installVideoPlaybackBackend(
          isWeb: false,
          platform: TargetPlatform.linux,
          initialize: () => throw StateError('libmpv.so.2: not found'),
        ),
        returnsNormally,
      );
    });
  });
}
