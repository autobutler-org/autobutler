import 'package:flutter/material.dart';
import 'package:quark_icons/quark_icons.dart';
import 'package:quark_widgets/quark_widgets.dart';

/// The row under the slide in the presentation viewer (#3268): previous,
/// where the reader is, next, and the speaker notes toggle.
///
/// Keys: `presentation_viewer_previous`, `presentation_viewer_position`,
/// `presentation_viewer_next` and `presentation_viewer_notes`.
class PresentationViewerNav extends StatelessWidget {
  /// The row for a reader at [position].
  const PresentationViewerNav({
    required this.position,
    required this.notesVisible,
    required this.onToggleNotes,
    this.onPrevious,
    this.onNext,
    super.key,
  });

  /// Where the reader is, such as "Slide 2 of 5".
  final String position;

  /// Whether the speaker notes are showing.
  final bool notesVisible;

  /// Shows the previous slide; null at the first, which disables it.
  final VoidCallback? onPrevious;

  /// Shows the next slide; null at the last, which disables it.
  final VoidCallback? onNext;

  /// Shows or hides the speaker notes.
  final VoidCallback onToggleNotes;

  @override
  Widget build(BuildContext context) {
    final tokens = QuarkTokens.of(context);
    return Padding(
      padding: EdgeInsets.all(tokens.spacingSm),
      child: Row(
        mainAxisAlignment: MainAxisAlignment.center,
        spacing: tokens.spacingSm,
        children: [
          QuarkBarIconButton(
            key: const ValueKey('presentation_viewer_previous'),
            icon: QuarkIcons.chevron_left,
            tooltip: 'Previous slide',
            onPressed: onPrevious,
          ),
          Flexible(
            child: Text(
              position,
              key: const ValueKey('presentation_viewer_position'),
              textAlign: TextAlign.center,
              style: Theme.of(context).textTheme.bodyMedium,
            ),
          ),
          QuarkBarIconButton(
            key: const ValueKey('presentation_viewer_next'),
            icon: QuarkIcons.chevron_right,
            tooltip: 'Next slide',
            onPressed: onNext,
          ),
          QuarkBarIconButton(
            key: const ValueKey('presentation_viewer_notes'),
            icon: QuarkIcons.notes_rounded,
            tooltip: notesVisible ? 'Hide speaker notes' : 'Show speaker notes',
            selected: notesVisible,
            onPressed: onToggleNotes,
          ),
        ],
      ),
    );
  }
}
