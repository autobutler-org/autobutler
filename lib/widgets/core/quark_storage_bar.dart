import 'package:quark_widgets/quark_widgets.dart';
import 'package:flutter/material.dart';

class QuarkStorageBar extends StatelessWidget {
  const QuarkStorageBar({
    required this.usedFraction,
    this.height = 8.0,
    super.key,
  });

  final double usedFraction;
  final double height;

  static Color colorForFraction(double fraction) {
    if (fraction >= 0.9) return QuarkColors.error;
    if (fraction >= 0.75) return const Color(0xFFF59E0B);
    return QuarkColors.primary;
  }

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;
    final clamped = usedFraction.clamp(0.0, 1.0);
    final color = colorForFraction(clamped);
    return Container(
      height: height,
      decoration: BoxDecoration(
        color: colorScheme.surfaceContainerHighest,
        border: Border.all(color: colorScheme.outline),
        borderRadius: BorderRadius.circular(QuarkColors.radiusMd),
      ),
      clipBehavior: Clip.antiAlias,
      child: AnimatedFractionallySizedBox(
        duration: const Duration(milliseconds: 400),
        curve: Curves.easeOut,
        alignment: Alignment.centerLeft,
        widthFactor: clamped,
        child: Container(
          decoration: BoxDecoration(
            color: color,
            borderRadius: BorderRadius.circular(QuarkColors.radiusMd),
          ),
        ),
      ),
    );
  }
}
