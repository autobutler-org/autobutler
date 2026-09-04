import 'package:flutter/material.dart';
import 'package:quark_icons/quark_icons.dart';

/// What a photo thumbnail wears while the grid is in selection mode: a dim
/// over anything unselected, a border around anything selected, and a checkbox
/// in the corner.
///
/// It also carries the gestures, so tapping a dimmed thumbnail selects it
/// rather than opening it.
class PhotoSelectionOverlay extends StatelessWidget {
  /// Wraps [child] in the selection chrome.
  const PhotoSelectionOverlay({
    required this.child,
    required this.isSelected,
    required this.selectionMode,
    required this.onToggleSelection,
    required this.onEnterSelectionMode,
    super.key,
  });

  /// The thumbnail underneath.
  final Widget child;

  /// Whether this photo is in the selection.
  final bool isSelected;

  /// Whether the grid is in selection mode. False still renders the gestures,
  /// so a long press can enter selection mode with this photo selected.
  final bool selectionMode;

  /// Adds this photo to the selection, or takes it out.
  final VoidCallback onToggleSelection;

  /// Enters selection mode, on a long press outside it.
  final VoidCallback onEnterSelectionMode;

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;

    return GestureDetector(
      onTap: onToggleSelection,
      onLongPress: () {
        if (!selectionMode) onEnterSelectionMode();
        onToggleSelection();
      },
      child: Stack(
        fit: StackFit.expand,
        children: [
          child,
          // Dim overlay for unselected
          if (selectionMode && !isSelected)
            Container(color: Colors.black.withValues(alpha: 0.3)),
          // Teal border for selected
          if (isSelected)
            DecoratedBox(
              decoration: BoxDecoration(
                border: Border.all(color: colorScheme.primary, width: 3),
              ),
            ),
          // Checkbox in top-left
          if (selectionMode)
            Positioned(
              top: 6,
              left: 6,
              child: AnimatedContainer(
                duration: const Duration(milliseconds: 150),
                width: 22,
                height: 22,
                decoration: BoxDecoration(
                  shape: BoxShape.circle,
                  color: isSelected ? colorScheme.primary : Colors.transparent,
                  border: Border.all(
                    color: isSelected
                        ? colorScheme.primary
                        : Colors.white.withValues(alpha: 0.8),
                    width: 2,
                  ),
                ),
                child: isSelected
                    ? const Icon(
                        QuarkIcons.check,
                        size: 14,
                        color: Colors.white,
                      )
                    : null,
              ),
            ),
        ],
      ),
    );
  }
}
