import 'package:flutter/material.dart';
import 'package:quark_widgets/quark_widgets.dart';

/// One choice in a [TopBarSegmentedToggle].
typedef TopBarSegment = ({IconData icon, String label});

/// A joined row of mutually exclusive choices. The selected segment is inert —
/// tapping what is already on is not a state change worth an ink splash.
class TopBarSegmentedToggle extends StatelessWidget {
  const TopBarSegmentedToggle({
    required this.segments,
    required this.selectedIndex,
    required this.onSelected,
    super.key,
  });

  final List<TopBarSegment> segments;
  final int selectedIndex;
  final ValueChanged<int> onSelected;

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;
    final radius = BorderRadius.circular(QuarkColors.radiusLg);
    return Material(
      color: colorScheme.surfaceContainerHighest,
      shape: RoundedRectangleBorder(
        side: BorderSide(color: colorScheme.outline),
        borderRadius: radius,
      ),
      clipBehavior: Clip.antiAlias,
      child: Row(
        mainAxisSize: MainAxisSize.min,
        children: List.generate(segments.length, (i) {
          final seg = segments[i];
          final isActive = i == selectedIndex;
          final isFirst = i == 0;
          final isLast = i == segments.length - 1;

          BorderRadius segRadius;
          if (isFirst && isLast) {
            segRadius = radius;
          } else if (isFirst) {
            segRadius = BorderRadius.only(
              topLeft: Radius.circular(QuarkColors.radiusLg),
              bottomLeft: Radius.circular(QuarkColors.radiusLg),
            );
          } else if (isLast) {
            segRadius = BorderRadius.only(
              topRight: Radius.circular(QuarkColors.radiusLg),
              bottomRight: Radius.circular(QuarkColors.radiusLg),
            );
          } else {
            segRadius = BorderRadius.zero;
          }

          return MouseRegion(
            cursor: isActive
                ? SystemMouseCursors.basic
                : SystemMouseCursors.click,
            child: Tooltip(
              message: seg.label,
              child: InkWell(
                onTap: isActive ? null : () => onSelected(i),
                borderRadius: segRadius,
                child: Container(
                  padding: const EdgeInsets.symmetric(
                    horizontal: 10,
                    vertical: 6,
                  ),
                  decoration: BoxDecoration(
                    color: isActive
                        ? colorScheme.primary.withValues(alpha: 0.12)
                        : Colors.transparent,
                    border: i > 0
                        ? Border(left: BorderSide(color: colorScheme.outline))
                        : null,
                  ),
                  child: Row(
                    mainAxisSize: MainAxisSize.min,
                    children: [
                      Icon(
                        seg.icon,
                        size: 14,
                        color: isActive
                            ? colorScheme.primary
                            : colorScheme.onSurfaceVariant,
                      ),
                      const SizedBox(width: 6),
                      Text(
                        seg.label,
                        style: TextStyle(
                          fontSize: 13,
                          color: isActive
                              ? colorScheme.primary
                              : colorScheme.onSurfaceVariant,
                        ),
                      ),
                    ],
                  ),
                ),
              ),
            ),
          );
        }),
      ),
    );
  }
}
