import 'package:flutter/material.dart';
import 'package:quark_icons/quark_icons.dart';

import '../theme/quark_tokens.dart';

/// The strip between a file browser's chrome and its listing.
///
/// Today it carries one thing: the search result summary, with a close button
/// that leaves search mode. Outside search mode it renders nothing, because
/// the sort headers live inside the listing itself.
///
/// The result count is an input rather than a future the widget awaits — the
/// page owns the load, and a null count is the "still counting" state.
///
/// Key prefixes: `file_header_close_search` on the close button.
///
/// ```dart
/// FileBrowserHeader(
///   isSearchMode: true,
///   searchQuery: 'invoice',
///   resultCount: results?.length,
///   onClose: controller.exitSearch,
/// );
/// ```
class FileBrowserHeader extends StatelessWidget {
  /// Creates the header strip.
  const FileBrowserHeader({
    required this.isSearchMode,
    this.resultCount,
    this.searchQuery,
    this.onClose,
    super.key,
  });

  /// Whether the listing is showing search results. False renders nothing.
  final bool isSearchMode;

  /// How many results the search returned, or null while it is still running,
  /// which shows a count of zero rather than a spinner.
  final int? resultCount;

  /// The query the results are for, quoted back to the user.
  final String? searchQuery;

  /// Leaves search mode. The close button is inert when null.
  final VoidCallback? onClose;

  @override
  Widget build(BuildContext context) {
    if (!isSearchMode) {
      // Sort headers are rendered inside the listing.
      return const SizedBox.shrink();
    }

    final colors = Theme.of(context).colorScheme;
    final tokens = QuarkTokens.of(context);
    final count = resultCount ?? 0;
    final query = searchQuery ?? '';

    return Container(
      padding: EdgeInsets.symmetric(
        horizontal: tokens.spacingMd,
        vertical: tokens.spacingSm + tokens.spacingXs / 2,
      ),
      decoration: BoxDecoration(
        color: colors.surface,
        border: Border(
          top: BorderSide(color: colors.outlineVariant.withValues(alpha: 0.6)),
          bottom: BorderSide(
            color: colors.outlineVariant.withValues(alpha: 0.6),
          ),
        ),
      ),
      child: Row(
        children: [
          Expanded(
            child: Text(
              "$count result${count == 1 ? '' : 's'} for '$query'",
              style: Theme.of(context).textTheme.bodyMedium,
            ),
          ),
          IconButton(
            key: const ValueKey('file_header_close_search'),
            onPressed: onClose,
            icon: const Icon(QuarkIcons.close),
            tooltip: 'Close search',
          ),
        ],
      ),
    );
  }
}
