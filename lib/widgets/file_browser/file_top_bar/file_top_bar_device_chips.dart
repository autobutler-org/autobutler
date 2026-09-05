import 'package:flutter/material.dart';
import 'package:quark/services/storage_service.dart';
import 'package:quark/widgets/file_browser/file_top_bar/top_bar_chip.dart';
import 'package:quark_icons/quark_icons.dart';

/// One toggle per attached device, shown on wide viewports only — the compact
/// layout folds the same filter into the Views menu.
class FileTopBarDeviceChips extends StatelessWidget {
  const FileTopBarDeviceChips({
    required this.devices,
    required this.activeDevicePaths,
    required this.onDeviceToggled,
    super.key,
  });

  final List<StorageDevice> devices;
  final Set<String>? activeDevicePaths;
  final ValueChanged<String>? onDeviceToggled;

  @override
  Widget build(BuildContext context) {
    return Row(
      mainAxisSize: MainAxisSize.min,
      children: devices.map((device) {
        final isSelected =
            activeDevicePaths?.contains(device.devicePath) ?? true;
        return Padding(
          padding: const EdgeInsets.only(right: 6),
          child: TopBarChip(
            icon: isSelected
                ? QuarkIcons.check_circle_outline_rounded
                : QuarkIcons.circle_outlined,
            label: device.name.isNotEmpty ? device.name : device.mountPoint,
            onTap: onDeviceToggled != null
                ? () => onDeviceToggled!(device.devicePath)
                : null,
            active: isSelected,
          ),
        );
      }).toList(),
    );
  }
}
