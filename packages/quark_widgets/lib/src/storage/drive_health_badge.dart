import 'package:flutter/material.dart';

import '../theme/quark_tokens.dart';
import 'drive_health_verdict.dart';

/// A chip naming a drive's health verdict.
///
/// The word is always drawn. Color repeats it and never replaces it, and
/// [DriveHealthVerdict.unknown] does not use the healthy color.
///
/// Key prefixes: `drive_health_badge`, or `drive_health_badge_<id>` when
/// [id] is set so several badges can share a tree.
///
/// ```dart
/// DriveHealthBadge(verdict: DriveHealthVerdict.watch)
/// ```
class DriveHealthBadge extends StatelessWidget {
  /// Creates a badge reading [verdict].
  const DriveHealthBadge({required this.verdict, this.id, super.key});

  /// The verdict the Quark reported.
  final DriveHealthVerdict verdict;

  /// Distinguishes this badge's key when another badge is in the same tree.
  /// Null uses `drive_health_badge`.
  final String? id;

  @override
  Widget build(BuildContext context) {
    final tokens = QuarkTokens.of(context);
    final color = driveHealthVerdictColor(tokens, verdict);
    final id = this.id;
    final key = id == null || id.isEmpty
        ? 'drive_health_badge'
        : 'drive_health_badge_$id';

    return Semantics(
      label: '${verdict.label} drive health',
      child: DecoratedBox(
        key: ValueKey(key),
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
            verdict.label,
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
