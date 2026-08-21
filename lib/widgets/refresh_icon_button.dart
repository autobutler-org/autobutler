import 'package:flutter/material.dart';
import 'package:quark_icons/quark_icons.dart';

/// An [IconButton] that shows a small [CircularProgressIndicator] in place of
/// the refresh icon while [isRefreshing] is true, and disables taps.
class RefreshIconButton extends StatelessWidget {
  const RefreshIconButton({
    super.key,
    required this.onPressed,
    required this.isRefreshing,
    this.tooltip = 'Refresh',
  });

  final VoidCallback? onPressed;
  final bool isRefreshing;
  final String tooltip;

  @override
  Widget build(BuildContext context) {
    return IconButton(
      tooltip: tooltip,
      onPressed: isRefreshing ? null : onPressed,
      icon: isRefreshing
          ? SizedBox(
              width: 20,
              height: 20,
              child: CircularProgressIndicator(
                strokeWidth: 2,
                color: Theme.of(context).colorScheme.onSurface,
              ),
            )
          : const Icon(QuarkIcons.refresh),
    );
  }
}
