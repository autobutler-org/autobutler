import 'package:flutter/material.dart';
import 'package:quark/controllers/presentation_viewer_controller.dart';
import 'package:quark/widgets/presentation_viewer/presentation_viewer_nav.dart';
import 'package:quark/widgets/presentation_viewer/presentation_viewer_notes.dart';
import 'package:quark/widgets/slides/slide_editor_canvas.dart';
import 'package:quark_slides/quark_slides.dart';
import 'package:quark_widgets/quark_widgets.dart';

/// The slide showing in the presentation viewer (#3268), drawn read-only
/// and fitted to the space there is, with the [PresentationViewerNav] under
/// it and, when toggled on, its speaker notes under that.
///
/// It reads [controller] and calls its commands; it never loads anything.
///
/// Key: `presentation_viewer_slide` on the slide.
class PresentationViewerStage extends StatelessWidget {
  /// The stage for [controller]'s [deck], which has at least one slide.
  const PresentationViewerStage({
    required this.controller,
    required this.deck,
    this.imageBuilder,
    super.key,
  });

  /// The viewer's state; the caller rebuilds this widget when it changes.
  final PresentationViewerController controller;

  /// The presentation being read.
  final Presentation deck;

  /// Draws image elements and background images.
  final SlideImageBuilder? imageBuilder;

  @override
  Widget build(BuildContext context) {
    final tokens = QuarkTokens.of(context);
    final c = controller;
    final slide = c.currentSlide!;
    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Expanded(
          child: Padding(
            padding: EdgeInsets.all(tokens.spacingMd),
            child: Center(
              child: Semantics(
                label: c.position,
                child: SlideCanvas.readOnly(
                  key: const ValueKey('presentation_viewer_slide'),
                  slide: slide,
                  size: deck.size,
                  theme: deck.theme,
                  imageBuilder: imageBuilder,
                  style: SlideEditorCanvas.styleOf(context),
                ),
              ),
            ),
          ),
        ),
        PresentationViewerNav(
          position: c.position,
          notesVisible: c.notesVisible,
          onPrevious: c.isFirst ? null : c.previous,
          onNext: c.isLast ? null : c.next,
          onToggleNotes: c.toggleNotes,
        ),
        if (c.notesVisible)
          ConstrainedBox(
            constraints: const BoxConstraints(maxHeight: 160),
            child: PresentationViewerNotes(notes: slide.notes),
          ),
      ],
    );
  }
}
