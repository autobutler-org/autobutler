import 'package:flutter/material.dart';

import 'format_version_time.dart';

/// Asks before a file is put back to an earlier version, and says why that
/// is safe: the current content is saved first as a version named "Before
/// restore", so the restore can itself be undone.
///
/// It does not close itself: [onConfirm] and [onCancel] fire, and the caller
/// that pushed the dialog pops it.
///
/// Keys: `file_version_restore_confirm` and `file_version_restore_cancel`.
///
/// ```dart
/// showDialog<bool>(
///   context: context,
///   builder: (ctx) => RestoreVersionDialog(
///     fileName: 'notes.md',
///     savedAt: version.createdAt,
///     onConfirm: () => Navigator.of(ctx).pop(true),
///     onCancel: () => Navigator.of(ctx).pop(false),
///   ),
/// );
/// ```
class RestoreVersionDialog extends StatelessWidget {
  /// Creates the confirmation for restoring [fileName] to the version from
  /// [savedAt].
  const RestoreVersionDialog({
    required this.fileName,
    required this.savedAt,
    required this.onConfirm,
    required this.onCancel,
    super.key,
  });

  /// The file being restored.
  final String fileName;

  /// When the version it goes back to was saved.
  final DateTime savedAt;

  /// Called when Restore is tapped.
  final VoidCallback onConfirm;

  /// Called when Cancel is tapped.
  final VoidCallback onCancel;

  @override
  Widget build(BuildContext context) {
    return AlertDialog(
      scrollable: true,
      title: const Text('Restore this version?'),
      content: Text(
        '$fileName goes back to the version from '
        '${formatVersionTime(context, savedAt)}. What it holds '
        'now is saved first as "Before restore", so you can undo this.',
      ),
      actions: [
        TextButton(
          key: const ValueKey('file_version_restore_cancel'),
          onPressed: onCancel,
          child: const Text('Cancel'),
        ),
        FilledButton(
          key: const ValueKey('file_version_restore_confirm'),
          onPressed: onConfirm,
          child: const Text('Restore'),
        ),
      ],
    );
  }
}
