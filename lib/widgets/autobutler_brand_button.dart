import 'package:autobutler/theme/autobutler_colors.dart';
import 'package:flutter/material.dart';

/// Styled brand button used in the top bar across all main pages.
/// Shows the AutoButler storage icon + a page title label and opens
/// the navigation drawer when tapped.
///
/// When used as [AppBar.leading], set [AppBar.leadingWidth] to
/// [AutobutlerBrandButton.preferredWidth] to avoid overflow.
class AutobutlerBrandButton extends StatelessWidget {
  const AutobutlerBrandButton({
    required this.label,
    required this.onTap,
    super.key,
  });

  /// Use this as [AppBar.leadingWidth] when placing this widget in an AppBar.
  static const double preferredWidth = 140.0;

  /// The page label shown next to the icon (e.g. "Files", "Photos").
  final String label;

  /// Called when the button is tapped — typically opens the drawer.
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    final radius = BorderRadius.circular(AutobutlerColors.radiusMd);
    return MouseRegion(
      cursor: SystemMouseCursors.click,
      child: Material(
        color: Colors.transparent,
        clipBehavior: Clip.antiAlias,
        borderRadius: radius,
        child: InkWell(
          onTap: onTap,
          borderRadius: radius,
          child: Padding(
            padding: const EdgeInsets.symmetric(horizontal: 4, vertical: 4),
            child: Row(
              mainAxisSize: MainAxisSize.min,
              children: [
                Container(
                  width: 28,
                  height: 28,
                  decoration: BoxDecoration(
                    color: AutobutlerColors.primary,
                    borderRadius: radius,
                  ),
                  child: const Center(
                    child: Icon(
                      Icons.storage_rounded,
                      size: 16,
                      color: AutobutlerColors.primaryForeground,
                    ),
                  ),
                ),
                const SizedBox(width: 10),
                Text(
                  label,
                  style: const TextStyle(
                    fontSize: 15,
                    fontWeight: FontWeight.w600,
                    color: AutobutlerColors.cardForeground,
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
