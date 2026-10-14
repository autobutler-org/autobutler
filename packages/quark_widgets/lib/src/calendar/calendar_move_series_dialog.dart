import 'package:flutter/material.dart';

/// Asks before a drag or a key press moves a repeating event, which moves
/// every time it repeats, not the one occurrence moved (#3184).
///
/// It does not close itself: [onConfirm] and [onCancel] fire, and the caller
/// that pushed the dialog pops it. The copy scrolls when the window is too
/// short for it, so the buttons stay reachable.
///
/// Key prefixes: `calendar_move_series_confirm` and
/// `calendar_move_series_cancel`.
///
/// ```dart
/// showDialog<bool>(
///   context: context,
///   builder: (ctx) => CalendarMoveSeriesDialog(
///     title: 'Soccer practice',
///     onConfirm: () => Navigator.of(ctx).pop(true),
///     onCancel: () => Navigator.of(ctx).pop(false),
///   ),
/// );
/// ```
class CalendarMoveSeriesDialog extends StatelessWidget {
  /// Creates the question for the event titled [title].
  const CalendarMoveSeriesDialog({
    required this.title,
    required this.onConfirm,
    required this.onCancel,
    super.key,
  });

  /// The repeating event's title.
  final String title;

  /// Called when "Move all" is tapped.
  final VoidCallback onConfirm;

  /// Called when "Cancel" is tapped.
  final VoidCallback onCancel;

  @override
  Widget build(BuildContext context) => AlertDialog(
    scrollable: true,
    title: Text(
      title.isEmpty ? 'Move every time it repeats?' : 'Move all of $title?',
    ),
    content: const Text(
      'This event repeats, and every time it repeats moves with it.',
    ),
    actions: [
      TextButton(
        key: const ValueKey('calendar_move_series_cancel'),
        onPressed: onCancel,
        child: const Text('Cancel'),
      ),
      FilledButton(
        key: const ValueKey('calendar_move_series_confirm'),
        autofocus: true,
        onPressed: onConfirm,
        child: const Text('Move all'),
      ),
    ],
  );
}
