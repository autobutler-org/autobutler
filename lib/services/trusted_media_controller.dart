import 'dart:io';

import 'package:flutter/foundation.dart';
import 'package:path_provider/path_provider.dart';
import 'package:quark/services/authenticated_service.dart';
import 'package:quark/services/local_trust.dart';
import 'package:video_player/video_player.dart';

/// A [VideoPlayerController] plus the temp file backing it, if any.
///
/// `video_player` plays through the platform's native media stack (AVPlayer
/// on iOS, ExoPlayer on Android), which enforces standard certificate
/// validation and has no hook for [isLocalTrustHost]'s self-signed-cert
/// exception — unlike the app's own `http`/WebSocket clients, which apply it
/// via [buildLocalTrustHttpClient]. Against a butler on the local network,
/// this made every audio/video file fail to play with a certificate-trust
/// error that native platforms report as a generic codec/format failure
/// (#1627).
///
/// [createTrustedMediaController] works around this by downloading the file
/// through the already-trusted client and playing it back from a local temp
/// file instead of streaming it directly. [tempFile] is non-null exactly when
/// this path was taken, so [dispose] knows whether there's a download to
/// clean up.
class TrustedMediaController {
  final VideoPlayerController controller;
  final File? tempFile;

  const TrustedMediaController(this.controller, this.tempFile);

  Future<void> dispose() async {
    await controller.dispose();
    final file = tempFile;
    if (file != null) {
      try {
        await file.delete();
      } catch (_) {
        // Best-effort cleanup; a leftover temp file is harmless.
      }
    }
  }
}

/// Creates a controller for playing the media at [url], routing around a
/// butler's self-signed certificate when needed. See [TrustedMediaController].
///
/// [fileName] supplies the extension for the temp file (some platforms use it
/// to pick a decoder when no explicit format hint is given). [formatHint]
/// carries through to [VideoPlayerController.networkUrl] as before.
///
/// Adaptive-streaming manifests (HLS/DASH/Smooth Streaming) are excluded from
/// the download path: a single downloaded file can't resolve the segment URLs
/// a manifest references, so those still stream directly and remain affected
/// by #1627 against a local butler.
Future<TrustedMediaController> createTrustedMediaController(
  Uri url, {
  required String fileName,
  VideoFormat? formatHint,
}) async {
  final isAdaptiveStreaming =
      formatHint == VideoFormat.hls ||
      formatHint == VideoFormat.dash ||
      formatHint == VideoFormat.ss;

  if (!kIsWeb && !isAdaptiveStreaming && isLocalTrustHost(url.host)) {
    final client = buildLocalTrustHttpClient();
    final List<int> bytes;
    try {
      final response = await client.get(url);
      if (response.statusCode < 200 || response.statusCode >= 300) {
        throw Exception('Failed to fetch media (${response.statusCode})');
      }
      bytes = response.bodyBytes;
    } finally {
      client.close();
    }

    final tempDir = await getTemporaryDirectory();
    final file = File(
      '${tempDir.path}/quark_media_${DateTime.now().microsecondsSinceEpoch}'
      '${_extensionOf(fileName)}',
    );
    await file.writeAsBytes(bytes);
    return TrustedMediaController(VideoPlayerController.file(file), file);
  }

  return TrustedMediaController(
    VideoPlayerController.networkUrl(url, formatHint: formatHint),
    null,
  );
}

String _extensionOf(String name) {
  final idx = name.lastIndexOf('.');
  return idx < 0 ? '' : name.substring(idx);
}
