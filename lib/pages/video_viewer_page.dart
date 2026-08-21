import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:quark/services/cirrus_service.dart';
import 'package:quark/utils/web_download_stub.dart'
    if (dart.library.html) 'package:quark/utils/web_download_web.dart'
    as web_download;
import 'package:quark/widgets/layout/theme_toggle_button.dart';
import 'package:quark_icons/quark_icons.dart';
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
  bool _savingFrame = false;
  bool _trimMode = false;
  bool _exportingTrim = false;
  // Trim handles as fractions [0.0, 1.0] of total duration.
  double _trimStart = 0.0;
  double _trimEnd = 1.0;

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

  void _enterTrimMode() {
    final controller = _controller;
    if (controller == null || !controller.value.isInitialized) return;
    controller.pause();
    setState(() {
      _trimMode = true;
      _trimStart = 0.0;
      _trimEnd = 1.0;
    });
  }

  Future<void> _exportTrim() async {
    final controller = _controller;
    if (controller == null || !controller.value.isInitialized) return;
    if (_exportingTrim) return;

    final params = widget.url.queryParameters;
    final relPath = params['filePath'] ?? '';
    final serial = params['serial'];
    if (relPath.isEmpty) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Cannot trim: file path unknown')),
      );
      return;
    }

    final totalMs = controller.value.duration.inMilliseconds;
    final startMs = (_trimStart * totalMs).round();
    final endMs = (_trimEnd * totalMs).round();

    setState(() => _exportingTrim = true);
    try {
      final savedPath = await CirrusService.trimVideo(
        relPath,
        serial: serial,
        startMs: startMs,
        endMs: endMs,
      );
      if (!mounted) return;
      final fileName = savedPath.split('/').last;
      setState(() {
        _trimMode = false;
        _trimStart = 0.0;
        _trimEnd = 1.0;
      });
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(SnackBar(content: Text('Clip saved as $fileName')));
    } catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(SnackBar(content: Text('Trim failed: $e')));
    } finally {
      if (mounted) setState(() => _exportingTrim = false);
    }
  }

  Future<void> _saveFrame() async {
    final controller = _controller;
    if (controller == null || !controller.value.isInitialized) return;
    if (_savingFrame) return;

    // Derive relPath and serial from the media URL query parameters.
    final params = widget.url.queryParameters;
    final relPath = params['filePath'] ?? '';
    final serial = params['serial'];
    if (relPath.isEmpty) {
      ScaffoldMessenger.of(context).showSnackBar(
        const SnackBar(content: Text('Cannot save frame: file path unknown')),
      );
      return;
    }

    final wasPlaying = controller.value.isPlaying;
    if (wasPlaying) await controller.pause();

    final positionMs = controller.value.position.inMilliseconds;

    if (!mounted) return;
    setState(() => _savingFrame = true);
    try {
      final savedPath = await CirrusService.extractVideoFrame(
        relPath,
        serial: serial,
        timestampMs: positionMs,
      );
      if (!mounted) return;
      final fileName = savedPath.split('/').last;
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(SnackBar(content: Text('Frame saved as $fileName')));
    } catch (e) {
      if (!mounted) return;
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(SnackBar(content: Text('Save frame failed: $e')));
    } finally {
      if (mounted) setState(() => _savingFrame = false);
      if (wasPlaying && _controller != null) await _controller!.play();
    }
  }

  @override
  Widget build(BuildContext context) {
    final controller = _controller;

    return Scaffold(
      appBar: AppBar(
        title: Text(widget.name),
        actions: [
          if (_exportingTrim)
            const Padding(
              padding: EdgeInsets.symmetric(horizontal: 12),
              child: SizedBox(
                width: 20,
                height: 20,
                child: CircularProgressIndicator(strokeWidth: 2),
              ),
            )
          else if (_trimMode) ...[
            TextButton(
              onPressed: _exportTrim,
              child: const Text(
                'Save Clip',
                style: TextStyle(color: Colors.white),
              ),
            ),
            TextButton(
              onPressed: () => setState(() {
                _trimMode = false;
                _trimStart = 0.0;
                _trimEnd = 1.0;
              }),
              child: const Text(
                'Cancel',
                style: TextStyle(color: Colors.white70),
              ),
            ),
          ] else if (!_savingFrame)
            PopupMenuButton<String>(
              tooltip: 'More options',
              onSelected: (action) {
                if (action == 'saveFrame') _saveFrame();
                if (action == 'trim') _enterTrimMode();
              },
              itemBuilder: (_) => const [
                PopupMenuItem(value: 'saveFrame', child: Text('Save Frame')),
                PopupMenuItem(value: 'trim', child: Text('Trim Clip')),
              ],
            )
          else
            const Padding(
              padding: EdgeInsets.symmetric(horizontal: 12),
              child: SizedBox(
                width: 20,
                height: 20,
                child: CircularProgressIndicator(strokeWidth: 2),
              ),
            ),
          const ThemeToggleButton(),
        ],
      ),
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
                          ? QuarkIcons.video_file_outlined
                          : QuarkIcons.error_outline,
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
                trimMode: _trimMode,
                trimStart: _trimStart,
                trimEnd: _trimEnd,
                onTrimStartChanged: (v) => setState(() => _trimStart = v),
                onTrimEndChanged: (v) => setState(() => _trimEnd = v),
              )
            : const SizedBox.shrink(),
      ),
    );
  }
}

class _InlineVideoPlayer extends StatelessWidget {
  final VideoPlayerController controller;
  final VoidCallback onToggleFullscreen;
  final bool trimMode;
  final double trimStart;
  final double trimEnd;
  final ValueChanged<double>? onTrimStartChanged;
  final ValueChanged<double>? onTrimEndChanged;

  const _InlineVideoPlayer({
    required this.controller,
    required this.onToggleFullscreen,
    this.trimMode = false,
    this.trimStart = 0.0,
    this.trimEnd = 1.0,
    this.onTrimStartChanged,
    this.onTrimEndChanged,
  });

  @override
  Widget build(BuildContext context) {
    return ClipRRect(
      borderRadius: BorderRadius.circular(12),
      child: _VideoSurfaceWithControls(
        controller: controller,
        isFullscreen: false,
        onToggleFullscreen: onToggleFullscreen,
        trimMode: trimMode,
        trimStart: trimStart,
        trimEnd: trimEnd,
        onTrimStartChanged: onTrimStartChanged,
        onTrimEndChanged: onTrimEndChanged,
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
              icon: const Icon(QuarkIcons.arrow_back, color: Colors.white),
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
  // Trim state passed down from _VideoViewerPageState.
  final bool trimMode;
  final double trimStart;
  final double trimEnd;
  final ValueChanged<double>? onTrimStartChanged;
  final ValueChanged<double>? onTrimEndChanged;

  const _VideoSurfaceWithControls({
    required this.controller,
    required this.isFullscreen,
    required this.onToggleFullscreen,
    this.topOverlay,
    this.trimMode = false,
    this.trimStart = 0.0,
    this.trimEnd = 1.0,
    this.onTrimStartChanged,
    this.onTrimEndChanged,
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
                trimMode: widget.trimMode,
                trimStart: widget.trimStart,
                trimEnd: widget.trimEnd,
                onTrimStartChanged: widget.onTrimStartChanged,
                onTrimEndChanged: widget.onTrimEndChanged,
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
  final bool trimMode;
  final double trimStart;
  final double trimEnd;
  final ValueChanged<double>? onTrimStartChanged;
  final ValueChanged<double>? onTrimEndChanged;

  const _PlayerControls({
    required this.controller,
    required this.isFullscreen,
    required this.onToggleFullscreen,
    required this.onInteraction,
    this.trimMode = false,
    this.trimStart = 0.0,
    this.trimEnd = 1.0,
    this.onTrimStartChanged,
    this.onTrimEndChanged,
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
              if (trimMode)
                _TrimBar(
                  start: trimStart,
                  end: trimEnd,
                  duration: controller.value.duration,
                  onStartChanged: (v) {
                    onTrimStartChanged?.call(v);
                    final ms = (v * controller.value.duration.inMilliseconds)
                        .round();
                    controller.seekTo(Duration(milliseconds: ms));
                  },
                  onEndChanged: (v) {
                    onTrimEndChanged?.call(v);
                    final ms = (v * controller.value.duration.inMilliseconds)
                        .round();
                    controller.seekTo(Duration(milliseconds: ms));
                  },
                )
              else
                VideoProgressIndicator(
                  controller,
                  allowScrubbing: !trimMode,
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
                        value.isPlaying
                            ? QuarkIcons.pause
                            : QuarkIcons.play_arrow,
                        color: Colors.white,
                      ),
                    ),
                    IconButton(
                      onPressed: () {
                        onInteraction();
                        _seekBy(const Duration(seconds: -10));
                      },
                      icon: const Icon(
                        QuarkIcons.replay_10,
                        color: Colors.white,
                      ),
                    ),
                    IconButton(
                      onPressed: () {
                        onInteraction();
                        _seekBy(const Duration(seconds: 10));
                      },
                      icon: const Icon(
                        QuarkIcons.forward_10,
                        color: Colors.white,
                      ),
                    ),
                    Text(
                      '${_formatTime(position)} / ${_formatTime(duration)}',
                      style: const TextStyle(color: Colors.white),
                    ),
                    IconButton(
                      onPressed: () {
                        onInteraction();
                        controller.setVolume(isMuted ? 1 : 0);
                      },
                      icon: Icon(
                        isMuted ? QuarkIcons.volume_off : QuarkIcons.volume_up,
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
                          '${value.playbackSpeed.toStringAsFixed(2)}x',
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
                        isFullscreen
                            ? QuarkIcons.fullscreen_exit
                            : QuarkIcons.fullscreen,
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

/// A trim range bar with two draggable handles (start and end).
///
/// Renders a track with:
/// - A semi-transparent overlay for the un-selected regions (before start, after end)
/// - A highlighted selected region between start and end
/// - Draggable circular handles at start and end positions
/// - Timestamp labels above each handle
class _TrimBar extends StatelessWidget {
  final double start; // 0.0–1.0 fraction
  final double end; // 0.0–1.0 fraction
  final Duration duration;
  final ValueChanged<double> onStartChanged;
  final ValueChanged<double> onEndChanged;

  const _TrimBar({
    required this.start,
    required this.end,
    required this.duration,
    required this.onStartChanged,
    required this.onEndChanged,
  });

  static const _handleSize = 24.0;
  static const _trackHeight = 6.0;

  // The label sits in a fixed slot above the handle. Reserving the slot (rather
  // than letting the label size itself around the handle) is what keeps the
  // handle's centre free to land exactly on the track.
  static const _labelHeight = 14.0;
  static const _labelWidth = 72.0;
  static const _labelGap = 2.0;
  static const _barHeight = _labelHeight + _labelGap + _handleSize;

  /// Shared centre line for the handles and the track, measured from the top
  /// of the bar. Everything vertical is anchored to this so the handles and
  /// the track share a centre instead of each being centred independently.
  static const _centerY = _labelHeight + _labelGap + _handleSize / 2;

  String _formatMs(int ms) {
    final d = Duration(milliseconds: ms);
    final h = d.inHours;
    final m = d.inMinutes % 60;
    final s = d.inSeconds % 60;
    final tenths = (ms % 1000) ~/ 100;
    if (h > 0) {
      return '${h}h${m.toString().padLeft(2, '0')}m${s.toString().padLeft(2, '0')}s';
    }
    return '${m.toString().padLeft(2, '0')}:${s.toString().padLeft(2, '0')}.$tenths';
  }

  @override
  Widget build(BuildContext context) {
    final primary = Theme.of(context).colorScheme.primary;
    final totalMs = duration.inMilliseconds;
    final startLabel = _formatMs((start * totalMs).round());
    final endLabel = _formatMs((end * totalMs).round());

    return Padding(
      padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 4),
      child: LayoutBuilder(
        builder: (context, constraints) {
          final width = constraints.maxWidth;
          final startX = start * width;
          final endX = end * width;

          return SizedBox(
            height: _barHeight,
            child: Stack(
              children: [
                // Track background
                Positioned(
                  left: 0,
                  right: 0,
                  top: _centerY - _trackHeight / 2,
                  child: Container(
                    height: _trackHeight,
                    decoration: BoxDecoration(
                      color: Colors.white24,
                      borderRadius: BorderRadius.circular(_trackHeight / 2),
                    ),
                  ),
                ),
                // Selected region highlight
                Positioned(
                  left: startX,
                  width: (endX - startX).clamp(0.0, width),
                  top: _centerY - _trackHeight / 2,
                  child: Container(
                    height: _trackHeight,
                    color: primary.withValues(alpha: 0.7),
                  ),
                ),
                // Start handle + label
                ..._buildHandle(
                  centerX: startX,
                  width: width,
                  label: startLabel,
                  color: primary,
                  onDragDelta: (dx) => onStartChanged(
                    ((startX + dx) / width).clamp(0.0, end - 0.01),
                  ),
                ),
                // End handle + label
                ..._buildHandle(
                  centerX: endX,
                  width: width,
                  label: endLabel,
                  color: primary,
                  onDragDelta: (dx) => onEndChanged(
                    ((endX + dx) / width).clamp(start + 0.01, 1.0),
                  ),
                ),
              ],
            ),
          );
        },
      ),
    );
  }

  /// The label and the handle are positioned separately on purpose. Nesting
  /// them in a single Column made the Column — not the handle — the thing
  /// being positioned, so the handle inherited the label's width and height
  /// and drifted right and down off the track.
  List<Widget> _buildHandle({
    required double centerX,
    required double width,
    required String label,
    required Color color,
    required ValueChanged<double> onDragDelta,
  }) {
    // Slot widths can exceed the bar on very narrow layouts; keep the clamp
    // bounds ordered so they stay valid.
    final labelLeftMax = (width - _labelWidth).clamp(0.0, double.infinity);
    final handleLeftMax = (width - _handleSize).clamp(0.0, double.infinity);

    return [
      Positioned(
        top: 0,
        left: (centerX - _labelWidth / 2).clamp(0.0, labelLeftMax),
        width: _labelWidth,
        height: _labelHeight,
        child: Center(
          child: Text(
            label,
            textAlign: TextAlign.center,
            style: const TextStyle(color: Colors.white, fontSize: 10),
          ),
        ),
      ),
      Positioned(
        top: _centerY - _handleSize / 2,
        left: (centerX - _handleSize / 2).clamp(0.0, handleLeftMax),
        child: GestureDetector(
          onHorizontalDragUpdate: (details) => onDragDelta(details.delta.dx),
          child: Container(
            width: _handleSize,
            height: _handleSize,
            decoration: BoxDecoration(
              color: color,
              shape: BoxShape.circle,
              boxShadow: [
                BoxShadow(
                  color: Colors.black.withValues(alpha: 0.4),
                  blurRadius: 4,
                ),
              ],
            ),
          ),
        ),
      ),
    ];
  }
}
