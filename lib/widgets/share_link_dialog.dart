import 'package:autobutler/services/share_service.dart';
import 'package:autobutler/utils/autobutler_widget.dart';
import 'package:autobutler_icons/autobutler_icons.dart';
import 'package:flutter/material.dart';
import 'package:flutter/services.dart';

/// Expiry choices offered when creating a share link. Hours of 0 = never.
const _expiryOptions = <(String, int)>[
  ('1 hour', 1),
  ('1 day', 24),
  ('7 days', 24 * 7),
  ('30 days', 24 * 30),
  ('Never', 0),
];

/// Shows the share-link creation dialog for [filePath]. On success the dialog
/// switches to showing the generated link with a copy button.
Future<void> showShareLinkDialog(
  BuildContext context, {
  required String filePath,
  required String displayName,
  String? serial,
}) {
  return AutobutlerWidget.showDialog<void>(
    context,
    useRootNavigator: true,
    barrierDismissible: true,
    builder: (context) => _ShareLinkDialog(
      filePath: filePath,
      displayName: displayName,
      serial: serial,
    ),
  );
}

class _ShareLinkDialog extends StatefulWidget {
  const _ShareLinkDialog({
    required this.filePath,
    required this.displayName,
    this.serial,
  });

  final String filePath;
  final String displayName;
  final String? serial;

  @override
  State<_ShareLinkDialog> createState() => _ShareLinkDialogState();
}

class _ShareLinkDialogState extends State<_ShareLinkDialog> {
  final _passwordController = TextEditingController();
  int _expiresInHours = 24 * 7;
  bool _creating = false;
  String? _error;
  ShareLink? _created;
  bool _copied = false;

  @override
  void dispose() {
    _passwordController.dispose();
    super.dispose();
  }

  Future<void> _create() async {
    setState(() {
      _creating = true;
      _error = null;
    });
    try {
      final link = await ShareService.create(
        filePath: widget.filePath,
        serial: widget.serial,
        expiresInHours: _expiresInHours,
        password: _passwordController.text,
      );
      if (!mounted) return;
      setState(() => _created = link);
    } catch (_) {
      if (!mounted) return;
      setState(() => _error = 'Failed to create share link');
    } finally {
      if (mounted) setState(() => _creating = false);
    }
  }

  Future<void> _copyLink() async {
    final link = _created;
    if (link == null) return;
    await Clipboard.setData(ClipboardData(text: link.fullUrl));
    if (!mounted) return;
    setState(() => _copied = true);
  }

  @override
  Widget build(BuildContext context) {
    final created = _created;
    return AlertDialog(
      title: Text(
        created == null ? 'Share "${widget.displayName}"' : 'Link created',
      ),
      content: created == null ? _buildForm() : _buildResult(created),
      actions: created == null
          ? [
              TextButton(
                onPressed: _creating ? null : () => Navigator.of(context).pop(),
                child: const Text('Cancel'),
              ),
              FilledButton(
                onPressed: _creating ? null : _create,
                child: _creating
                    ? const SizedBox(
                        width: 16,
                        height: 16,
                        child: CircularProgressIndicator(strokeWidth: 2),
                      )
                    : const Text('Create link'),
              ),
            ]
          : [
              TextButton(
                onPressed: () => Navigator.of(context).pop(),
                child: const Text('Done'),
              ),
            ],
    );
  }

  Widget _buildForm() {
    return Column(
      mainAxisSize: MainAxisSize.min,
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        const Text('Anyone with the link can download this item.'),
        const SizedBox(height: 16),
        DropdownButtonFormField<int>(
          initialValue: _expiresInHours,
          decoration: const InputDecoration(labelText: 'Expires'),
          items: _expiryOptions
              .map(
                (option) => DropdownMenuItem<int>(
                  value: option.$2,
                  child: Text(option.$1),
                ),
              )
              .toList(),
          onChanged: (value) =>
              setState(() => _expiresInHours = value ?? _expiresInHours),
        ),
        const SizedBox(height: 12),
        TextField(
          controller: _passwordController,
          obscureText: true,
          decoration: const InputDecoration(
            labelText: 'Password (optional)',
            hintText: 'Leave empty for no password',
          ),
        ),
        if (_error != null) ...[
          const SizedBox(height: 12),
          Text(
            _error!,
            style: TextStyle(color: Theme.of(context).colorScheme.error),
          ),
        ],
      ],
    );
  }

  Widget _buildResult(ShareLink link) {
    return Column(
      mainAxisSize: MainAxisSize.min,
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        SelectableText(link.fullUrl),
        const SizedBox(height: 12),
        Align(
          alignment: Alignment.centerLeft,
          child: OutlinedButton.icon(
            onPressed: _copyLink,
            icon: Icon(
              _copied ? AutobutlerIcons.check : AutobutlerIcons.content_copy,
            ),
            label: Text(_copied ? 'Copied' : 'Copy link'),
          ),
        ),
        if (link.passwordProtected) ...[
          const SizedBox(height: 12),
          const Text(
            'Remember to send the password separately.',
            style: TextStyle(fontSize: 12),
          ),
        ],
      ],
    );
  }
}
