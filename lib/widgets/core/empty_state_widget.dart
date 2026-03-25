import 'package:autobutler/theme/autobutler_colors.dart';
import 'package:flutter/material.dart';

class EmptyStateWidget extends StatelessWidget {
  const EmptyStateWidget({
    required this.icon,
    required this.headline,
    this.subtext,
    this.action,
    super.key,
  });

  final IconData icon;
  final String headline;
  final String? subtext;
  final Widget? action;

  @override
  Widget build(BuildContext context) {
    return Center(
      child: Padding(
        padding: const EdgeInsets.symmetric(horizontal: 32, vertical: 24),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(icon, size: 56, color: AutobutlerColors.mutedForeground),
            const SizedBox(height: 16),
            Text(
              headline,
              textAlign: TextAlign.center,
              style: const TextStyle(
                fontSize: 17,
                fontWeight: FontWeight.w600,
                color: AutobutlerColors.cardForeground,
              ),
            ),
            if (subtext != null) ...[
              const SizedBox(height: 8),
              Text(
                subtext!,
                textAlign: TextAlign.center,
                style: const TextStyle(
                  fontSize: 14,
                  color: AutobutlerColors.mutedForeground,
                  height: 1.5,
                ),
              ),
            ],
            if (action != null) ...[const SizedBox(height: 20), action!],
          ],
        ),
      ),
    );
  }
}
