import 'package:autobutler/services/cirrus_service.dart';
import 'package:autobutler/widgets/layout/theme_toggle_button.dart';
import 'package:flutter/material.dart';
import 'package:pdfrx/pdfrx.dart';

/// Renders a PDF file fetched from the butler.
///
/// Supports pinch-to-zoom, scroll-to-page, and a page counter in the AppBar.
/// Uses [pdfrx] which is backed by PDFium and works on Android, iOS, Web,
/// macOS, Windows, and Linux.
class PdfViewerPage extends StatefulWidget {
  final String filePath;
  final String? deviceSerial;

  const PdfViewerPage({required this.filePath, this.deviceSerial, super.key});

  @override
  State<PdfViewerPage> createState() => _PdfViewerPageState();
}

class _PdfViewerPageState extends State<PdfViewerPage> {
  late final PdfViewerController _controller;
  int _currentPage = 1;
  int _totalPages = 0;
  bool _loading = true;
  String? _error;

  String get _fileName =>
      widget.filePath.split('/').last.replaceAll(RegExp(r'\?.*'), '');

  @override
  void initState() {
    super.initState();
    _controller = PdfViewerController();
  }

  Uri get _downloadUri => Uri.parse(
    CirrusService.constructMediaUrl(
      widget.filePath,
      serial: widget.deviceSerial,
    ),
  );

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: Text(_fileName),
        actions: [
          if (_totalPages > 0)
            Padding(
              padding: const EdgeInsets.symmetric(horizontal: 8),
              child: Center(
                child: Text(
                  '$_currentPage / $_totalPages',
                  style: Theme.of(context).textTheme.bodyMedium,
                ),
              ),
            ),
          const ThemeToggleButton(),
        ],
      ),
      body: _buildBody(),
      floatingActionButton: _totalPages > 1
          ? Column(
              mainAxisSize: MainAxisSize.min,
              children: [
                FloatingActionButton.small(
                  heroTag: 'prev_page',
                  tooltip: 'Previous page',
                  onPressed: _currentPage > 1
                      ? () => _controller.goToPage(pageNumber: _currentPage - 1)
                      : null,
                  child: const Icon(Icons.keyboard_arrow_up),
                ),
                const SizedBox(height: 8),
                FloatingActionButton.small(
                  heroTag: 'next_page',
                  tooltip: 'Next page',
                  onPressed: _currentPage < _totalPages
                      ? () => _controller.goToPage(pageNumber: _currentPage + 1)
                      : null,
                  child: const Icon(Icons.keyboard_arrow_down),
                ),
              ],
            )
          : null,
    );
  }

  Widget _buildBody() {
    if (_error != null) {
      return Center(
        child: Padding(
          padding: const EdgeInsets.all(24),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            children: [
              const Icon(
                Icons.broken_image_outlined,
                size: 48,
                color: Colors.red,
              ),
              const SizedBox(height: 16),
              Text(_error!, textAlign: TextAlign.center),
              const SizedBox(height: 24),
              ElevatedButton.icon(
                onPressed: () => Navigator.of(context).pop(),
                icon: const Icon(Icons.arrow_back),
                label: const Text('Go Back'),
              ),
            ],
          ),
        ),
      );
    }

    return Stack(
      children: [
        PdfViewer.uri(
          _downloadUri,
          controller: _controller,
          params: PdfViewerParams(
            pageAnchor: PdfPageAnchor.top,
            onDocumentLoadFinished: (doc) {
              if (mounted) {
                setState(() {
                  _totalPages = doc.pages.length;
                  _loading = false;
                });
              }
            },
            errorBannerBuilder: (context, error, stackTrace, documentRef) {
              WidgetsBinding.instance.addPostFrameCallback((_) {
                if (mounted) {
                  setState(() {
                    _error = 'Failed to load PDF: $error';
                    _loading = false;
                  });
                }
              });
              return const SizedBox.shrink();
            },
            onPageChanged: (page) {
              if (mounted) {
                setState(() => _currentPage = page ?? 1);
              }
            },
          ),
        ),
        if (_loading) const Center(child: CircularProgressIndicator()),
      ],
    );
  }
}
