import 'package:flutter/material.dart';
import 'package:quark/services/storage_service.dart';
import 'package:quark/widgets/device_upload_picker/device_upload_picker.dart';

/// Shows a bottom sheet letting the user pick a target device for upload.
///
/// Returns the selected [StorageDevice], or `null` if the user cancels.
Future<StorageDevice?> showDeviceUploadPicker(
  BuildContext context,
  List<StorageDevice> devices,
) {
  return showModalBottomSheet<StorageDevice>(
    context: context,
    builder: (ctx) => DeviceUploadPicker(devices: devices),
  );
}
