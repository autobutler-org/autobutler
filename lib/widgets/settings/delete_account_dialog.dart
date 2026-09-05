import 'package:flutter/material.dart';
import 'package:quark/utils/quark_widget.dart';

/// What the user reads when their files will outlive their account.
///
/// A constant so the widget that shows it and the test that guards it cannot
/// drift apart, and so removing the copy has to happen somewhere visible.
const String kDeleteAccountFilesWarning =
    'Your files stay on this Quark. Whoever sets it up next will be able to '
    'open them. To erase them too, use Reset this Quark instead.';

/// The confirmation body for deleting the signed-in account (#1762).
///
/// Data in, callbacks out: it neither deletes anything nor closes itself.
/// [onConfirm] fires with the typed confirmation, and the caller that pushed
/// the dialog is the one that pops it.
///
/// Deleting an account and factory-resetting an appliance are two intents, and
/// this dialog only has one of them. There is no control here that can reach
/// the appliance-wide aspects of the endpoint — those live behind Reset this
/// Quark, in their own words, on their own surface. Nobody can arrive here to
/// delete a login and leave having wiped a device.
///
/// The cost of that narrowness is the thing this dialog has to say out loud:
/// the file tree survives, the Quark returns to setup with it intact, and
/// whoever claims it next can read it. [kDeleteAccountFilesWarning] says so,
/// above the confirmation rather than under it.
///
/// The typed confirmation goes to the Quark, which rejects anything but the
/// authenticated username. Checking it here too keeps a typo from spending a
/// round trip, and is skipped when [username] is null: a session recovered by
/// phrase never named a user, so the Quark is the only thing that can judge.
///
/// Key prefixes: `delete_account_confirm_field`,
/// `delete_account_files_warning`, `delete_account_cancel`, and
/// `delete_account_submit`.
///
/// ```dart
/// QuarkWidget.showDialog<String>(
///   context,
///   builder: (ctx) => DeleteAccountDialog(
///     username: AppSettings.instance.username,
///     onConfirm: (confirmUsername) => Navigator.of(ctx).pop(confirmUsername),
///     onCancel: () => Navigator.of(ctx).pop(),
///   ),
/// );
/// ```
class DeleteAccountDialog extends StatefulWidget {
  /// Creates the confirmation body for [username]'s account.
  const DeleteAccountDialog({
    required this.onConfirm,
    required this.onCancel,
    this.username,
    super.key,
  });

  /// The account being deleted, or null when this session never named one.
  final String? username;

  /// Called with the typed confirmation, once it matches [username].
  final ValueChanged<String> onConfirm;

  /// Called when the user backs out through the cancel button.
  final VoidCallback onCancel;

  @override
  State<DeleteAccountDialog> createState() => _DeleteAccountDialogState();
}

class _DeleteAccountDialogState extends State<DeleteAccountDialog> {
  final _confirmController = TextEditingController();

  @override
  void dispose() {
    _confirmController.dispose();
    super.dispose();
  }

  /// Whether what has been typed is enough to send.
  ///
  /// With a known username nothing but that username will do. Without one the
  /// only check available is that the user typed something, and the Quark
  /// rejects it if they typed the wrong thing.
  bool get _isConfirmed {
    final typed = _confirmController.text.trim();
    final expected = widget.username;
    return expected == null ? typed.isNotEmpty : typed == expected;
  }

  void _submit() {
    if (!_isConfirmed) return;
    widget.onConfirm(_confirmController.text.trim());
  }

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;
    final textTheme = Theme.of(context).textTheme;
    final username = widget.username;

    return QuarkWidget.alertDialog(
      title: const Text('Delete account'),
      scrollable: true,
      content: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            username == null
                ? 'Your account will be deleted and you will be signed out '
                      'everywhere. This cannot be undone.'
                : 'The account $username will be deleted and you will be '
                      'signed out everywhere. This cannot be undone.',
          ),
          const SizedBox(height: 12),
          const Text(
            'If this is the only account, the Quark returns to setup and has '
            'to be set up again before anyone can use it.',
          ),
          const SizedBox(height: 12),
          Container(
            key: const ValueKey('delete_account_files_warning'),
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
                    kDeleteAccountFilesWarning,
                    style: textTheme.bodyMedium?.copyWith(
                      color: colorScheme.onErrorContainer,
                    ),
                  ),
                ),
              ],
            ),
          ),
          const SizedBox(height: 16),
          Text(
            username == null
                ? 'Type your username to confirm.'
                : 'Type $username to confirm.',
          ),
          const SizedBox(height: 8),
          QuarkWidget.textField(
            key: const ValueKey('delete_account_confirm_field'),
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
          key: const ValueKey('delete_account_cancel'),
          onPressed: widget.onCancel,
          child: const Text('Cancel'),
        ),
        FilledButton(
          key: const ValueKey('delete_account_submit'),
          style: FilledButton.styleFrom(
            backgroundColor: colorScheme.error,
            foregroundColor: colorScheme.onError,
          ),
          onPressed: _isConfirmed ? _submit : null,
          child: const Text('Delete my account'),
        ),
      ],
    );
  }
}
