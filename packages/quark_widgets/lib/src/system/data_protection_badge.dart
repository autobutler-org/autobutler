import 'package:flutter/material.dart';

import '../theme/quark_tokens.dart';
import 'data_protection_level.dart';

/// A compact chip naming [level], for the backup drive a person starts a
/// backup from.
///
/// It draws the level it is given and nothing else. The status card carries
/// the sentence and the next step; this is the same level in one word.
///
/// Key prefixes: `data_protection_badge`, or `data_protection_badge_<id>`
/// when [id] is set so several badges can share a tree.
///
/// ```dart
/// DataProtectionBadge(level: DataProtectionLevel.basic)
/// ```
class DataProtectionBadge extends StatelessWidget {
  /// Creates a badge reading [level].
  const DataProtectionBadge({required this.level, this.id, super.key});

  /// The level the Quark reported.
  final DataProtectionLevel level;

  /// Distinguishes this badge's key when another badge is in the same tree.
  /// Null uses `data_protection_badge`.
  final String? id;

  @override
  Widget build(BuildContext context) {
    final tokens = QuarkTokens.of(context);
    final color = dataProtectionLevelColor(tokens, level);
    final id = this.id;

    return Semantics(
      label: '${level.label} protection',
      child: DecoratedBox(
        key: ValueKey(
          id == null ? 'data_protection_badge' : 'data_protection_badge_$id',
        ),
        decoration: BoxDecoration(
          color: color.withValues(alpha: 0.12),
          border: Border.all(color: color),
          borderRadius: BorderRadius.circular(tokens.radiusSm),
        ),
        child: Padding(
          padding: EdgeInsets.symmetric(
            horizontal: tokens.spacingSm,
            vertical: tokens.spacingXs,
          ),
          child: Text(
            level.label,
            maxLines: 1,
            overflow: TextOverflow.ellipsis,
            style: TextStyle(
              color: color,
              fontSize: 12,
              fontWeight: FontWeight.w600,
              height: 1.3,
            ),
          ),
        ),
      ),
    );
  }
}
