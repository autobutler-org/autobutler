import 'package:quark/widgets/quark_brand_button.dart';
import 'package:quark/widgets/layout/theme_toggle_button.dart';
import 'package:flutter/material.dart';

class QuarkAppBar extends StatelessWidget implements PreferredSizeWidget {
  const QuarkAppBar({
    required this.label,
    required this.icon,
    this.actions = const [],
    super.key,
  });

  final String label;
  final IconData icon;
  final List<Widget> actions;

  @override
  Size get preferredSize => const Size.fromHeight(kToolbarHeight);

  @override
  Widget build(BuildContext context) {
    return AppBar(
      leadingWidth: QuarkBrandButton.preferredWidth + 8,
      leading: Builder(
        builder: (ctx) => Padding(
          padding: const EdgeInsets.only(left: 8),
          child: QuarkBrandButton(
            label: label,
            icon: icon,
            onTap: () => Scaffold.of(ctx).openDrawer(),
          ),
        ),
      ),
      title: null,
      actions: [...actions, const ThemeToggleButton()],
    );
  }
}
