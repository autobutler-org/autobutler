import 'package:flutter/material.dart'
    show
        StatelessWidget,
        Widget,
        MouseCursor,
        BuildContext,
        Border,
        Theme,
        BorderRadius,
        BoxDecoration,
        Container,
        MouseRegion;

class Cell extends StatelessWidget {
  final Widget child;
  final bool isActive;
  final bool isHighlighted;
  final MouseCursor cursor;

  const Cell({
    super.key,
    required this.child,
    required this.isActive,
    required this.isHighlighted,
    required this.cursor,
  });

  @override
  Widget build(BuildContext context) {
    final cs = Theme.of(context).colorScheme;
    return MouseRegion(
      cursor: cursor,
      child: Container(
        height: 40,
        decoration: BoxDecoration(
          color: isActive ? cs.primaryContainer : null,
          border: Border.all(
            color: (isActive || isHighlighted)
                ? cs.primary
                : cs.onSurface.withValues(alpha: 0.2),
            width: 1.0,
          ),
          borderRadius: BorderRadius.zero,
        ),
        child: child,
      ),
    );
  }
}
