import 'package:flutter/material.dart';
import 'package:quark/services/slides_service.dart';
import 'package:quark/widgets/slides/import/power_point_import_summary_dialog.dart';
import 'package:quark_icons/quark_icons.dart';
import 'package:quark_widgets/quark_widgets.dart';

/// What the presentation viewer could not show (#3268), above the slides
/// until it is dismissed: one line per kind of thing left out or
/// approximated ("SmartArt not shown"), with the slides it was on.
///
/// Keys: `presentation_viewer_warnings` on the notice,
/// `presentation_viewer_warning_<i>` on each line, in order, and
/// `presentation_viewer_warnings_dismiss` on its close button.
class PresentationViewerWarningsNotice extends StatelessWidget {
  /// A notice listing [warnings].
  const PresentationViewerWarningsNotice({
    required this.warnings,
    required this.onDismiss,
    super.key,
  });

  /// What the conversion left out or approximated; slide 0 is the deck.
  final List<PowerPointImportWarning> warnings;

  /// Hides the notice.
  final VoidCallback onDismiss;

  @override
  Widget build(BuildContext context) {
    final tokens = QuarkTokens.of(context);
    final textTheme = Theme.of(context).textTheme;
    final groups = PowerPointImportSummaryDialog.grouped(warnings);
    return Container(
      key: const ValueKey('presentation_viewer_warnings'),
      margin: EdgeInsets.all(tokens.spacingSm),
      padding: EdgeInsets.all(tokens.spacingSm),
      // A deck with many kinds of warning would otherwise push the slide
      // off a phone's screen.
      constraints: const BoxConstraints(maxHeight: 160),
      decoration: BoxDecoration(
        color: tokens.card,
        borderRadius: BorderRadius.circular(tokens.radiusMd),
        border: Border.all(color: tokens.border),
      ),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        spacing: tokens.spacingSm,
        children: [
          Icon(QuarkIcons.info_outline, color: tokens.mutedForeground),
          Expanded(
            child: SingleChildScrollView(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                spacing: tokens.spacingXs,
                children: [
                  Text(
                    'Some of this presentation is left out or simplified here.',
                    style: textTheme.bodyMedium,
                  ),
                  for (final (i, g) in groups.indexed)
                    Text(
                      '${g.message} (${PowerPointImportSummaryDialog.where(g.slides)})',
                      key: ValueKey('presentation_viewer_warning_$i'),
                      style: textTheme.bodySmall?.copyWith(
                        color: tokens.mutedForeground,
                      ),
                    ),
                ],
              ),
            ),
          ),
          QuarkBarIconButton(
            key: const ValueKey('presentation_viewer_warnings_dismiss'),
            icon: QuarkIcons.close,
            tooltip: 'Dismiss',
            onPressed: onDismiss,
          ),
        ],
      ),
    );
  }
}
