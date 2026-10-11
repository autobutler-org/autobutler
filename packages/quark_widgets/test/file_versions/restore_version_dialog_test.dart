import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:quark_widgets/quark_widgets.dart';

import '../support/pump.dart';

/// The restore confirmation: it has to say the current content is kept, and
/// it leaves closing to the caller.
void main() {
  Future<void> pumpDialog(
    WidgetTester tester, {
    Size size = wideViewport,
    String fileName = 'notes.md',
    VoidCallback? onConfirm,
    VoidCallback? onCancel,
  }) => pumpAt(
    tester,
    RestoreVersionDialog(
      fileName: fileName,
      savedAt: DateTime(2026, 10, 11, 15, 4),
      onConfirm: onConfirm ?? () {},
      onCancel: onCancel ?? () {},
    ),
    size: size,
  );

  testBothViewports('names the file, the version and the safety net', (
    tester,
    size,
  ) async {
    await pumpDialog(tester, size: size);

    expect(find.text('Restore this version?'), findsOneWidget);
    expect(find.textContaining('notes.md'), findsOneWidget);
    expect(find.textContaining('Oct 11, 2026, 3:04 PM'), findsOneWidget);
    expect(find.textContaining('"Before restore"'), findsOneWidget);
    expect(find.textContaining('undo'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });

  testBothViewports('fires each button once and does not close itself', (
    tester,
    size,
  ) async {
    final events = <String>[];
    await pumpDialog(
      tester,
      size: size,
      onConfirm: () => events.add('confirm'),
      onCancel: () => events.add('cancel'),
    );

    await tester.tap(
      find.byKey(const ValueKey('file_version_restore_confirm')),
    );
    await tester.tap(find.byKey(const ValueKey('file_version_restore_cancel')));
    await tester.pump();

    expect(events, ['confirm', 'cancel']);
    expect(find.byType(RestoreVersionDialog), findsOneWidget);
  });

  testBothViewports('survives a file name far wider than the dialog', (
    tester,
    size,
  ) async {
    await pumpDialog(tester, size: size, fileName: 'quarterly-report-' * 20);

    expect(tester.takeException(), isNull);
  });

  testLargeText('keeps both buttons reachable', (tester, size) async {
    await pumpDialog(tester, size: size);

    expect(tester.takeException(), isNull);
    expect(
      find.byKey(const ValueKey('file_version_restore_confirm')),
      findsOneWidget,
    );
  });
}
