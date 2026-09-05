import 'package:flutter/material.dart';
import 'package:quark/utils/quark_widget.dart';

/// The aspects of an appliance a reset can wipe, as chosen in the dialog.
@immutable
class QuarkResetSelection {
  /// Creates a selection. The defaults are the dialog's own: everything on
  /// the appliance, and nothing on a drive that merely happens to be plugged
  /// in.
  const QuarkResetSelection({
    this.database = true,
    this.files = true,
    this.devices = false,
  });

  /// Whether to drop and re-migrate the Quark's databases. Takes the accounts
  /// with it, which is why a reset never selects `account` separately.
  final bool database;

  /// Whether to erase the stored file tree.
  final bool files;

  /// Whether to reach the Quark data directory on attached external drives.
  final bool devices;

  /// Whether the Quark would have anything to do. The endpoint rejects a
  /// request that selects nothing, and so does the dialog's button.
  bool get isEmpty => !database && !files && !devices;

  /// Whether this selection leaves data on the appliance behind.
  ///
  /// Files are the ones a person can still open, so they are what the warning
  /// is about; a database left standing keeps the accounts and the metadata
  /// that point at them.
  bool get leavesDataBehind => !database || !files;

  /// A copy of this selection with the named aspects changed.
  QuarkResetSelection copyWith({bool? database, bool? files, bool? devices}) =>
      QuarkResetSelection(
        database: database ?? this.database,
        files: files ?? this.files,
        devices: devices ?? this.devices,
      );
}

/// What the user reads when a reset would leave data on the appliance.
const String kResetQuarkPartialWarning =
    'This leaves data on the Quark. Whoever sets it up next will be able to '
    'reach whatever you leave.';

/// The confirmation body for factory-resetting the appliance (#1762).
///
/// Deliberately not the account-deletion dialog. Deleting an account removes a
/// person from a Quark; this empties the Quark itself, and a person who wanted
/// the first must never be able to reach the second by checking something. They
/// are separate entries, separate words, and separate calls.
///
/// Everything on the appliance is selected by default, because leaving nothing
/// behind is what a person coming here wants. External drives are not: a drive
/// plugged in for unrelated reasons must not be wiped because a form arrived
/// with the box already checked, so reaching one is always a deliberate act.
///
/// The warning appears only once the user has turned something off and left
/// data behind. A notice that is always on is a notice nobody reads, so this
/// one only speaks when it has something to say.
///
/// Key prefixes: `reset_quark_database`, `reset_quark_files`,
/// `reset_quark_devices`, `reset_quark_warning`, `reset_quark_confirm_field`,
/// `reset_quark_cancel`, and `reset_quark_submit`.
///
/// ```dart
/// QuarkWidget.showDialog<(QuarkResetSelection, String)>(
///   context,
///   builder: (ctx) => ResetQuarkDialog(
///     username: AppSettings.instance.username,
///     onConfirm: (selection, confirm) =>
///         Navigator.of(ctx).pop((selection, confirm)),
///     onCancel: () => Navigator.of(ctx).pop(),
///   ),
/// );
/// ```
class ResetQuarkDialog extends StatefulWidget {
  /// Creates the reset confirmation for the Quark [username] is signed in to.
  const ResetQuarkDialog({
    required this.onConfirm,
    required this.onCancel,
    this.username,
    super.key,
  });

  /// The signed-in account, or null when this session never named one. Only
  /// used to check and phrase the confirmation.
  final String? username;

  /// Called with the chosen aspects and the typed confirmation, once the
  /// confirmation matches and at least one aspect is selected.
  final void Function(QuarkResetSelection selection, String confirmUsername)
  onConfirm;

  /// Called when the user backs out through the cancel button.
  final VoidCallback onCancel;

  @override
  State<ResetQuarkDialog> createState() => _ResetQuarkDialogState();
}

class _ResetQuarkDialogState extends State<ResetQuarkDialog> {
  final _confirmController = TextEditingController();
  QuarkResetSelection _selection = const QuarkResetSelection();

  @override
  void dispose() {
    _confirmController.dispose();
    super.dispose();
  }

  bool get _isConfirmed {
    final typed = _confirmController.text.trim();
    final expected = widget.username;
    return expected == null ? typed.isNotEmpty : typed == expected;
  }

  void _submit() {
    if (!_isConfirmed || _selection.isEmpty) return;
    widget.onConfirm(_selection, _confirmController.text.trim());
  }

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;
    final textTheme = Theme.of(context).textTheme;
    final username = widget.username;

    return QuarkWidget.alertDialog(
      title: const Text('Reset this Quark'),
      scrollable: true,
      content: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          const Text(
            'Returns this Quark to first-boot setup. Every account on it goes, '
            'yours included, and it has to be set up again. This cannot be '
            'undone.',
          ),
          const SizedBox(height: 12),
          CheckboxListTile(
            key: const ValueKey('reset_quark_database'),
            value: _selection.database,
            onChanged: (value) => setState(
              () => _selection = _selection.copyWith(database: value ?? false),
            ),
            contentPadding: EdgeInsets.zero,
            controlAffinity: ListTileControlAffinity.leading,
            title: const Text('Accounts and settings'),
            subtitle: const Text(
              'Every account, album, calendar, and setting on this Quark.',
            ),
          ),
          CheckboxListTile(
            key: const ValueKey('reset_quark_files'),
            value: _selection.files,
            onChanged: (value) => setState(
              () => _selection = _selection.copyWith(files: value ?? false),
            ),
            contentPadding: EdgeInsets.zero,
            controlAffinity: ListTileControlAffinity.leading,
            title: const Text('Stored files'),
            subtitle: const Text(
              'Every photo and document stored on the Quark itself.',
            ),
          ),
          CheckboxListTile(
            key: const ValueKey('reset_quark_devices'),
            value: _selection.devices,
            onChanged: (value) => setState(
              () => _selection = _selection.copyWith(devices: value ?? false),
            ),
            contentPadding: EdgeInsets.zero,
            controlAffinity: ListTileControlAffinity.leading,
            title: const Text('Quark data on attached drives'),
            subtitle: const Text(
              'Off by default. Only the Quark data directory on drives '
              'attached right now; anything else on them is left alone.',
            ),
          ),
          if (_selection.leavesDataBehind) ...[
            const SizedBox(height: 8),
            Container(
              key: const ValueKey('reset_quark_warning'),
              padding: const EdgeInsets.all(12),
              decoration: BoxDecoration(
                color: colorScheme.errorContainer,
                borderRadius: BorderRadius.circular(8),
              ),
              child: Row(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Icon(
                    Icons.warning_amber_rounded,
                    size: 20,
                    color: colorScheme.onErrorContainer,
                  ),
                  const SizedBox(width: 8),
                  Expanded(
                    child: Text(
                      kResetQuarkPartialWarning,
                      style: textTheme.bodyMedium?.copyWith(
                        color: colorScheme.onErrorContainer,
                      ),
                    ),
                  ),
                ],
              ),
            ),
          ],
          const SizedBox(height: 16),
          Text(
            username == null
                ? 'Type your username to confirm.'
                : 'Type $username to confirm.',
          ),
          const SizedBox(height: 8),
          QuarkWidget.textField(
            key: const ValueKey('reset_quark_confirm_field'),
            controller: _confirmController,
            autofocus: true,
            autocorrect: false,
            enableSuggestions: false,
            hintText: username ?? 'username',
            onChanged: (_) => setState(() {}),
            onSubmitted: (_) => _submit(),
          ),
        ],
      ),
      actions: [
        TextButton(
          key: const ValueKey('reset_quark_cancel'),
          onPressed: widget.onCancel,
          child: const Text('Cancel'),
        ),
        FilledButton(
          key: const ValueKey('reset_quark_submit'),
          style: FilledButton.styleFrom(
            backgroundColor: colorScheme.error,
            foregroundColor: colorScheme.onError,
          ),
          onPressed: _isConfirmed && !_selection.isEmpty ? _submit : null,
          child: const Text('Reset this Quark'),
        ),
      ],
    );
  }
}
