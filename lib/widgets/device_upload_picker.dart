import 'package:flutter/material.dart';
import 'package:quark/services/storage_service.dart';

/// Shows a bottom sheet letting the user pick a target device for upload.
///
/// Returns the selected [StorageDevice], or `null` if the user cancels.
Future<StorageDevice?> showDeviceUploadPicker(
  BuildContext context,
  List<StorageDevice> devices,
) {
  return showModalBottomSheet<StorageDevice>(
    context: context,
    builder: (ctx) => _DeviceUploadPicker(devices: devices),
  );
}

class _DeviceUploadPicker extends StatefulWidget {
  const _DeviceUploadPicker({required this.devices});

  final List<StorageDevice> devices;

  @override
  State<_DeviceUploadPicker> createState() => _DeviceUploadPickerState();
}

class _DeviceUploadPickerState extends State<_DeviceUploadPicker> {
  late StorageDevice _selected;

  @override
  void initState() {
    super.initState();
    _selected = widget.devices.first;
  }

  String _subtitle(StorageDevice d) {
    final parts = <String>[];
    if (d.mountPoint.isNotEmpty) parts.add(d.mountPoint);
    if (d.isInternal) parts.add('Internal');
    return parts.join(' · ');
  }

  @override
  Widget build(BuildContext context) {
    return SafeArea(
      child: Padding(
        padding: const EdgeInsets.symmetric(vertical: 16),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Padding(
              padding: const EdgeInsets.symmetric(horizontal: 16),
              child: Text(
                'Upload to device',
                style: Theme.of(context).textTheme.titleMedium,
              ),
            ),
            const SizedBox(height: 8),
            RadioGroup<StorageDevice>(
              groupValue: _selected,
              onChanged: (v) {
                if (v != null) setState(() => _selected = v);
              },
              child: Column(
                mainAxisSize: MainAxisSize.min,
                children: [
                  for (final device in widget.devices)
                    RadioListTile<StorageDevice>(
                      title: Text(
                        device.name.isNotEmpty ? device.name : 'Device',
                      ),
                      subtitle: Text(_subtitle(device)),
                      value: device,
                    ),
                ],
              ),
            ),
            const SizedBox(height: 8),
            Padding(
              padding: const EdgeInsets.symmetric(horizontal: 16),
              child: Row(
                mainAxisAlignment: MainAxisAlignment.end,
                children: [
                  TextButton(
                    onPressed: () => Navigator.pop(context),
                    child: const Text('Cancel'),
                  ),
                  const SizedBox(width: 8),
                  FilledButton(
                    onPressed: () => Navigator.pop(context, _selected),
                    child: const Text('Upload'),
                  ),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }
}
