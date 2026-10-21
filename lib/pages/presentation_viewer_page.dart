import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:quark/controllers/presentation_viewer_controller.dart';
import 'package:quark/router.dart';
import 'package:quark/utils/file_browser_path_utils.dart';
import 'package:quark/utils/file_viewer_actions.dart';
import 'package:quark/widgets/layout/chrome_app_bar.dart';
import 'package:quark/widgets/layout/theme_toggle_button.dart';
import 'package:quark/widgets/presentation_viewer/presentation_viewer_body.dart';
import 'package:quark/widgets/slides/import/import_power_point.dart';
import 'package:quark_icons/quark_icons.dart';
import 'package:quark_widgets/quark_widgets.dart';

/// The read-only presentation viewer at `/presentation/<path>` (#3268): a
/// `.pptx` or `.pptm` read on the Quark and shown a slide at a time, with a
/// strip of thumbnails on a wide window and the speaker notes behind a
/// toggle. Closes #196.
///
/// Macros are never read, and the pictures come from inside the file.
/// "Present" hands the deck already read to the presenter, which returns
/// here at the slide it ended on; "Open as Quark slides" imports the file
/// beside itself as a `.qslide` and opens that in the editor.
///
/// `?slide=N`, counting from 1, opens at the Nth slide.
///
/// Keys: `presentation_viewer_present`, `presentation_viewer_open_as_qslide`
/// and `presentation_viewer_download` on the bar's actions.
class PresentationViewerPage extends StatefulWidget {
  /// The presentation's files-relative path.
  final String filePath;

  /// The device the presentation is on, empty for the internal drive.
  final String deviceSerial;

  /// The slide to open at, counting from 0.
  final int startIndex;

  /// The viewer's state; a test passes one with fake service calls.
  final PresentationViewerController? controller;

  const PresentationViewerPage({
    required this.filePath,
    this.deviceSerial = '',
    this.startIndex = 0,
    this.controller,
    super.key,
  });

  @override
  State<PresentationViewerPage> createState() => _PresentationViewerPageState();
}

class _PresentationViewerPageState extends State<PresentationViewerPage> {
  late final PresentationViewerController _controller =
      widget.controller ??
      PresentationViewerController(
        path: widget.filePath,
        serial: serialOrNull(widget.deviceSerial),
        startIndex: widget.startIndex,
      );

  String get _name => widget.filePath.split('/').last;

  @override
  void initState() {
    super.initState();
    _controller.load();
  }

  @override
  void dispose() {
    if (widget.controller == null) _controller.dispose();
    super.dispose();
  }

  void _present() {
    final deck = _controller.deck;
    if (deck == null) return;
    context.go(
      AppRoutes.slidePresent(
        widget.filePath,
        serial: widget.deviceSerial,
        slide: _controller.index + 1,
      ),
      extra: deck,
    );
  }

  Future<void> _openAsQuarkSlides() => importPowerPointAndOpen(
    context,
    name: _name,
    run: _controller.importAsQuarkSlides,
    serial: widget.deviceSerial,
  );

  @override
  Widget build(BuildContext context) {
    // Nothing underneath is the usual case: Files opens the viewer at its own
    // URL, so leaving goes to the folder that holds the file (#1749).
    final canPop = context.canPop();
    void leave() => context.go(
      AppRoutes.containingFolder(widget.filePath, serial: widget.deviceSerial),
    );

    return PopScope(
      // A system back with nothing underneath would close the app.
      canPop: canPop,
      onPopInvokedWithResult: (didPop, _) {
        if (!didPop && !canPop) leave();
      },
      child: ListenableBuilder(
        listenable: _controller,
        builder: (context, _) => Scaffold(
          appBar: ChromeAppBar(
            title: Text(_name),
            leading: BackButton(onPressed: canPop ? context.pop : leave),
            actions: [
              Row(
                mainAxisSize: MainAxisSize.min,
                spacing: QuarkTokens.of(context).spacingSm,
                children: [
                  QuarkBarChip(
                    key: const ValueKey('presentation_viewer_present'),
                    icon: QuarkIcons.play_arrow,
                    label: 'Present',
                    tooltip: 'Present from this slide',
                    onPressed: _controller.slides.isEmpty ? null : _present,
                  ),
                  QuarkBarChip(
                    key: const ValueKey('presentation_viewer_open_as_qslide'),
                    icon: QuarkIcons.import_file,
                    label: 'Open as Quark slides',
                    tooltip: 'Save a copy as Quark slides and edit it',
                    onPressed: _openAsQuarkSlides,
                  ),
                  QuarkBarIconButton(
                    key: const ValueKey('presentation_viewer_download'),
                    icon: QuarkIcons.download_outlined,
                    tooltip: 'Download',
                    onPressed: () => downloadViewedFile(
                      context,
                      path: widget.filePath,
                      serial: serialOrNull(widget.deviceSerial),
                      name: _name,
                    ),
                  ),
                  const AppThemeToggle(),
                ],
              ),
            ],
          ),
          body: SafeArea(
            top: false,
            child: PresentationViewerBody(controller: _controller),
          ),
        ),
      ),
    );
  }
}
