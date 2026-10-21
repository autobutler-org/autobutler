import 'package:flutter/material.dart';
import 'package:quark/widgets/slides/notes/slide_notes_view.dart';
import 'package:quark_widgets/quark_widgets.dart';

/// The current slide's speaker notes in the presentation viewer (#3268), in
/// a bordered panel under the slide that scrolls when they run long.
///
/// Key: `presentation_viewer_notes_panel`.
class PresentationViewerNotes extends StatelessWidget {
  /// A panel showing [notes].
  const PresentationViewerNotes({required this.notes, super.key});

  /// The notes, plain text; empty when the slide has none.
  final String notes;

  @override
  Widget build(BuildContext context) {
    final tokens = QuarkTokens.of(context);
    return Container(
      key: const ValueKey('presentation_viewer_notes_panel'),
      width: double.infinity,
      padding: EdgeInsets.all(tokens.spacingMd),
      decoration: BoxDecoration(
        color: tokens.card,
        border: Border(top: BorderSide(color: tokens.border)),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        spacing: tokens.spacingSm,
        children: [
          Text('Notes', style: Theme.of(context).textTheme.titleSmall),
          Flexible(child: SlideNotesView(notes: notes)),
        ],
      ),
    );
  }
}
