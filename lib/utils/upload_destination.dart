import 'package:quark/services/storage_service.dart';
import 'package:quark/utils/quark_widget_items.dart';
import 'package:quark_widgets/quark_widgets.dart';

/// The storage devices as they are now, past [StorageService]'s cache: a
/// drive plugged in or enabled a moment ago has to be offered (#3191).
Future<List<StorageDevice>> listDevicesFresh() {
  StorageService.invalidateDeviceCache();
  return StorageService.listDevices();
}

/// The serial of the device an upload or a new folder goes to — `''` for
/// built-in storage — or null when the user dismissed the picker.
///
/// Only enabled devices are destinations. [viewedSerial] is the one device
/// the view is filtered to, null when it shows several: that device is the
/// destination without asking. Otherwise a single enabled device is the
/// destination, and [pick] chooses between several.
///
/// Throws what [listDevices] threw. A destination that cannot be worked out
/// is not built-in storage; the caller stops and says so (#3191).
Future<String?> resolveUploadSerial({
  required Future<UploadTarget?> Function(List<UploadTarget> targets) pick,
  String? viewedSerial,
  Future<List<StorageDevice>> Function() listDevices = listDevicesFresh,
}) async {
  final targets = [
    for (final device in await listDevices())
      if (device.isEnabled) device.toUploadTarget(),
  ];
  if (targets.any((target) => target.serial == viewedSerial)) {
    return viewedSerial;
  }
  if (targets.length > 1) return (await pick(targets))?.serial;
  return targets.firstOrNull?.serial ?? '';
}
