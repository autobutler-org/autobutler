import 'package:flutter/material.dart';

class FileBrowserHeader extends StatelessWidget {
  const FileBrowserHeader({required this.isGridView, super.key});

  final bool isGridView;

  @override
  Widget build(BuildContext context) {
    if (isGridView) {
      // Effectively empty
      return const SizedBox.shrink();
    }
    final colors = Theme.of(context).colorScheme;

    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
      decoration: BoxDecoration(
        color: colors.surface,
        border: Border(
          top: BorderSide(color: colors.outlineVariant.withValues(alpha: 0.6)),
          bottom: BorderSide(
            color: colors.outlineVariant.withValues(alpha: 0.6),
          ),
        ),
      ),
      child: const Row(
        children: [
          Expanded(flex: 6, child: Text('Name')),
          Expanded(flex: 2, child: Text('Device')),
          Expanded(flex: 2, child: Text('Size')),
        ],
      ),
    );
  }
}
