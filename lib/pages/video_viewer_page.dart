import 'dart:async';
import 'dart:typed_data';

import 'package:autobutler/services/cirrus_service.dart';
import 'package:autobutler/utils/web_download_stub.dart'
    if (dart.library.html) 'package:autobutler/utils/web_download_web.dart'
    as web_download;
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:video_player/video_player.dart';

class VideoViewerPage extends StatefulWidget {
  final Uri url;
  final String name;

  const VideoViewerPage({super.key, required this.url, required this.name});

  @override
  State<VideoViewerPage> createState() => _VideoViewerPageState();
}

class _VideoViewerPageState extends State<VideoViewerPage> {
  VideoPlayerController? _controller;
  bool _loading = true;
  String? _errorMessage;
  bool _isUnsupportedFormat = false;
  bool _downloading = false;

  static const _nonWebNativeExtensions = {
    '.mov',
    '.avi',
    '.mkv',
    '.wmv',
    '.flv',
    '.3gp',
  };

  bool _isNonWebNativeFormat(String name) {
    final lower = name.toLowerCase();
    return _nonWebNativeExtensions.any((ext) => lower.endsWith(ext));
  }

  String _extensionOf(String name) {
    final idx = name.lastIndexOf('.');
    if (idx < 0) return '';
    return name.substring(idx).toUpperCase();
  }

  @override
  void initState() {
    super.initState();
    _initializePlayer();
  }

  Future<void> _initializePlayer() async {
    setState(() {
      _loading = true;
      _errorMessage = null;
      _isUnsupportedFormat = false;
    });

    VideoPlayerController? networkController;
    try {
      networkController = VideoPlayerController.networkUrl(
        widget.url,
        formatHint: _formatHintFromFileName(widget.name),
      );
      await networkController.initialize();
    } catch (e) {
      debugPrint('[video_viewer_page.dart] initialize error: $e');
      await networkController?.dispose();
      if (!mounted) {
        return;
      }
      final nonWebNative = _isNonWebNativeFormat(widget.name);
      setState(() {
        _loading = false;
        _isUnsupportedFormat = nonWebNative;
        _errorMessage = nonWebNative
            ? 'This video format (${_extensionOf(widget.name)}) isn\'t supported '
                  'for in-browser playback. Download the file to watch it locally.'
            : 'Unable to play this media. The file may use an unsupported '
                  'codec/profile. ($e)';
      });
      return;
    }

    if (!mounted) {
      await networkController.dispose();
      return;
    }

    setState(() {
      _controller = networkController;
      _loading = false;
    });

    // Best-effort autoplay. On web, the browser may block play() if the page
    // was opened without a prior user gesture (e.g. a direct deep-link URL).
    // In that case the video shows in a paused state — the user can tap the
    // play button to start playback.
    try {
      await networkController.play();
    } catch (_) {
      // Autoplay blocked or unsupported; stay paused.
    }
  }

  VideoFormat? _formatHintFromFileName(String name) {
    final lower = name.toLowerCase();
    if (lower.endsWith('.m3u8')) {
      return VideoFormat.hls;
    }
    if (lower.endsWith('.mpd')) {
      return VideoFormat.dash;
    }
    if (lower.endsWith('.ism') || lower.endsWith('.isml')) {
      return VideoFormat.ss;
    }
    return VideoFormat.other;
  }

  @override
  void dispose() {
    _controller?.dispose();
    _controller = null;
    super.dispose();
  }

  Future<void> _openFullscreen() async {
    final controller = _controller;
    if (controller == null) {
      return;
    }

    await Navigator.of(context).push(
      MaterialPageRoute<void>(
        builder: (_) => _FullscreenVideoPage(controller: controller),
      ),
    );

    if (mounted) {
      setState(() {});
    }
  }

  List<Widget> _buildDownloadSection() {
    return [
      const SizedBox(height: 16),
      FilledButton.icon(
        onPressed: _downloading ? null : _downloadVideo,
        icon: _downloading
            ? const SizedBox(
                width: 16,
                height: 16,
                child: CircularProgressIndicator(strokeWidth: 2),
              )
            : const Icon(Icons.download_rounded, size: 16),
        label: Text(_downloading ? 'Downloading…' : 'Download'),
      ),
    ];
  }

  Future<void> _downloadVideo() async {
    setState(() => _downloading = true);
    try {
      final Uint8List? bytes = await CirrusService.downloadFileBytes(
        widget.url.path,
      );
      if (bytes == null) throw Exception('Empty response from server');
      await web_download.saveBytesForDownload(bytes, widget.name);
    } catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(SnackBar(content: Text('Download failed: $e')));
    } finally {
      if (mounted) setState(() => _downloading = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final controller = _controller;

    return Scaffold(
      appBar: AppBar(title: Text(widget.name)),
      body: Center(
        child: _loading
            ? const CircularProgressIndicator()
            : _errorMessage != null
            ? Padding(
                padding: const EdgeInsets.all(16),
                child: Column(
                  mainAxisSize: MainAxisSize.min,
                  children: [
                    Icon(
                      _isUnsupportedFormat
                          ? Icons.video_file_outlined
                          : Icons.error_outline,
                      size: 36,
                    ),
                    const SizedBox(height: 12),
                    Text(_errorMessage!, textAlign: TextAlign.center),
                    if (_isUnsupportedFormat) ..._buildDownloadSection(),
                  ],
                ),
              )
            : controller != null
            ? _InlineVideoPlayer(
                controller: controller,
                onToggleFullscreen: _openFullscreen,
              )
            : const SizedBox.shrink(),
      ),
    );
  }
}

class _InlineVideoPlayer extends StatelessWidget {
  final VideoPlayerController controller;
  final VoidCallback onToggleFullscreen;

  const _InlineVideoPlayer({
    required this.controller,
    required this.onToggleFullscreen,
  });

  @override
  Widget build(BuildContext context) {
    return ClipRRect(
      borderRadius: BorderRadius.circular(12),
      child: _VideoSurfaceWithControls(
        controller: controller,
        isFullscreen: false,
        onToggleFullscreen: onToggleFullscreen,
      ),
    );
  }
}

class _FullscreenVideoPage extends StatefulWidget {
  final VideoPlayerController controller;

  const _FullscreenVideoPage({required this.controller});

  @override
  State<_FullscreenVideoPage> createState() => _FullscreenVideoPageState();
}

class _FullscreenVideoPageState extends State<_FullscreenVideoPage> {
  @override
  void initState() {
    super.initState();
    _enterFullscreenMode();
  }

  Future<void> _enterFullscreenMode() async {
    await SystemChrome.setEnabledSystemUIMode(SystemUiMode.immersiveSticky);
    await SystemChrome.setPreferredOrientations(const [
      DeviceOrientation.landscapeLeft,
      DeviceOrientation.landscapeRight,
    ]);
  }

  Future<void> _exitFullscreenMode() async {
    await SystemChrome.setEnabledSystemUIMode(SystemUiMode.edgeToEdge);
    await SystemChrome.setPreferredOrientations(const [
      DeviceOrientation.portraitUp,
      DeviceOrientation.landscapeLeft,
      DeviceOrientation.landscapeRight,
    ]);
  }

  @override
  void dispose() {
    _exitFullscreenMode();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final topInset = MediaQuery.of(context).padding.top;

    return Scaffold(
      backgroundColor: Colors.black,
      body: _VideoSurfaceWithControls(
        controller: widget.controller,
        isFullscreen: true,
        onToggleFullscreen: () => Navigator.of(context).pop(),
        topOverlay: Padding(
          padding: EdgeInsets.only(top: topInset + 8, left: 8),
          child: Align(
            alignment: Alignment.topLeft,
            child: IconButton(
              icon: const Icon(Icons.arrow_back, color: Colors.white),
              onPressed: () => Navigator.of(context).pop(),
            ),
          ),
        ),
      ),
    );
  }
}

class _VideoSurfaceWithControls extends StatefulWidget {
  final VideoPlayerController controller;
  final bool isFullscreen;
  final VoidCallback onToggleFullscreen;
  final Widget? topOverlay;

  const _VideoSurfaceWithControls({
    required this.controller,
    required this.isFullscreen,
    required this.onToggleFullscreen,
    this.topOverlay,
  });

  @override
  State<_VideoSurfaceWithControls> createState() =>
      _VideoSurfaceWithControlsState();
}

class _VideoSurfaceWithControlsState extends State<_VideoSurfaceWithControls> {
  static const Duration _hideDelay = Duration(seconds: 3);
  Timer? _hideTimer;
  bool _controlsVisible = true;

  @override
  void initState() {
    super.initState();
    widget.controller.addListener(_handlePlayerStateChanged);
    _scheduleHideIfPlaying();
  }

  @override
  void didUpdateWidget(covariant _VideoSurfaceWithControls oldWidget) {
    super.didUpdateWidget(oldWidget);
    if (oldWidget.controller != widget.controller) {
      oldWidget.controller.removeListener(_handlePlayerStateChanged);
      widget.controller.addListener(_handlePlayerStateChanged);
      _scheduleHideIfPlaying();
    }
  }

  @override
  void dispose() {
    widget.controller.removeListener(_handlePlayerStateChanged);
    _cancelHideTimer();
    super.dispose();
  }

  void _handlePlayerStateChanged() {
    if (!mounted) {
      return;
    }
    if (widget.controller.value.isPlaying) {
      _scheduleHideIfPlaying();
      return;
    }

    _cancelHideTimer();
    if (!_controlsVisible) {
      setState(() {
        _controlsVisible = true;
      });
    }
  }

  void _cancelHideTimer() {
    _hideTimer?.cancel();
    _hideTimer = null;
  }

  void _scheduleHideIfPlaying() {
    _cancelHideTimer();
    if (!widget.controller.value.isPlaying) {
      return;
    }
    _hideTimer = Timer(_hideDelay, () {
      if (!mounted || !widget.controller.value.isPlaying) {
        return;
      }
      setState(() {
        _controlsVisible = false;
      });
    });
  }

  void _onSurfaceTap() {
    setState(() {
      _controlsVisible = !_controlsVisible;
    });
    if (_controlsVisible) {
      _scheduleHideIfPlaying();
    } else {
      _cancelHideTimer();
    }
  }

  void _onControlsInteraction() {
    if (!_controlsVisible) {
      setState(() {
        _controlsVisible = true;
      });
    }
    _scheduleHideIfPlaying();
  }

  double _safeAspectRatio(double raw) {
    if (!raw.isFinite || raw <= 0) {
      return 16 / 9;
    }
    return raw;
  }

  @override
  Widget build(BuildContext context) {
    final ratio = _safeAspectRatio(widget.controller.value.aspectRatio);

    return ColoredBox(
      color: Colors.black,
      child: Stack(
        alignment: Alignment.bottomCenter,
        children: [
          Center(
            child: AspectRatio(
              aspectRatio: ratio,
              child: VideoPlayer(widget.controller),
            ),
          ),
          Positioned.fill(
            child: GestureDetector(
              behavior: HitTestBehavior.opaque,
              onTap: _onSurfaceTap,
              child: const SizedBox.expand(),
            ),
          ),
          if (widget.topOverlay != null)
            AnimatedOpacity(
              opacity: _controlsVisible ? 1 : 0,
              duration: const Duration(milliseconds: 180),
              child: IgnorePointer(
                ignoring: !_controlsVisible,
                child: widget.topOverlay,
              ),
            ),
          AnimatedOpacity(
            opacity: _controlsVisible ? 1 : 0,
            duration: const Duration(milliseconds: 180),
            child: IgnorePointer(
              ignoring: !_controlsVisible,
              child: _PlayerControls(
                controller: widget.controller,
                isFullscreen: widget.isFullscreen,
                onToggleFullscreen: () {
                  _onControlsInteraction();
                  widget.onToggleFullscreen();
                },
                onInteraction: _onControlsInteraction,
              ),
            ),
          ),
        ],
      ),
    );
  }
}

class _PlayerControls extends StatelessWidget {
  final VideoPlayerController controller;
  final bool isFullscreen;
  final VoidCallback onToggleFullscreen;
  final VoidCallback onInteraction;

  const _PlayerControls({
    required this.controller,
    required this.isFullscreen,
    required this.onToggleFullscreen,
    required this.onInteraction,
  });

  Future<void> _seekBy(Duration delta) async {
    final value = controller.value;
    final duration = value.duration;
    if (duration <= Duration.zero) {
      return;
    }

    final current = value.position;
    final target = current + delta;
    if (target < Duration.zero) {
      await controller.seekTo(Duration.zero);
      return;
    }
    if (target > duration) {
      await controller.seekTo(duration);
      return;
    }
    await controller.seekTo(target);
  }

  String _formatTime(Duration duration) {
    final clamped = duration < Duration.zero ? Duration.zero : duration;
    final hours = clamped.inHours;
    final minutes = clamped.inMinutes.remainder(60).toString().padLeft(2, '0');
    final seconds = clamped.inSeconds.remainder(60).toString().padLeft(2, '0');
    if (hours > 0) {
      return '$hours:$minutes:$seconds';
    }
    return '${clamped.inMinutes}:$seconds';
  }

  @override
  Widget build(BuildContext context) {
    return ValueListenableBuilder<VideoPlayerValue>(
      valueListenable: controller,
      builder: (context, value, _) {
        final duration = value.duration;
        final position = value.position;
        final isMuted = value.volume == 0;

        return Container(
          padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 10),
          decoration: const BoxDecoration(
            gradient: LinearGradient(
              begin: Alignment.topCenter,
              end: Alignment.bottomCenter,
              colors: [Colors.transparent, Color(0xB3000000)],
            ),
          ),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              VideoProgressIndicator(
                controller,
                allowScrubbing: true,
                padding: const EdgeInsets.symmetric(vertical: 6),
                colors: VideoProgressColors(
                  playedColor: Theme.of(context).colorScheme.primary,
                  bufferedColor: Colors.white54,
                  backgroundColor: Colors.white24,
                ),
              ),
              SingleChildScrollView(
                scrollDirection: Axis.horizontal,
                child: Row(
                  children: [
                    IconButton(
                      onPressed: () {
                        onInteraction();
                        value.isPlaying
                            ? controller.pause()
                            : controller.play();
                      },
                      icon: Icon(
                        value.isPlaying ? Icons.pause : Icons.play_arrow,
                        color: Colors.white,
                      ),
                    ),
                    IconButton(
                      onPressed: () {
                        onInteraction();
                        _seekBy(const Duration(seconds: -10));
                      },
                      icon: const Icon(Icons.replay_10, color: Colors.white),
                    ),
                    IconButton(
                      onPressed: () {
                        onInteraction();
                        _seekBy(const Duration(seconds: 10));
                      },
                      icon: const Icon(Icons.forward_10, color: Colors.white),
                    ),
                    Text(
                      '\${_formatTime(position)} / \${_formatTime(duration)}',
                      style: const TextStyle(color: Colors.white),
                    ),
                    IconButton(
                      onPressed: () {
                        onInteraction();
                        controller.setVolume(isMuted ? 1 : 0);
                      },
                      icon: Icon(
                        isMuted ? Icons.volume_off : Icons.volume_up,
                        color: Colors.white,
                      ),
                    ),
                    PopupMenuButton<double>(
                      tooltip: 'Playback speed',
                      initialValue: value.playbackSpeed,
                      onSelected: (speed) {
                        onInteraction();
                        controller.setPlaybackSpeed(speed);
                      },
                      itemBuilder: (_) => const [
                        PopupMenuItem(value: 0.5, child: Text('0.5x')),
                        PopupMenuItem(value: 0.75, child: Text('0.75x')),
                        PopupMenuItem(value: 1.0, child: Text('1.0x')),
                        PopupMenuItem(value: 1.25, child: Text('1.25x')),
                        PopupMenuItem(value: 1.5, child: Text('1.5x')),
                        PopupMenuItem(value: 2.0, child: Text('2.0x')),
                      ],
                      child: Padding(
                        padding: const EdgeInsets.symmetric(horizontal: 10),
                        child: Text(
                          '\${value.playbackSpeed.toStringAsFixed(2)}x',
                          style: const TextStyle(color: Colors.white),
                        ),
                      ),
                    ),
                    IconButton(
                      onPressed: () {
                        onInteraction();
                        onToggleFullscreen();
                      },
                      icon: Icon(
                        isFullscreen ? Icons.fullscreen_exit : Icons.fullscreen,
                        color: Colors.white,
                      ),
                    ),
                  ],
                ),
              ),
            ],
          ),
        );
      },
    );
  }
}
