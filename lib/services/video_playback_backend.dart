import 'package:flutter/foundation.dart';
import 'package:quark/utils/app_log.dart';
import 'package:video_player_media_kit/video_player_media_kit.dart';

/// Makes `video_player` work on Linux, which has no implementation of its own
/// (#3151): there, media_kit's libmpv-backed player is registered as the
/// `VideoPlayerPlatform`, so every `VideoPlayerController` in the app plays
/// through libmpv unchanged. Everywhere else this does nothing, and Android,
/// iOS and web keep the backends `video_player` endorses.
///
/// [isWeb], [platform] and [initialize] exist for tests; the defaults are the
/// running platform and the real media_kit setup. A libmpv that fails to load
/// is logged rather than thrown, so the app still launches without playback.
void installVideoPlaybackBackend({
  bool isWeb = kIsWeb,
  TargetPlatform? platform,
  void Function() initialize = _initializeMediaKit,
}) {
  if (isWeb || (platform ?? defaultTargetPlatform) != TargetPlatform.linux) {
    return;
  }
  try {
    initialize();
  } catch (e, stack) {
    AppLog.instance.error('Video playback is unavailable', e, stack);
  }
}

void _initializeMediaKit() =>
    VideoPlayerMediaKit.ensureInitialized(linux: true);
