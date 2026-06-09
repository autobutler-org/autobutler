import 'package:autobutler/pages/file_browser_page.dart';
import 'package:autobutler/pages/image_viewer_page.dart';
import 'package:autobutler/pages/video_viewer_page.dart';
import 'package:autobutler/services/cirrus_service.dart';
import 'package:autobutler/widgets/layout/theme_toggle_button.dart';
import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';

/// A routing shim that resolves a Cirrus file path to its correct viewer.
///
/// On [initState] it calls [CirrusService.statFile] to determine the file
/// type, then immediately navigates to the appropriate page:
///
/// | fileType          | Destination                  |
/// |-------------------|------------------------------|
/// | `image`           | [ImageViewerPage]            |
/// | `video` / `audio` | [VideoViewerPage]            |
/// | `abdoc`           | /docs/&ltpath&gt                 |
/// | `absheet`         | /sheets/&ltpath&gt               |
/// | directory / other | /cirrus/&ltpath&gt (FileBrowser) |
///
/// Navigate to the route built with [AppRoutes.viewFile] to trigger this.
class FileViewerPage extends StatefulWidget {
  final String filePath;
  final String deviceSerial;

  const FileViewerPage({
    required this.filePath,
    this.deviceSerial = '',
    super.key,
  });

  @override
  State<FileViewerPage> createState() => _FileViewerPageState();
}

// ---------------------------------------------------------------------------
// Private helpers — avoids circular import with router.dart
// ---------------------------------------------------------------------------

String _cirrusPath(String path) {
  final clean = path.replaceAll(RegExp(r'^/+'), '');
  return clean.isEmpty ? '/cirrus' : '/cirrus/$clean';
}

String _buildRoute(String base, String path, {String? serial}) {
  final clean = path.replaceAll(RegExp(r'^/+'), '');
  final url = '$base/$clean';
  return (serial != null && serial.isNotEmpty)
      ? '$url?serial=${Uri.encodeQueryComponent(serial)}'
      : url;
}

// ---------------------------------------------------------------------------

class _FileViewerPageState extends State<FileViewerPage> {
  String? _errorMessage;

  @override
  void initState() {
    super.initState();
    _resolveAndNavigate();
  }

  Future<void> _resolveAndNavigate() async {
    try {
      final serial = widget.deviceSerial.trim().isEmpty
          ? null
          : widget.deviceSerial;
      final stat = await CirrusService.statFile(
        widget.filePath,
        serial: serial,
      );

      if (!mounted) return;

      final name = stat.name.isEmpty
          ? widget.filePath.split('/').last
          : stat.name;

      if (stat.isDir) {
        // Navigate into a directory by pushing FileBrowserPage directly.
        // Using context.go(_cirrusPath(...)) would re-route back to
        // FileViewerPage and create an infinite stat loop.
        Navigator.of(context).pushReplacement(
          MaterialPageRoute<void>(
            builder: (_) => FileBrowserPage(initialPath: widget.filePath),
          ),
        );
        return;
      }

      if (stat.fileType == 'archive' || stat.fileType == 'generic') {
        // Archive/generic — hand off to the file browser via route.
        context.go(_cirrusPath(widget.filePath));
        return;
      }

      switch (stat.fileType) {
        case 'image':
          final bytes = await CirrusService.downloadFileBytes(
            widget.filePath,
            serial: serial,
          );
          if (!mounted) return;
          if (bytes == null) {
            setState(() => _errorMessage = 'Failed to download image.');
            return;
          }
          Navigator.of(context).pushReplacement(
            MaterialPageRoute<void>(
              builder: (_) => ImageViewerPage(
                bytes: bytes,
                name: name,
                relPath: widget.filePath,
                serial: serial,
              ),
            ),
          );

        case 'video':
        case 'audio':
          final url = CirrusService.constructMediaUrl(
            widget.filePath,
            serial: serial,
          );
          if (!mounted) return;
          Navigator.of(context).pushReplacement(
            MaterialPageRoute<void>(
              builder: (_) => VideoViewerPage(url: url, name: name),
            ),
          );

        case 'abdoc':
          context.go(_buildRoute('/docs', widget.filePath, serial: serial));

        case 'absheet':
          context.go(_buildRoute('/sheets', widget.filePath, serial: serial));

        case 'text':
          context.push(_buildRoute('/edit', widget.filePath, serial: serial));

        default:
          // Unknown type — fall back to the parent folder in the file browser.
          final parentPath = widget.filePath.contains('/')
              ? widget.filePath.substring(0, widget.filePath.lastIndexOf('/'))
              : '';
          Navigator.of(context).pushReplacement(
            MaterialPageRoute<void>(
              builder: (_) => FileBrowserPage(initialPath: parentPath),
            ),
          );
      }
    } catch (e) {
      if (mounted) {
        setState(() => _errorMessage = e.toString());
      }
    }
  }

  @override
  Widget build(BuildContext context) {
    if (_errorMessage != null) {
      return Scaffold(
        appBar: AppBar(
          title: const Text('Open File'),
          actions: const [ThemeToggleButton()],
        ),
        body: Center(
          child: Padding(
            padding: const EdgeInsets.all(24),
            child: Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                const Icon(Icons.error_outline, size: 48, color: Colors.red),
                const SizedBox(height: 16),
                Text(_errorMessage!, textAlign: TextAlign.center),
                const SizedBox(height: 24),
                ElevatedButton.icon(
                  onPressed: () {
                    if (context.canPop()) {
                      context.pop();
                    } else {
                      context.go('/cirrus');
                    }
                  },
                  icon: const Icon(Icons.arrow_back),
                  label: const Text('Go Back'),
                ),
              ],
            ),
          ),
        ),
      );
    }

    return const Scaffold(body: Center(child: CircularProgressIndicator()));
  }
}
