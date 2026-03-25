import 'package:autobutler/theme/autobutler_colors.dart';
import 'package:flutter/material.dart';

class AutobutlerStorageBar extends StatelessWidget {
  const AutobutlerStorageBar({
    required this.usedFraction,
    this.height = 8.0,
    super.key,
  });

  final double usedFraction;
  final double height;

  static Color colorForFraction(double fraction) {
    if (fraction >= 0.9) return AutobutlerColors.error;
    if (fraction >= 0.75) return const Color(0xFFF59E0B);
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
      child: AnimatedFractionallySizedBox(
        duration: const Duration(milliseconds: 400),
        curve: Curves.easeOut,
        alignment: Alignment.centerLeft,
        widthFactor: clamped,
        child: Container(
          decoration: BoxDecoration(
            color: color,
            borderRadius: BorderRadius.circular(AutobutlerColors.radiusMd),
          ),
        ),
      ),
    );
  }
}
