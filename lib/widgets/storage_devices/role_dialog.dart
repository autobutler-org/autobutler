import 'package:flutter/material.dart';
import 'package:quark/widgets/storage_devices/role_option.dart';

/// Asks which role a device should take on. Pops with the chosen role string.
///
/// Default storage is where the upload and new folder pickers start (#3194),
/// and on a USB drive it is also what live sync copies built-in storage onto.
class RoleDialog extends StatelessWidget {
  const RoleDialog({
    required this.currentRole,
    this.isInternal = false,
    super.key,
  });

  final String currentRole;

  /// Whether the device is built-in storage, which cannot be a snapshot
  /// backup: the Quark refuses that role for it.
  final bool isInternal;

  @override
  Widget build(BuildContext context) {
    return SimpleDialog(
      title: const Text('Set Device Role'),
      children: [
        RoleOption(
          title: 'Default Storage',
          subtitle: isInternal
              ? 'Uploads and new folders start here'
              : 'Uploads and new folders start here. Built-in storage is '
                    'also copied onto it.',
          selected: currentRole == 'default-storage',
          onTap: () => Navigator.pop(context, 'default-storage'),
        ),
        if (!isInternal)
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
