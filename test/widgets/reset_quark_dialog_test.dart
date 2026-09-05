import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:quark/widgets/settings/reset_quark_dialog.dart';

/// #1762: the appliance-wide wipe is a different intent from deleting an
/// account, so it is a different dialog. What it defaults to is the whole
/// argument for splitting them — everything on the Quark, nothing on a drive
/// that merely happens to be plugged in — and the warning only speaks once the
/// user has turned something off and left data behind.
void main() {
  const narrowViewport = Size(360, 640);
  const wideViewport = Size(1280, 800);

  final confirmField = find.byKey(const ValueKey('reset_quark_confirm_field'));
  final database = find.byKey(const ValueKey('reset_quark_database'));
  final files = find.byKey(const ValueKey('reset_quark_files'));
  final devices = find.byKey(const ValueKey('reset_quark_devices'));
  final warning = find.byKey(const ValueKey('reset_quark_warning'));
  final submit = find.byKey(const ValueKey('reset_quark_submit'));

  bool isChecked(WidgetTester tester, Finder tile) =>
      tester.widget<CheckboxListTile>(tile).value ?? false;

  /// Taps [target], scrolling it into view first.
  ///
  /// The dialog scrolls its own content on a narrow viewport, so a checkbox
  /// below the fold is there but not hittable until it has been scrolled to.
  Future<void> tapVisible(WidgetTester tester, Finder target) async {
    await tester.ensureVisible(target);
    await tester.pump();
    await tester.tap(target);
    await tester.pump();
  }

  /// Pumps the dialog at [size] and returns the confirmations it emits.
  Future<List<(QuarkResetSelection, String)>> pumpDialog(
    WidgetTester tester, {
    Size size = wideViewport,
    String? username = 'ada',
  }) async {
    tester.view.physicalSize = size;
    tester.view.devicePixelRatio = 1.0;
    addTearDown(tester.view.reset);

    final confirmations = <(QuarkResetSelection, String)>[];
    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: ResetQuarkDialog(
            username: username,
            onConfirm: (selection, typed) =>
                confirmations.add((selection, typed)),
            onCancel: () {},
          ),
        ),
      ),
    );
    await tester.pump();
    return confirmations;
  }

  /// Runs [body] against both viewports every widget has to survive.
  void testBothViewports(
    String description,
    Future<void> Function(WidgetTester tester, Size size) body,
  ) {
    for (final size in [narrowViewport, wideViewport]) {
      final label = size == narrowViewport ? 'narrow' : 'wide';
      testWidgets('$description ($label)', (tester) => body(tester, size));
    }
  }

  testBothViewports('selects the whole appliance but no attached drive', (
    tester,
    size,
  ) async {
    await pumpDialog(tester, size: size);

    expect(isChecked(tester, database), isTrue);
    expect(isChecked(tester, files), isTrue);
    // Reaching a drive plugged in for unrelated reasons stays a deliberate act.
    expect(isChecked(tester, devices), isFalse);
  });

  testBothViewports('stays quiet while nothing would be left behind', (
    tester,
    size,
  ) async {
    await pumpDialog(tester, size: size);

    expect(warning, findsNothing);
  });

  testBothViewports('warns once stored files are left out', (
    tester,
    size,
  ) async {
    await pumpDialog(tester, size: size);

    await tapVisible(tester, files);

    expect(warning, findsOneWidget);
    expect(find.text(kResetQuarkPartialWarning), findsOneWidget);
  });

  testWidgets('warns once the accounts are left out', (tester) async {
    await pumpDialog(tester, size: narrowViewport);

    await tapVisible(tester, database);

    expect(warning, findsOneWidget);
  });

  testWidgets('goes quiet again once everything is back on', (tester) async {
    await pumpDialog(tester, size: narrowViewport);

    await tapVisible(tester, files);
    await tapVisible(tester, files);

    expect(warning, findsNothing);
  });

  testBothViewports('does not reset until the username matches', (
    tester,
    size,
  ) async {
    final confirmations = await pumpDialog(tester, size: size);

    expect(tester.widget<FilledButton>(submit).onPressed, isNull);
    await tester.enterText(confirmField, 'adam');
    await tester.pump();
    expect(tester.widget<FilledButton>(submit).onPressed, isNull);

    await tapVisible(tester, submit);

    expect(confirmations, isEmpty);
  });

  testBothViewports('confirms the selection once the username matches', (
    tester,
    size,
  ) async {
    final confirmations = await pumpDialog(tester, size: size);

    await tapVisible(tester, devices);
    await tester.enterText(confirmField, 'ada');
    await tester.pump();
    await tapVisible(tester, submit);

    expect(confirmations.single.$2, 'ada');
    final selection = confirmations.single.$1;
    expect(selection.database, isTrue);
    expect(selection.files, isTrue);
    expect(selection.devices, isTrue);
  });

  testWidgets('refuses to send a reset that would do nothing', (tester) async {
    final confirmations = await pumpDialog(tester, size: narrowViewport);

    await tester.enterText(confirmField, 'ada');
    await tester.pump();
    await tapVisible(tester, database);
    await tapVisible(tester, files);

    // The Quark answers 400 to a request that selects nothing. Not sending it
    // is how the user finds that out.
    expect(tester.widget<FilledButton>(submit).onPressed, isNull);
    await tapVisible(tester, submit);

    expect(confirmations, isEmpty);
  });
}
