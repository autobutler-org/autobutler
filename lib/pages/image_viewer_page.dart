import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

/// A full-screen image viewer with left/right navigation.
///
/// Pass [initialIndex] and [imageCount] to enable navigation; supply
/// [onLoadImage] to fetch bytes for adjacent images on demand.
///
/// Navigation:
///   - Keyboard: ← / → arrow keys, Escape to close
///   - Swipe: horizontal drag on touch/trackpad
///   - Buttons: chevron overlays on left/right edges
class ImageViewerPage extends StatefulWidget {
  /// Bytes for the initially displayed image.
  final Uint8List bytes;

  /// Display name for the initial image (shown in the app bar).
  final String name;

  /// Zero-based index of the initial image within the full list.
  final int initialIndex;

  /// Total number of images in the list. Set to 1 (default) for a single image.
  final int imageCount;

  /// Called when the user navigates to a different image. Receives the new
  /// index and must return the bytes and name for that image.
  /// Required when [imageCount] > 1.
  final Future<(Uint8List?, String)> Function(int index)? onLoadImage;

  const ImageViewerPage({
    super.key,
    required this.bytes,
    required this.name,
    this.initialIndex = 0,
    this.imageCount = 1,
    this.onLoadImage,
  });

  @override
  State<ImageViewerPage> createState() => _ImageViewerPageState();
}

class _ImageViewerPageState extends State<ImageViewerPage> {
  late int _currentIndex;
  late Uint8List _currentBytes;
  late String _currentName;
  bool _loading = false;

  final _focusNode = FocusNode();

  @override
  void initState() {
    super.initState();
    _currentIndex = widget.initialIndex;
    _currentBytes = widget.bytes;
    _currentName = widget.name;
  }

  @override
  void dispose() {
    _focusNode.dispose();
    super.dispose();
  }

  bool get _hasPrev => _currentIndex > 0;
  bool get _hasNext => _currentIndex < widget.imageCount - 1;

  Future<void> _navigate(int delta) async {
    if (_loading) return;
    final newIndex = _currentIndex + delta;
    if (newIndex < 0 || newIndex >= widget.imageCount) return;
    if (widget.onLoadImage == null) return;

    setState(() => _loading = true);
    try {
      final (bytes, name) = await widget.onLoadImage!(newIndex);
      if (!mounted) return;
      if (bytes == null) {
        if (mounted) {
          setState(() => _loading = false);
          ScaffoldMessenger.of(context).showSnackBar(
            const SnackBar(
              content: Text('Image no longer available'),
              duration: Duration(seconds: 2),
            ),
          );
        }
        return;
      }
      setState(() {
        _currentIndex = newIndex;
        _currentBytes = bytes;
        _currentName = name;
        _loading = false;
      });
    } catch (e) {
      debugPrint('[image_viewer_page.dart] Navigation error: $e');
      if (mounted) {
        setState(() => _loading = false);
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(
            content: Text('Image no longer available'),
            duration: Duration(seconds: 2),
          ),
        );
      }
    }
  }

  KeyEventResult _handleKey(FocusNode node, KeyEvent event) {
    if (event is! KeyDownEvent) return KeyEventResult.ignored;
    if (event.logicalKey == LogicalKeyboardKey.arrowLeft) {
      _navigate(-1);
      return KeyEventResult.handled;
    }
    if (event.logicalKey == LogicalKeyboardKey.arrowRight) {
      _navigate(1);
      return KeyEventResult.handled;
    }
    if (event.logicalKey == LogicalKeyboardKey.escape) {
      Navigator.of(context).pop();
      return KeyEventResult.handled;
    }
    return KeyEventResult.ignored;
  }

  @override
  Widget build(BuildContext context) {
    final showNav = widget.imageCount > 1;
    return KeyboardListener(
      focusNode: _focusNode,
      autofocus: true,
      onKeyEvent: (event) => _handleKey(_focusNode, event),
      child: Scaffold(
        backgroundColor: Colors.black,
        appBar: AppBar(
          backgroundColor: Colors.black,
          foregroundColor: Colors.white,
          title: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Text(
                _currentName,
                style: const TextStyle(color: Colors.white),
                overflow: TextOverflow.ellipsis,
              ),
              if (showNav)
                Text(
                  '${_currentIndex + 1} / ${widget.imageCount}',
                  style: const TextStyle(color: Colors.white70, fontSize: 12),
                ),
            ],
          ),
        ),
        body: GestureDetector(
          onHorizontalDragEnd: (details) {
            if (details.primaryVelocity == null) return;
            if (details.primaryVelocity! < -200) {
              _navigate(1); // swipe left → next
            } else if (details.primaryVelocity! > 200) {
              _navigate(-1); // swipe right → prev
            }
          },
          child: Stack(
            children: [
              // Main image
              Center(
                child: _loading
                    ? const CircularProgressIndicator(color: Colors.white)
                    : InteractiveViewer(
                        child: Image.memory(
                          _currentBytes,
                          fit: BoxFit.contain,
                          errorBuilder: (c, e, st) => const Icon(
                            Icons.broken_image,
                            size: 64,
                            color: Colors.white54,
                          ),
                        ),
                      ),
              ),

              // Prev button
              if (showNav && _hasPrev)
                Positioned(
                  left: 8,
                  top: 0,
                  bottom: 0,
                  child: Center(
                    child: _NavButton(
                      icon: Icons.chevron_left,
                      tooltip: 'Previous (←)',
                      onTap: _loading ? null : () => _navigate(-1),
                    ),
                  ),
                ),

              // Next button
              if (showNav && _hasNext)
                Positioned(
                  right: 8,
                  top: 0,
                  bottom: 0,
                  child: Center(
                    child: _NavButton(
                      icon: Icons.chevron_right,
                      tooltip: 'Next (→)',
                      onTap: _loading ? null : () => _navigate(1),
                    ),
                  ),
                ),
            ],
          ),
        ),
      ),
    );
  }
}

class _NavButton extends StatelessWidget {
  final IconData icon;
  final String tooltip;
  final VoidCallback? onTap;

  const _NavButton({
    required this.icon,
    required this.tooltip,
    required this.onTap,
  });

  @override
  Widget build(BuildContext context) {
    return Tooltip(
      message: tooltip,
      child: Material(
        color: Colors.black54,
        borderRadius: BorderRadius.circular(32),
        child: InkWell(
          borderRadius: BorderRadius.circular(32),
          onTap: onTap,
          child: Padding(
            padding: const EdgeInsets.all(8),
            child: Icon(icon, color: Colors.white, size: 36),
          ),
        ),
      ),
    );
  }
}
