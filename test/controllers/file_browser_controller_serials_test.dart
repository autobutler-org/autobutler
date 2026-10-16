import 'package:flutter_test/flutter_test.dart';
import 'package:quark/controllers/file_browser_controller.dart';
import 'package:quark/services/storage_service.dart';

StorageDevice _device(String path, {String serial = ''}) => StorageDevice(
  name: path,
  devicePath: path,
  mountPoint: '/mnt$path',
  fileSystem: 'ext4',
  totalBytes: 0,
  usedBytes: 0,
  availableBytes: 0,
  isInternal: serial.isEmpty,
  isEnabled: true,
  serial: serial,
);

void main() {
  final internal = _device('/dev/mmcblk0');
  final usb1 = _device('/dev/sda', serial: 'USB1');
  final usb2 = _device('/dev/sdb', serial: 'USB2');
  final devices = [internal, usb1, usb2];

  group('serialFilter (#3201)', () {
    test('every device selected sends no filter', () {
      expect(
        FileBrowserController.serialFilter(devices, {
          internal.devicePath,
          usb1.devicePath,
          usb2.devicePath,
        }),
        isNull,
      );
    });

    test('built-in storage alone names the internal drive', () {
      expect(
        FileBrowserController.serialFilter(devices, {internal.devicePath}),
        [''],
      );
    });

    test('built-in storage plus one drive keeps the internal drive', () {
      expect(
        FileBrowserController.serialFilter(devices, {
          internal.devicePath,
          usb1.devicePath,
        }),
        ['', 'USB1'],
      );
    });

    test('one drive alone sends only its serial', () {
      expect(FileBrowserController.serialFilter(devices, {usb2.devicePath}), [
        'USB2',
      ]);
    });

    test('no devices known sends no filter', () {
      expect(FileBrowserController.serialFilter(const [], const {}), isNull);
    });
  });
}
