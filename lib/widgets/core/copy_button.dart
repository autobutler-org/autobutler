import 'package:quark/utils/clipboard_utils.dart';
import 'package:flutter/material.dart';
import 'package:quark_icons/quark_icons.dart';
import 'package:flutter/services.dart';

/// A reusable copy-to-clipboard button that handles insecure context detection.
///
/// When the clipboard is unavailable (e.g. non-HTTPS web context), the button
/// renders disabled with a tooltip explaining the limitation.
///
/// Use this anywhere copy functionality is needed instead of directly calling
/// [Clipboard.setData].
class CopyButton extends StatelessWidget {
  const CopyButton({
    required this.text,
    this.icon = QuarkIcons.content_copy,
    this.iconSize = 16,
    this.label,
    this.successMessage = 'Copied to clipboard',
    this.variant = CopyButtonVariant.icon,
    super.key,
  });

  /// The text to copy to the clipboard.
  final String text;

  /// Icon to display. Defaults to [QuarkIcons.content_copy].
  final IconData icon;

  /// Icon size. Defaults to 16.
  final double iconSize;

  /// Optional label for [CopyButtonVariant.outlined] variant.
  final String? label;

  /// Message shown in the snackbar after a successful copy.
  final String successMessage;

  /// Visual style of the button.
  final CopyButtonVariant variant;

  Future<void> _copy(BuildContext context) async {
    await Clipboard.setData(ClipboardData(text: text));
    if (!context.mounted) return;
    ScaffoldMessenger.of(context).showSnackBar(
      SnackBar(
        content: Text(successMessage),
        duration: const Duration(seconds: 2),
      ),
    );
  }

  @override
  Widget build(BuildContext context) {
    final available = isClipboardAvailable;
    final tooltip = available
        ? 'Copy to clipboard'
        : 'Clipboard unavailable — use HTTPS to enable';

    switch (variant) {
      case CopyButtonVariant.icon:
        return IconButton(
          icon: Icon(icon, size: iconSize),
          tooltip: tooltip,
          onPressed: available ? () => _copy(context) : null,
        );
      case CopyButtonVariant.outlined:
        return Tooltip(
          message: available ? '' : tooltip,
          child: OutlinedButton.icon(
            onPressed: available ? () => _copy(context) : null,
            icon: Icon(icon, size: iconSize),
            label: Text(label ?? 'Copy to clipboard'),
          ),
        );
    }
  }
}

enum CopyButtonVariant {
  /// A simple [IconButton] — for inline use in rows/toolbars.
  icon,

  /// An [OutlinedButton.icon] with a text label — for standalone CTAs.
  outlined,
}
