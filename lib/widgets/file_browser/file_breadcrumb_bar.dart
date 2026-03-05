import 'package:flutter/material.dart';

class FileBreadcrumbBar extends StatelessWidget {
  const FileBreadcrumbBar({
    required this.currentPath,
    required this.onGoHome,
    required this.onGoUp,
    required this.onPathSelected,
    required this.isSearchMode,
    super.key,
  });

  final String currentPath;
  final VoidCallback onGoHome;
  final VoidCallback onGoUp;
  final ValueChanged<String> onPathSelected;
  final bool isSearchMode;

  @override
  Widget build(BuildContext context) {
    if (isSearchMode) {
      return const SizedBox.shrink();
    }
    return Padding(
      padding: const EdgeInsets.fromLTRB(16, 0, 16, 8),
      child: Row(
        children: [
          IconButton(
            onPressed: currentPath.isEmpty ? null : onGoUp,
            icon: const Icon(Icons.chevron_left_rounded),
            tooltip: 'Up one level',
          ),
          Expanded(
            child: SingleChildScrollView(
              scrollDirection: Axis.horizontal,
              child: Row(
                children: [
                  GestureDetector(
                    onTap: onGoHome,
                    child: Padding(
                      padding: const EdgeInsets.symmetric(horizontal: 4),
                      child: Icon(
                        Icons.home_rounded,
                        size: 20,
                        color: Theme.of(context).colorScheme.primary,
                      ),
                    ),
                  ),
                  ..._buildBreadcrumbs(context),
                ],
              ),
            ),
          ),
        ],
      ),
    );
  }

  List<Widget> _buildBreadcrumbs(BuildContext context) {
    final style = Theme.of(context).textTheme.titleMedium;
    if (currentPath.isEmpty) {
      return [Text('/', style: style)];
    }

    final segments = currentPath.substring(1).split('/');
    final children = <Widget>[
      Padding(
        padding: const EdgeInsets.only(right: 4),
        child: Text('/', style: style),
      ),
    ];

    for (var index = 0; index < segments.length; index++) {
      if (index > 0) {
        children.add(
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 4),
            child: Text('/', style: style),
          ),
        );
      }

      final segment = segments[index];
      final isLast = index == segments.length - 1;

      if (isLast) {
        children.add(
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 4),
            child: Text(segment, style: style),
          ),
        );
        continue;
      }

      final targetPath = '/${segments.take(index + 1).join('/')}';
      children.add(
        GestureDetector(
          onTap: () => onPathSelected(targetPath),
          child: Padding(
            padding: const EdgeInsets.symmetric(horizontal: 4),
            child: Text(
              segment,
              style: style?.copyWith(
                color: Theme.of(context).colorScheme.primary,
              ),
            ),
          ),
        ),
      );
    }

    return children;
  }
}
