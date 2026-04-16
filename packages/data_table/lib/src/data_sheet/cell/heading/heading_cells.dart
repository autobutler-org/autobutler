import 'package:flutter/material.dart';

const double kGutterWidth = 48.0;
const double kHeaderHeight = 28.0;

class HeaderCornerCell extends StatelessWidget {
  const HeaderCornerCell({super.key});

  @override
  Widget build(BuildContext context) {
    final cs = Theme.of(context).colorScheme;
    return Container(
      width: kGutterWidth,
      height: kHeaderHeight,
      decoration: BoxDecoration(
        color: cs.surfaceContainerHighest,
        border: Border.all(color: cs.onSurface.withValues(alpha: 0.2)),
      ),
    );
  }
}

class ColumnHeaderCell extends StatelessWidget {
  final String label;

  const ColumnHeaderCell({super.key, required this.label});

  @override
  Widget build(BuildContext context) {
    final cs = Theme.of(context).colorScheme;
    return Container(
      height: kHeaderHeight,
      decoration: BoxDecoration(
        color: cs.surfaceContainerHighest,
        border: Border.all(color: cs.onSurface.withValues(alpha: 0.2)),
      ),
      alignment: Alignment.center,
      child: Text(
        label,
        style: TextStyle(
          fontSize: 12,
          fontWeight: FontWeight.w600,
          color: cs.onSurface,
        ),
        overflow: TextOverflow.clip,
      ),
    );
  }
}

class RowNumberCell extends StatelessWidget {
  final int number;

  const RowNumberCell({super.key, required this.number});

  @override
  Widget build(BuildContext context) {
    final cs = Theme.of(context).colorScheme;
    return Container(
      width: kGutterWidth,
      height: 40,
      decoration: BoxDecoration(
        color: cs.surfaceContainerHighest,
        border: Border.all(color: cs.onSurface.withValues(alpha: 0.2)),
      ),
      alignment: Alignment.center,
      child: Text(
        '$number',
        style: TextStyle(
          fontSize: 12,
          fontWeight: FontWeight.w500,
          color: cs.onSurface,
        ),
      ),
    );
  }
}
