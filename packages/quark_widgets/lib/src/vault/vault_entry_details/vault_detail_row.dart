import 'package:flutter/material.dart';

import '../../theme/quark_tokens.dart';

/// One labeled field of `VaultEntryDetails`, with the controls that act on
/// it beside the value.
class VaultDetailRow extends StatelessWidget {
  /// Creates the row.
  const VaultDetailRow({
    required this.label,
    required this.value,
    this.actions = const [],
    super.key,
  });

  /// What the field is.
  final String label;

  /// What it shows, which may be a mask.
  final String value;

  /// Buttons beside the value, such as copy.
  final List<Widget> actions;

  @override
  Widget build(BuildContext context) {
    final tokens = QuarkTokens.of(context);
    return Padding(
      padding: EdgeInsets.symmetric(vertical: tokens.spacingSm),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            label,
            style: Theme.of(
              context,
            ).textTheme.labelSmall?.copyWith(color: tokens.mutedForeground),
          ),
          Row(
            children: [
              Expanded(
                child: ConstrainedBox(
                  constraints: const BoxConstraints(
                    minHeight: kMinInteractiveDimension,
                  ),
                  // Long-press selects, so the text is a target like the
                  // buttons beside it and gets the same 48dp height.
                  child: SelectableText(value),
                ),
              ),
              ...actions,
            ],
          ),
        ],
      ),
    );
  }
}
