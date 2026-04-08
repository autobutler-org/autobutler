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
    this.icon = Icons.storage_rounded,
    super.key,
  });

  /// Use this as [AppBar.leadingWidth] when placing this widget in an AppBar.
  static const double preferredWidth = 140.0;

  /// The page label shown next to the icon (e.g. "Files", "Photos").
  final String label;

  /// Called when the button is tapped — typically opens the drawer.
  final VoidCallback onTap;

  /// Icon shown inside the brand badge. Defaults to [Icons.storage_rounded]
  /// (used for the Files/Cirrus page). Pass a page-appropriate icon for other pages.
  final IconData icon;

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;
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
                    color: colorScheme.primary,
                    borderRadius: radius,
                  ),
                  child: Center(
                    child: Icon(icon, size: 16, color: colorScheme.onPrimary),
                  ),
                ),
                const SizedBox(width: 10),
                Text(
                  label,
                  style: TextStyle(
                    fontSize: 15,
                    fontWeight: FontWeight.w600,
                    color: colorScheme.onSurface,
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
