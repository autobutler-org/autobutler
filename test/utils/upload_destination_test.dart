import 'package:flutter_test/flutter_test.dart';
import 'package:quark/services/storage_service.dart';
import 'package:quark/utils/upload_destination.dart';
import 'package:quark_widgets/quark_widgets.dart';

/// #3191: which device an upload or a new folder goes to.
void main() {
  StorageDevice device(
    String name, {
    String serial = '',
    bool isEnabled = true,
  }) => StorageDevice(
    name: name,
    devicePath: '/dev/$name',
    mountPoint: '/mnt/$name',
    fileSystem: 'ext4',
    totalBytes: 0,
    usedBytes: 0,
    availableBytes: 0,
    isInternal: serial.isEmpty,
    isEnabled: isEnabled,
    serial: serial,
  );

  final builtIn = device('Built-in');
  final stick = device('Stick', serial: 'USB1');

  /// A picker that records what it was offered and answers [choose].
  late List<List<UploadTarget>> offered;
  Future<UploadTarget?> Function(List<UploadTarget>) picker(
    UploadTarget? Function(List<UploadTarget>) choose,
  ) => (targets) async {
    offered.add(targets);
    return choose(targets);
  };

  setUp(() => offered = []);

  test('several enabled devices ask, and the answer is the serial', () async {
    final serial = await resolveUploadSerial(
      pick: picker((targets) => targets.last),
      listDevices: () async => [builtIn, stick],
    );

    expect(serial, 'USB1');
    expect(offered.single.map((t) => t.name), ['Built-in', 'Stick']);
  });

  test('built-in storage is the empty serial, not a dismissal', () async {
    final serial = await resolveUploadSerial(
      pick: picker((targets) => targets.first),
      listDevices: () async => [builtIn, stick],
    );

    expect(serial, '');
  });

  test('dismissing the picker is null', () async {
    final serial = await resolveUploadSerial(
      pick: picker((_) => null),
      listDevices: () async => [builtIn, stick],
    );

    expect(serial, isNull);
  });

  test('one enabled device is the destination without asking', () async {
    final serial = await resolveUploadSerial(
      pick: picker((targets) => targets.first),
      listDevices: () async => [
        device('Off', serial: 'USB0', isEnabled: false),
        stick,
      ],
    );

    expect(serial, 'USB1');
    expect(offered, isEmpty);
  });

  test('the one device the view shows is the destination', () async {
    for (final viewed in [builtIn, stick]) {
      final serial = await resolveUploadSerial(
        pick: picker((_) => null),
        viewedSerial: viewed.serial,
        listDevices: () async => [builtIn, stick],
      );

      expect(serial, viewed.serial);
    }
    expect(offered, isEmpty);
  });

  test('a viewed device that is no longer enabled asks instead', () async {
    final serial = await resolveUploadSerial(
      pick: picker((targets) => targets.first),
      viewedSerial: 'USB9',
      listDevices: () async => [builtIn, stick],
    );

    expect(serial, '');
    expect(offered, hasLength(1));
  });

  test('a failed device listing is thrown, not built-in storage', () async {
    await expectLater(
      resolveUploadSerial(
        pick: picker((targets) => targets.first),
        listDevices: () async => throw Exception('offline'),
      ),
      throwsException,
    );
    expect(offered, isEmpty);
  });

  test('the device list is read when asked, every time', () async {
    var devices = [builtIn];
    Future<List<StorageDevice>> list() async => devices;
    final pick = picker((targets) => targets.last);

    expect(await resolveUploadSerial(pick: pick, listDevices: list), '');
    devices = [builtIn, stick];
    expect(await resolveUploadSerial(pick: pick, listDevices: list), 'USB1');
  });
}
