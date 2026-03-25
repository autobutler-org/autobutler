import 'package:autobutler/theme/autobutler_colors.dart';
import 'package:flutter/material.dart';

/// A reusable storage usage bar with consistent color thresholds.
///
/// Color logic:
/// - `usedFraction < 0.75` → primary (green)
/// - `0.75 ≤ usedFraction < 0.9` → amber (warning)
/// - `usedFraction ≥ 0.9` → error (red)
class AutobutlerStorageBar extends StatelessWidget {
  const AutobutlerStorageBar({
    required this.usedFraction,
    this.height = 8.0,
    this.animated = true,
    super.key,
  });

  /// Storage used as a fraction from 0.0 to 1.0.
  final double usedFraction;

  /// Height of the bar in logical pixels.
  final double height;

  /// Whether to animate changes to the bar width.
  final bool animated;

  /// Returns the canonical bar color for the given usage fraction.
  static Color colorForFraction(double fraction) {
    if (fraction >= 0.9) return AutobutlerColors.error;
    if (fraction >= 0.75) return const Color(0xFFF59E0B); // amber
    return AutobutlerColors.primary;
  }

  @override
  Widget build(BuildContext context) {
    final clamped = usedFraction.clamp(0.0, 1.0);
    final color = colorForFraction(clamped);

    return Container(
      height: height,
      decoration: BoxDecoration(
        color: AutobutlerColors.input,
        border: Border.all(color: AutobutlerColors.border),
        borderRadius: BorderRadius.circular(AutobutlerColors.radiusMd),
      ),
      clipBehavior: Clip.antiAlias,
      child: animated
          ? AnimatedFractionallySizedBox(
              duration: const Duration(milliseconds: 400),
              curve: Curves.easeOut,
              alignment: Alignment.centerLeft,
              widthFactor: clamped,
              child: Container(
                decoration: BoxDecoration(
                  color: color,
                  borderRadius: BorderRadius.circular(
                    AutobutlerColors.radiusMd,
                  ),
                ),
              ),
            )
          : FractionallySizedBox(
              alignment: Alignment.centerLeft,
              widthFactor: clamped,
              child: Container(
                decoration: BoxDecoration(
                  color: color,
                  borderRadius: BorderRadius.circular(
                    AutobutlerColors.radiusMd,
                  ),
                ),
              ),
            ),
    );
  }
}
