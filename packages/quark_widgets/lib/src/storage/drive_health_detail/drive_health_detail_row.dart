// cspell:words reallocated
import 'package:flutter/material.dart';

import '../../theme/quark_tokens.dart';

/// One labeled reading in a drive health detail, such as power-on hours.
///
/// The row prints [value] as it is given. It is not exported: the detail
/// panel is the only caller, and the panel decides which readings exist.
class DriveHealthDetailRow extends StatelessWidget {
  /// Creates a row named [label] showing [value], keyed by [rowKey].
  const DriveHealthDetailRow({
    required this.label,
    required this.value,
    required this.rowKey,
    super.key,
  });

  /// The reading's name, such as "Power-on hours".
  final String label;

  /// The reading, already in words.
  final String value;

  /// The row's [ValueKey], `drive_health_row_<field>` or
  /// `drive_health_row_<id>_<field>`.
  final String rowKey;

  @override
  Widget build(BuildContext context) {
    final tokens = QuarkTokens.of(context);
    final theme = Theme.of(context);

    return Padding(
      key: ValueKey(rowKey),
      padding: EdgeInsets.only(top: tokens.spacingXs),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(
            label,
            style: theme.textTheme.bodySmall?.copyWith(
              color: tokens.secondaryForeground,
            ),
          ),
          Text(value, style: theme.textTheme.bodyMedium),
        ],
      ),
    );
  }
}
