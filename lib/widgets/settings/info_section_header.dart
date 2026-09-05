import 'package:flutter/material.dart';
import 'package:quark_icons/quark_icons.dart';

/// Section header for read-only informational sections.
/// Uses a subtler visual treatment than action-oriented sections to signal
/// that the content is reference material, not something the user configures.
class InfoSectionHeader extends StatelessWidget {
  const InfoSectionHeader({super.key, required this.label});

  final String label;

  @override
  Widget build(BuildContext context) {
    final color = Theme.of(context).colorScheme.onSurfaceVariant;
    return Row(
      children: [
        Icon(QuarkIcons.info_outline, size: 16, color: color),
        const SizedBox(width: 6),
        Text(
          label,
          style: TextStyle(
            fontSize: 14,
            fontWeight: FontWeight.w600,
            color: color,
            letterSpacing: 0.3,
          ),
        ),
      ],
    );
  }
}
