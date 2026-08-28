import 'package:flutter/material.dart';
import 'package:quark/services/app_settings.dart';
import 'package:quark/utils/quark_widget.dart';

/// Add/edit dialog for a single Quark.
///
/// Owns its text controllers so they live exactly as long as the dialog's
/// element. They used to be created by the caller and disposed in a post-frame
/// callback once `showDialog` resolved, which killed them while the dismiss
/// transition was still running and the fields were still mounted — hence
/// "A TextEditingController was used after being disposed".
///
/// It also only ever pops a [HostEntry]: saving is the caller's job, so the
/// route stack is never mutated while this dialog is on screen (#1623).
class HostDialog extends StatefulWidget {
  const HostDialog({super.key, required this.isEdit, this.initial});

  final bool isEdit;
  final HostEntry? initial;

  @override
  State<HostDialog> createState() => _HostDialogState();
}

class _HostDialogState extends State<HostDialog> {
  late final TextEditingController _name = TextEditingController(
    text: widget.initial?.name ?? '',
  );
  late final TextEditingController _address = TextEditingController(
    text: widget.initial?.hostAddress ?? '',
  );

  @override
  void dispose() {
    _name.dispose();
    _address.dispose();
    super.dispose();
  }

  void _submit() {
    final name = _name.text.trim();
    final address = _address.text.trim();
    if (name.isEmpty || address.isEmpty) return;
    Navigator.of(context).pop(HostEntry(name: name, hostAddress: address));
  }

  @override
  Widget build(BuildContext context) {
    return QuarkWidget.alertDialog(
      title: Text(widget.isEdit ? 'Edit Quark' : 'Add Quark'),
      content: Column(
        mainAxisSize: MainAxisSize.min,
        children: [
          QuarkWidget.textField(
            controller: _name,
            autofocus: true,
            textInputAction: TextInputAction.next,
            hintText: 'Nickname (e.g. Home)',
          ),
          const SizedBox(height: 8),
          QuarkWidget.textField(
            controller: _address,
            textInputAction: TextInputAction.done,
            // Enter in the address field saves, same as the button.
            onSubmitted: (_) => _submit(),
            hintText: 'https://quark.home.local',
          ),
          const SizedBox(height: 6),
          const Text(
            'Usually https://quark.home.local or the IP address shown on your device.',
            style: TextStyle(fontSize: 12, color: Colors.grey),
          ),
        ],
      ),
      actions: [
        TextButton(
          onPressed: () => Navigator.of(context).pop(),
          child: const Text('Cancel'),
        ),
        TextButton(onPressed: _submit, child: const Text('Save')),
      ],
    );
  }
}
