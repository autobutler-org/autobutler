import 'package:flutter/material.dart';
import 'package:quark/utils/quark_widget.dart';

/// Confirms a snapshot backup to the drive named [targetName] and offers to
/// include the vault. Pops with the recovery password, empty when the vault
/// is left out, or with nothing on Cancel.
///
/// The dialog owns its text controller, so it lives until its route has
/// finished animating out (#3102).
class SnapshotBackupDialog extends StatefulWidget {
  const SnapshotBackupDialog({required this.targetName, super.key});

  /// The backup drive's name; empty reads as "this drive".
  final String targetName;

  @override
  State<SnapshotBackupDialog> createState() => _SnapshotBackupDialogState();
}

class _SnapshotBackupDialogState extends State<SnapshotBackupDialog> {
  final _recoveryCtrl = TextEditingController();

  @override
  void dispose() {
    _recoveryCtrl.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return QuarkWidget.alertDialog(
      title: const Text('Snapshot Backup'),
      content: Column(
        mainAxisSize: MainAxisSize.min,
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            'Back up all devices to ${widget.targetName.isNotEmpty ? widget.targetName : "this drive"}.',
          ),
          const SizedBox(height: 16),
          const Text(
            'Optionally include your vault with a recovery password:',
            style: TextStyle(fontWeight: FontWeight.w500),
          ),
          const SizedBox(height: 8),
          TextField(
            controller: _recoveryCtrl,
            obscureText: true,
            decoration: const InputDecoration(
              labelText: 'Recovery password (optional)',
              helperText: 'Min 8 characters. Separate from master password.',
              border: OutlineInputBorder(),
            ),
          ),
        ],
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.pop(context),
          child: const Text('Cancel'),
        ),
        TextButton(
          onPressed: () => Navigator.pop(context, _recoveryCtrl.text),
          child: const Text('Start Backup'),
        ),
      ],
    );
  }
}
