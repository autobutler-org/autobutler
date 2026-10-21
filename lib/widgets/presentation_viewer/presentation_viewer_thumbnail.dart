import 'package:flutter/material.dart';
import 'package:quark/widgets/slides/slide_editor_canvas.dart';
import 'package:quark_slides/quark_slides.dart';
import 'package:quark_widgets/quark_widgets.dart';

/// One slide in the presentation viewer's strip (#3268): the slide drawn by
/// a read-only [SlideCanvas] under its number, the one showing outlined.
/// Unlike the editor's thumbnail it has no menu: the viewer changes nothing.
///
/// Key: `presentation_viewer_thumb_<index>`, counting from 0, on the
/// tappable thumbnail.
class PresentationViewerThumbnail extends StatelessWidget {
  /// The thumbnail of [slide], the slide at [index] in the show.
  const PresentationViewerThumbnail({
    required this.slide,
    required this.index,
    required this.size,
    required this.selected,
    required this.onSelect,
    this.theme,
    this.imageBuilder,
    super.key,
  });

  /// The slide to draw.
  final Slide slide;

  /// The slide's position in the show, counting from 0.
  final int index;

  /// The presentation's slide size.
  final SlideSize size;

  /// Whether this is the slide showing.
  final bool selected;

  /// Called when the thumbnail is tapped.
  final VoidCallback onSelect;

  /// The presentation's theme, which the slide is drawn in; null for none.
  final SlideTheme? theme;

  /// Draws image elements and background images.
  final SlideImageBuilder? imageBuilder;

  @override
  Widget build(BuildContext context) {
    final tokens = QuarkTokens.of(context);
    final number = index + 1;
    return Semantics(
      button: true,
      selected: selected,
      label: 'Slide $number',
      excludeSemantics: true,
      child: InkWell(
        key: ValueKey('presentation_viewer_thumb_$index'),
        onTap: onSelect,
        borderRadius: BorderRadius.circular(tokens.radiusMd),
        child: Padding(
          padding: EdgeInsets.all(tokens.spacingXs),
          child: Row(
            crossAxisAlignment: CrossAxisAlignment.start,
            spacing: tokens.spacingXs,
            children: [
              SizedBox(
                width: tokens.spacingLg,
                child: Text(
                  '$number',
                  textAlign: TextAlign.end,
                  style: Theme.of(context).textTheme.labelSmall?.copyWith(
                    color: selected ? tokens.primary : tokens.mutedForeground,
                  ),
                ),
              ),
              Expanded(
                child: DecoratedBox(
                  position: DecorationPosition.foreground,
                  decoration: BoxDecoration(
                    borderRadius: BorderRadius.circular(tokens.radiusSm),
                    border: Border.all(
                      color: selected ? tokens.primary : tokens.border,
                      width: selected ? 2 : 1,
                    ),
                  ),
                  child: AspectRatio(
                    aspectRatio: size.aspectRatio,
                    child: RepaintBoundary(
                      child: SlideCanvas.readOnly(
                        slide: slide,
                        size: size,
                        theme: theme,
                        imageBuilder: imageBuilder,
                        style: SlideEditorCanvas.styleOf(context),
                      ),
                    ),
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
