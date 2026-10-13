import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:quark_widgets/quark_widgets.dart';

import '../support/pump.dart';

const _internal = UploadTarget(
  serial: '',
  name: '',
  mountPoint: '/data',
  isInternal: true,
);
const _usb = UploadTarget(serial: 'usb-1', name: 'Backup drive');
const _off = UploadTarget(
  serial: 'usb-2',
  name: 'Old drive',
  mountPoint: '/mnt/off',
  unavailableReason: 'Mount it first',
);

void main() {
  testBothViewports('lists the targets and emits every action', (
    tester,
    size,
  ) async {
    final events = <String>[];
    await pumpInSheet(
      tester,
      UploadTargetPicker(
        targets: const [_internal, _usb],
        selected: _internal,
        onSelected: (t) => events.add('select:${t.serial}'),
        onCancel: () => events.add('cancel'),
        onConfirm: () => events.add('confirm'),
      ),
      size: size,
    );

    expect(tester.takeException(), isNull);
    expect(find.text('Device'), findsOneWidget);
    expect(find.text('/data · Internal'), findsOneWidget);
    expect(find.text('Backup drive'), findsOneWidget);

    await tester.tap(find.byKey(const ValueKey('upload_target_1')));
    await tester.tap(find.byKey(const ValueKey('upload_target_cancel')));
    await tester.tap(find.byKey(const ValueKey('upload_target_confirm')));

    expect(events, ['select:usb-1', 'cancel', 'confirm']);
  });

  // #3191: a drive that is plugged in but not enabled is shown, not hidden,
  // so the user learns what the missing step is.
  testBothViewports('an unavailable target shows its reason and cannot be '
      'chosen', (tester, size) async {
    final events = <String>[];
    await pumpInSheet(
      tester,
      UploadTargetPicker(
        targets: const [_internal, _usb, _off],
        selected: _internal,
        onSelected: (t) => events.add('select:${t.serial}'),
        onCancel: () {},
        onConfirm: () {},
      ),
      size: size,
    );

    expect(tester.takeException(), isNull);
    expect(find.text('/mnt/off · Mount it first'), findsOneWidget);
    final row = find.byKey(const ValueKey('upload_target_2'));
    expect(tester.widget<RadioListTile<UploadTarget>>(row).enabled, isFalse);

    await tester.tap(row);
    await tester.tap(find.byKey(const ValueKey('upload_target_1')));

    expect(events, ['select:usb-1']);
  });
}
