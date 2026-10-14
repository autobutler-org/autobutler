import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:quark_widgets/quark_widgets.dart';

import '../support/pump.dart';

/// The question in front of moving a repeating event's whole series (#3184).
void main() {
  testBothViewports('names the event and answers through its keys', (
    tester,
    size,
  ) async {
    final events = <String>[];
    await pumpAt(
      tester,
      CalendarMoveSeriesDialog(
        title: 'Soccer practice',
        onConfirm: () => events.add('confirm'),
        onCancel: () => events.add('cancel'),
      ),
      size: size,
    );
    expect(find.text('Move all of Soccer practice?'), findsOneWidget);

    await tester.tap(find.byKey(const ValueKey('calendar_move_series_cancel')));
    await tester.tap(
      find.byKey(const ValueKey('calendar_move_series_confirm')),
    );
    await tester.pump();
    expect(events, ['cancel', 'confirm']);
    expect(tester.takeException(), isNull);
  });

  testBothViewports('asks plainly about an untitled event, at large text', (
    tester,
    size,
  ) async {
    tester.platformDispatcher.textScaleFactorTestValue = 2;
    addTearDown(tester.platformDispatcher.clearTextScaleFactorTestValue);
    await pumpAt(
      tester,
      CalendarMoveSeriesDialog(title: '', onConfirm: () {}, onCancel: () {}),
      size: size,
    );
    expect(find.text('Move every time it repeats?'), findsOneWidget);
    expect(tester.takeException(), isNull);
  });
}
