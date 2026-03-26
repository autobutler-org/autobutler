import 'package:autobutler/widgets/autobutler_brand_button.dart';
import 'package:flutter/material.dart';

class AutobutlerAppBar extends StatelessWidget implements PreferredSizeWidget {
  const AutobutlerAppBar({
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
      leadingWidth: AutobutlerBrandButton.preferredWidth,
      leading: Builder(
        builder: (ctx) => AutobutlerBrandButton(
          label: label,
          icon: icon,
          onTap: () => Scaffold.of(ctx).openDrawer(),
        ),
      ),
      title: null,
      actions: actions,
    );
  }
}
