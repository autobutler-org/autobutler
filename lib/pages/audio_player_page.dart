import 'dart:async';

import 'package:flutter/material.dart';
import 'package:quark/services/cirrus_service.dart';
import 'package:quark/utils/web_download_stub.dart'
    if (dart.library.html) 'package:quark/utils/web_download_web.dart'
    as web_download;
import 'package:quark/widgets/layout/theme_toggle_button.dart';
import 'package:video_player/video_player.dart';

class AudioPlayerPage extends StatefulWidget {
  final Uri url;
  final String name;

  const AudioPlayerPage({super.key, required this.url, required this.name});

  @override
  State<AudioPlayerPage> createState() => _AudioPlayerPageState();
}

class _AudioPlayerPageState extends State<AudioPlayerPage> {
  VideoPlayerController? _controller;
  bool _loading = true;
  String? _errorMessage;
  bool _downloading = false;

  @override
  void initState() {
    super.initState();
    _initializePlayer();
  }

  Future<void> _initializePlayer() async {
    setState(() {
      _loading = true;
      _errorMessage = null;
    });

    VideoPlayerController? controller;
    try {
      controller = VideoPlayerController.networkUrl(widget.url);
      await controller.initialize();
    } catch (e) {
      debugPrint('[audio_player_page.dart] initialize error: $e');
      await controller?.dispose();
      if (!mounted) return;
      setState(() {
        _loading = false;
        _errorMessage =
            'Unable to play this audio file. The format may not be '
            'supported by this browser. ($e)';
      });
      return;
    }

    if (!mounted) {
      await controller.dispose();
      return;
    }

    setState(() {
      _controller = controller;
      _loading = false;
    });

    try {
      await controller.play();
    } catch (_) {}
  }

  @override
  void dispose() {
    _controller?.dispose();
    _controller = null;
    super.dispose();
  }

  Future<void> _download() async {
    setState(() => _downloading = true);
    try {
      final bytes = await CirrusService.downloadFileBytes(widget.url.path);
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
    return Scaffold(
      appBar: AppBar(
        title: Text(widget.name),
        actions: [
          IconButton(
            onPressed: _downloading ? null : _download,
            icon: _downloading
                ? const SizedBox(
                    width: 18,
                    height: 18,
                    child: CircularProgressIndicator(strokeWidth: 2),
                  )
                : const Icon(Icons.download_rounded),
            tooltip: 'Download',
          ),
          const ThemeToggleButton(),
        ],
      ),
      body: Center(
        child: _loading
            ? const CircularProgressIndicator()
            : _errorMessage != null
            ? _ErrorView(
                message: _errorMessage!,
                onDownload: _download,
                downloading: _downloading,
              )
            : _controller != null
            ? _AudioControls(controller: _controller!)
            : const SizedBox.shrink(),
      ),
    );
  }
}

class _ErrorView extends StatelessWidget {
  final String message;
  final VoidCallback onDownload;
  final bool downloading;

  const _ErrorView({
    required this.message,
    required this.onDownload,
    required this.downloading,
  });

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.all(24),
      child: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          const Icon(Icons.audio_file_outlined, size: 48),
          const SizedBox(height: 16),
          Text(message, textAlign: TextAlign.center),
          const SizedBox(height: 16),
          FilledButton.icon(
            onPressed: downloading ? null : onDownload,
            icon: downloading
                ? const SizedBox(
                    width: 16,
                    height: 16,
                    child: CircularProgressIndicator(strokeWidth: 2),
                  )
                : const Icon(Icons.download_rounded, size: 16),
            label: Text(downloading ? 'Downloading…' : 'Download'),
          ),
        ],
      ),
    );
  }
}

class _AudioControls extends StatefulWidget {
  final VideoPlayerController controller;

  const _AudioControls({required this.controller});

  @override
  State<_AudioControls> createState() => _AudioControlsState();
}

class _AudioControlsState extends State<_AudioControls> {
  Timer? _positionTimer;

  @override
  void initState() {
    super.initState();
    _positionTimer = Timer.periodic(const Duration(milliseconds: 250), (_) {
      if (mounted) setState(() {});
    });
  }

  @override
  void dispose() {
    _positionTimer?.cancel();
    super.dispose();
  }

  String _formatDuration(Duration d) {
    final clamped = d < Duration.zero ? Duration.zero : d;
    final hours = clamped.inHours;
    final minutes = clamped.inMinutes.remainder(60).toString().padLeft(2, '0');
    final seconds = clamped.inSeconds.remainder(60).toString().padLeft(2, '0');
    if (hours > 0) return '$hours:$minutes:$seconds';
    return '${clamped.inMinutes}:$seconds';
  }

  Future<void> _seekBy(Duration delta) async {
    final value = widget.controller.value;
    final target = value.position + delta;
    if (target < Duration.zero) {
      await widget.controller.seekTo(Duration.zero);
    } else if (target > value.duration) {
      await widget.controller.seekTo(value.duration);
    } else {
      await widget.controller.seekTo(target);
    }
  }

  @override
  Widget build(BuildContext context) {
    final value = widget.controller.value;
    final theme = Theme.of(context);
    final colors = theme.colorScheme;
    final isPlaying = value.isPlaying;
    final position = value.position;
    final duration = value.duration;
    final isMuted = value.volume == 0;
    final progress = duration.inMilliseconds > 0
        ? position.inMilliseconds / duration.inMilliseconds
        : 0.0;

    return ConstrainedBox(
      constraints: const BoxConstraints(maxWidth: 480),
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 32),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(
              Icons.music_note_rounded,
              size: 120,
              color: colors.primary.withValues(alpha: 0.3),
            ),
            const SizedBox(height: 32),
            SliderTheme(
              data: SliderTheme.of(context).copyWith(
                trackHeight: 4,
                thumbShape: const RoundSliderThumbShape(enabledThumbRadius: 6),
                overlayShape: const RoundSliderOverlayShape(overlayRadius: 14),
              ),
              child: Slider(
                value: progress.clamp(0.0, 1.0),
                onChanged: (v) {
                  final target = Duration(
                    milliseconds: (v * duration.inMilliseconds).round(),
                  );
                  widget.controller.seekTo(target);
                },
              ),
            ),
            Padding(
              padding: const EdgeInsets.symmetric(horizontal: 16),
              child: Row(
                mainAxisAlignment: MainAxisAlignment.spaceBetween,
                children: [
                  Text(
                    _formatDuration(position),
                    style: theme.textTheme.bodySmall,
                  ),
                  Text(
                    _formatDuration(duration),
                    style: theme.textTheme.bodySmall,
                  ),
                ],
              ),
            ),
            const SizedBox(height: 16),
            Row(
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                IconButton(
                  onPressed: () => _seekBy(const Duration(seconds: -10)),
                  icon: const Icon(Icons.replay_10),
                  iconSize: 28,
                ),
                const SizedBox(width: 12),
                IconButton.filled(
                  onPressed: () {
                    isPlaying
                        ? widget.controller.pause()
                        : widget.controller.play();
                  },
                  icon: Icon(isPlaying ? Icons.pause : Icons.play_arrow),
                  iconSize: 40,
                  style: IconButton.styleFrom(
                    padding: const EdgeInsets.all(16),
                  ),
                ),
                const SizedBox(width: 12),
                IconButton(
                  onPressed: () => _seekBy(const Duration(seconds: 10)),
                  icon: const Icon(Icons.forward_10),
                  iconSize: 28,
                ),
              ],
            ),
            const SizedBox(height: 16),
            Row(
              mainAxisAlignment: MainAxisAlignment.center,
              children: [
                IconButton(
                  onPressed: () {
                    widget.controller.setVolume(isMuted ? 1 : 0);
                  },
                  icon: Icon(isMuted ? Icons.volume_off : Icons.volume_up),
                ),
                PopupMenuButton<double>(
                  tooltip: 'Playback speed',
                  initialValue: value.playbackSpeed,
                  onSelected: (speed) {
                    widget.controller.setPlaybackSpeed(speed);
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
                    padding: const EdgeInsets.symmetric(
                      horizontal: 8,
                      vertical: 4,
                    ),
                    child: Text(
                      '${value.playbackSpeed.toStringAsFixed(2)}x',
                      style: theme.textTheme.bodyMedium,
                    ),
                  ),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }
}
