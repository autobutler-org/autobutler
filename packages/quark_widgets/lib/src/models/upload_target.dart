import 'package:flutter/foundation.dart';

/// A storage device an upload can be sent to.
///
/// The package's own view of a device, so the picker does not have to import
/// the app's storage model. Controllers map their device type into this.
@immutable
class UploadTarget {
  /// Creates an upload target.
  const UploadTarget({
    required this.serial,
    required this.name,
    this.mountPoint = '',
    this.isInternal = false,
    this.unavailableReason,
  });

  /// The device's serial, which is what an upload names. May be empty for the
  /// default device.
  final String serial;

  /// The device's display name. Empty falls back to "Device".
  final String name;

  /// Where the device is mounted, shown under its name. May be empty.
  final String mountPoint;

  /// Whether the device is built into the server rather than plugged in.
  final bool isInternal;

  /// Why an upload cannot go to this device, in words the caller wrote for a
  /// user to read — a drive that is plugged in but not enabled, say. Null
  /// when the device can be chosen.
  final String? unavailableReason;

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is UploadTarget &&
          other.serial == serial &&
          other.name == name &&
          other.mountPoint == mountPoint &&
          other.isInternal == isInternal &&
          other.unavailableReason == unavailableReason;

  @override
  int get hashCode =>
      Object.hash(serial, name, mountPoint, isInternal, unavailableReason);
}
