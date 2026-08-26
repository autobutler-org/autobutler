import 'package:flutter/material.dart';

/// Top bar shown in place of `FileTopBar` while multi-select mode is active.
///
/// Like [FileTopBar], this is custom chrome rather than a real [AppBar], so it
/// has to consult the display insets itself: the [SafeArea] below is what keeps
/// the controls clear of the status bar, notch, or Dynamic Island (#1597). The
/// surface color lives on the outer container so the inset region is painted
/// rather than left showing whatever is behind the bar.
class FileSelectionBar extends StatelessWidget {
  const FileSelectionBar({
    required this.selectedCount,
    required this.totalCount,
    required this.onSelectAll,
    required this.onDeselectAll,
    required this.onCancel,
    this.onDelete,
    super.key,
  });

  final int selectedCount;
  final int totalCount;
  final VoidCallback onSelectAll;
  final VoidCallback onDeselectAll;
  final VoidCallback onCancel;
  final VoidCallback? onDelete;

  @override
  Widget build(BuildContext context) {
    final colors = Theme.of(context).colorScheme;
    return Container(
      color: colors.surfaceContainerHighest,
      // `bottom: false` because this bar only ever sits at the top of the
      // page; the left/right insets still apply, which is what keeps the
      // controls clear of the notch in landscape.
      child: SafeArea(
        bottom: false,
        child: SizedBox(
          height: 56,
          child: Padding(
            padding: const EdgeInsets.symmetric(horizontal: 8),
            child: Row(
              children: [
                IconButton(
                  icon: const Icon(Icons.close),
                  tooltip: 'Cancel selection',
                  onPressed: onCancel,
                ),
                Text(
                  '$selectedCount selected',
                  style: const TextStyle(fontWeight: FontWeight.w600),
                ),
                const Spacer(),
                TextButton(
                  onPressed: selectedCount < totalCount
                      ? onSelectAll
                      : onDeselectAll,
                  child: Text(
                    selectedCount < totalCount ? 'Select all' : 'Deselect all',
                  ),
                ),
                const SizedBox(width: 4),
                IconButton(
                  icon: Icon(
                    Icons.delete_outline,
                    color: onDelete != null
                        ? colors.error
                        : colors.onSurface.withValues(alpha: 0.38),
                  ),
                  tooltip: 'Delete selected',
                  onPressed: onDelete,
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
