import 'package:flutter/material.dart'
    show
        StatelessWidget,
        Widget,
        MouseCursor,
        BuildContext,
        Colors,
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
    const borderWidth = 1.0;
    return MouseRegion(
      cursor: cursor,
      child: Container(
        height: 40,
        decoration: BoxDecoration(
          color: isActive ? Colors.grey.shade300 : null,
          border: Border.all(
            color: (isActive || isHighlighted)
                ? Theme.of(context).colorScheme.primary
                : Colors.grey.shade400,
            width: borderWidth * ((isActive || isHighlighted) ? 2 : 1),
          ),
          borderRadius: BorderRadius.zero,
        ),
        child: child,
      ),
    );
  }
}
