import 'package:flutter/material.dart';

import '../theme/quark_tokens.dart';

/// What a [QuarkToolbar] does when its actions do not fit on one line.
enum QuarkToolbarOverflow {
  /// Continue on the next line, growing the toolbar taller. The default, and
  /// the right answer anywhere the toolbar's height is free.
  wrap,

  /// Scroll horizontally, keeping the toolbar one line tall. The right answer
  /// in a slot with a fixed height, such as a 56 pixel bar.
  scroll,
}

/// A row of actions that never overflows: it wraps onto another line, or
/// scrolls sideways, depending on [overflow].
///
/// A plain `Row` of buttons is fine until someone opens the app on a 360
/// pixel phone, at which point it throws a layout error and paints the yellow
/// stripes over the controls. Every bar in the app solved that separately.
/// This solves it once, so a new bar starts out narrow-safe.
///
/// The toolbar spaces its actions with the theme's small spacing token.
///
/// Key prefixes: none of its own. The toolbar has nothing tappable, so the
/// keys a test or a `.probe` script reaches for are the ones the actions
/// carry.
///
/// ```dart
/// QuarkToolbar(
///   actions: [
///     IconButton(key: const ValueKey('files_refresh'), ...),
///     FilledButton(onPressed: upload, child: const Text('Upload')),
///   ],
/// );
/// ```
class QuarkToolbar extends StatelessWidget {
  /// Creates a toolbar of [actions].
  const QuarkToolbar({
    required this.actions,
    this.overflow = QuarkToolbarOverflow.wrap,
    super.key,
  });

  /// The controls, rendered in order along the main axis.
  final List<Widget> actions;

  /// What happens when the actions do not fit on one line.
  final QuarkToolbarOverflow overflow;

  @override
  Widget build(BuildContext context) {
    final tokens = QuarkTokens.of(context);

    if (overflow == QuarkToolbarOverflow.scroll) {
      return SingleChildScrollView(
        scrollDirection: Axis.horizontal,
        child: Row(
          mainAxisSize: MainAxisSize.min,
          spacing: tokens.spacingSm,
          children: actions,
        ),
      );
    }

    return Wrap(
      spacing: tokens.spacingSm,
      runSpacing: tokens.spacingXs,
      crossAxisAlignment: WrapCrossAlignment.center,
      children: actions,
    );
  }
}
