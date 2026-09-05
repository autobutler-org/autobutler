import 'package:flutter/material.dart';
import 'package:quark/utils/clipboard_utils.dart';
import 'package:quark_icons/quark_icons.dart';
import 'package:quark_widgets/quark_widgets.dart';

/// Shows the recovery phrase and requires acknowledgement before proceeding.
class RecoveryPhraseStep extends StatelessWidget {
  final String phrase;
  final bool acknowledged;
  final ValueChanged<bool?> onAcknowledgedChanged;
  final VoidCallback onContinue;

  const RecoveryPhraseStep({
    super.key,
    required this.phrase,
    required this.acknowledged,
    required this.onAcknowledgedChanged,
    required this.onContinue,
  });

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    return Column(
      mainAxisSize: MainAxisSize.min,
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Icon(
          QuarkIcons.key_rounded,
          size: 56,
          color: theme.colorScheme.primary,
          semanticLabel: 'Recovery phrase',
        ),
        const SizedBox(height: 16),
        Text(
          'Save your recovery phrase',
          style: theme.textTheme.headlineMedium?.copyWith(
            fontWeight: FontWeight.bold,
          ),
          textAlign: TextAlign.center,
        ),
        const SizedBox(height: 8),
        Text(
          'This phrase is the only way to reset your password if you forget it. '
          "It will not be shown again. Write it down somewhere safe — don't store it digitally on this device.",
          style: theme.textTheme.bodyMedium?.copyWith(
            color: theme.colorScheme.onSurface.withValues(alpha: 0.7),
          ),
          textAlign: TextAlign.center,
        ),
        const SizedBox(height: 24),

        // Phrase display box
        Semantics(
          label: 'Recovery phrase: $phrase',
          child: Container(
            padding: const EdgeInsets.all(20),
            decoration: BoxDecoration(
              color: theme.colorScheme.surfaceContainerHighest,
              borderRadius: BorderRadius.circular(12),
              border: Border.all(color: theme.colorScheme.outlineVariant),
            ),
            child: Column(
              children: [
                SelectableText(
                  phrase,
                  style: theme.textTheme.titleMedium?.copyWith(
                    fontFamily: 'monospace',
                    fontWeight: FontWeight.bold,
                    letterSpacing: 1.2,
                    height: 1.8,
                  ),
                  textAlign: TextAlign.center,
                ),
                const SizedBox(height: 12),
                CopyButton(
                  text: phrase,
                  icon: QuarkIcons.copy_outlined,
                  variant: CopyButtonVariant.outlined,
                  unavailableReason: clipboardUnavailableReason,
                  onCopy: (value) => copyToClipboard(
                    context,
                    value,
                    message: 'Recovery phrase copied to clipboard',
                  ),
                ),
              ],
            ),
          ),
        ),
        const SizedBox(height: 24),

        // Acknowledgement checkbox
        CheckboxListTile(
          value: acknowledged,
          onChanged: onAcknowledgedChanged,
          title: const Text(
            'I have written down my recovery phrase and stored it safely.',
          ),
          controlAffinity: ListTileControlAffinity.leading,
          contentPadding: EdgeInsets.zero,
        ),
        const SizedBox(height: 16),

        FilledButton(
          onPressed: acknowledged ? onContinue : null,
          child: const Text('Continue'),
        ),
      ],
    );
  }
}
