import 'package:flutter/material.dart';
import 'package:quark/widgets/presentation_viewer/presentation_viewer_thumbnail.dart';
import 'package:quark_slides/quark_slides.dart';
import 'package:quark_widgets/quark_widgets.dart';

/// The presentation viewer's column of slide thumbnails (#3268), built
/// lazily so a long deck draws only what is on screen. Tapping one shows it.
///
/// Key: `presentation_viewer_strip` on the list; each thumbnail's is
/// [PresentationViewerThumbnail]'s.
class PresentationViewerStrip extends StatelessWidget {
  /// A strip of [deck]'s slides with the one at [index] selected.
  const PresentationViewerStrip({
    required this.deck,
    required this.index,
    required this.onSelect,
    this.imageBuilder,
    super.key,
  });

  /// The presentation whose slides are listed.
  final Presentation deck;

  /// The position of the slide showing, counting from 0.
  final int index;

  /// Called with a slide's position when its thumbnail is tapped.
  final ValueChanged<int> onSelect;

  /// Draws image elements and background images.
  final SlideImageBuilder? imageBuilder;

  @override
  Widget build(BuildContext context) {
    final tokens = QuarkTokens.of(context);
    return ListView.builder(
      key: const ValueKey('presentation_viewer_strip'),
      padding: EdgeInsets.all(tokens.spacingSm),
      itemCount: deck.slides.length,
      itemBuilder: (context, i) => PresentationViewerThumbnail(
        slide: deck.slides[i],
        index: i,
        size: deck.size,
        theme: deck.theme,
        selected: i == index,
        onSelect: () => onSelect(i),
        imageBuilder: imageBuilder,
      ),
    );
  }
}
