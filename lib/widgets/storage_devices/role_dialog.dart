import 'package:flutter/material.dart';
import 'package:quark/widgets/storage_devices/role_option.dart';

/// Asks which role a device should take on. Pops with the chosen role string.
class RoleDialog extends StatelessWidget {
  final String currentRole;
  const RoleDialog({required this.currentRole, super.key});

  @override
  Widget build(BuildContext context) {
    return SimpleDialog(
      title: const Text('Set Device Role'),
      children: [
        RoleOption(
          title: 'Default Storage',
          subtitle: 'Always-connected drive for daily file storage',
          selected: currentRole == 'default-storage',
          onTap: () => Navigator.pop(context, 'default-storage'),
        ),
        RoleOption(
          title: 'Snapshot Backup',
          subtitle: 'Plug in, back up everything, unplug',
          selected: currentRole == 'snapshot-backup',
          onTap: () => Navigator.pop(context, 'snapshot-backup'),
        ),
        RoleOption(
          title: 'Unassigned',
          subtitle: 'No special role',
          selected: currentRole == 'unassigned',
          onTap: () => Navigator.pop(context, 'unassigned'),
        ),
      ],
    );
  }
}
