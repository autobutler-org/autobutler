import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:quark/controllers/presentation_viewer_controller.dart';
import 'package:quark/widgets/presentation_viewer/presentation_viewer_stage.dart';
import 'package:quark/widgets/presentation_viewer/presentation_viewer_strip.dart';
import 'package:quark/widgets/presentation_viewer/presentation_viewer_warnings_notice.dart';
import 'package:quark/widgets/slides/slide_image.dart';
import 'package:quark/widgets/slides/slides_error_view.dart';
import 'package:quark_slides/quark_slides.dart';
import 'package:quark_widgets/quark_widgets.dart';

/// The presentation viewer's surface (#3268): the loader, the disconnected
/// view or the load error, and otherwise the warnings notice over the slide
/// strip and the [PresentationViewerStage].
///
/// The strip shows only on a window at least
/// [QuarkSplitView.collapseBreakpoint] wide; a phone steps through with the
/// arrows under the slide. The left and right arrow keys step too.
///
/// Pictures are [SlideImage]s fetched from inside the file through
/// [PresentationViewerController.imageUrl]; a picture that names anything
/// else is not drawn.
///
/// Key: `presentation_viewer_empty` on the message for a deck with no
/// slides.
class PresentationViewerBody extends StatelessWidget {
  /// Shows [controller]'s presentation.
  const PresentationViewerBody({required this.controller, super.key});

  /// The viewer's state; the caller rebuilds this widget when it changes.
  final PresentationViewerController controller;

  @override
  Widget build(BuildContext context) {
    final c = controller;
    final deck = c.deck;
    final error = c.error;
    if (deck == null) {
      if (error != null) {
        return SlidesErrorView(
          error: error,
          action: 'open the presentation',
          onRetry: c.load,
        );
      }
      return const Center(child: QuarkLoader());
    }
    if (deck.slides.isEmpty) {
      return const Center(
        key: ValueKey('presentation_viewer_empty'),
        child: Text('This presentation has no slides.'),
      );
    }

    Widget imageBuilder(BuildContext context, SlideImageSource image) {
      final url = c.imageUrl(image.source);
      return url == null
          ? const SizedBox.shrink()
          : SlideImage(image: NetworkImage(url.toString()), fit: image.fit);
    }

    final tokens = QuarkTokens.of(context);
    return CallbackShortcuts(
      bindings: {
        const SingleActivator(LogicalKeyboardKey.arrowLeft): c.previous,
        const SingleActivator(LogicalKeyboardKey.arrowRight): c.next,
      },
      child: Focus(
        autofocus: true,
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            if (c.warnings.isNotEmpty)
              PresentationViewerWarningsNotice(
                warnings: c.warnings,
                onDismiss: c.dismissWarnings,
              ),
            Expanded(
              child: Row(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  if (!QuarkSplitView.isCollapsed(context))
                    DecoratedBox(
                      decoration: BoxDecoration(
                        border: BorderDirectional(
                          end: BorderSide(color: tokens.border),
                        ),
                      ),
                      child: SizedBox(
                        width: 200,
                        child: PresentationViewerStrip(
                          deck: deck,
                          index: c.index,
                          onSelect: c.goTo,
                          imageBuilder: imageBuilder,
                        ),
                      ),
                    ),
                  Expanded(
                    child: PresentationViewerStage(
                      controller: c,
                      deck: deck,
                      imageBuilder: imageBuilder,
                    ),
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}
