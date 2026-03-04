import 'package:flutter/material.dart';
import 'package:autobutler/models/cirrus_file_node.dart';

class FileBrowserHeader extends StatelessWidget {
  const FileBrowserHeader({
    required this.isGridView,
    required this.isSearchMode,
    this.filesFuture,
    this.searchQuery,
    super.key,
  });

  final bool isGridView;
  final bool isSearchMode;
  final Future<List<CirrusFileNode>>? filesFuture;
  final String? searchQuery;

  @override
  Widget build(BuildContext context) {
    final colors = Theme.of(context).colorScheme;

    if (isSearchMode) {
      return Container(
        padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 10),
        decoration: BoxDecoration(
          color: colors.surface,
          border: Border(
            top: BorderSide(
              color: colors.outlineVariant.withValues(alpha: 0.6),
            ),
            bottom: BorderSide(
              color: colors.outlineVariant.withValues(alpha: 0.6),
            ),
          ),
        ),
        child: FutureBuilder<List<CirrusFileNode>>(
          future: filesFuture ?? Future.value(const <CirrusFileNode>[]),
          builder: (context, snapshot) {
            final count = snapshot.hasData ? snapshot.data!.length : 0;
            final query = searchQuery ?? '';
            return Row(
              children: [
                Expanded(
                  child: Text(
                    "$count result${count == 1 ? '' : 's'} for '$query'",
                    style: Theme.of(context).textTheme.bodyMedium,
                  ),
                ),
              ],
            );
          },
        ),
      );
    }

    if (isGridView) {
      // Effectively empty in grid view
      return const SizedBox.shrink();
    }

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
