import 'package:flutter/material.dart';

/// The small upper-case label that separates one group of items from the next
/// inside the top bar's Views menu.
class TopBarMenuSectionHeader extends StatelessWidget {
  const TopBarMenuSectionHeader({required this.title, super.key});

  final String title;

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;
    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 8, 16, 4),
      child: Text(
        title.toUpperCase(),
        style: TextStyle(
          fontSize: 11,
          fontWeight: FontWeight.w600,
          letterSpacing: 0.8,
          color: colorScheme.onSurface.withValues(alpha: 0.45),
        ),
      ),
    );
  }
}
