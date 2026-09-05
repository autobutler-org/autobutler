import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:quark/widgets/settings/delete_account_dialog.dart';

/// #1762: App Store Review Guideline 5.1.1(v) needs deletion initiated in the
/// app, and a destructive action needs friction in front of it. These pin the
/// friction — the typed confirmation — and the two things the dialog must not
/// do: reach anything but the account, or stay quiet about the files it leaves
/// behind for whoever sets the Quark up next.
void main() {
  const narrowViewport = Size(360, 640);
  const wideViewport = Size(1280, 800);

  final confirmField = find.byKey(
    const ValueKey('delete_account_confirm_field'),
  );
  final filesWarning = find.byKey(
    const ValueKey('delete_account_files_warning'),
  );
  final submit = find.byKey(const ValueKey('delete_account_submit'));
  final cancel = find.byKey(const ValueKey('delete_account_cancel'));

  /// Pumps the dialog at [size] and returns the confirmations it emits.
  Future<List<String>> pumpDialog(
    WidgetTester tester, {
    Size size = wideViewport,
    String? username = 'ada',
    List<String>? events,
  }) async {
    tester.view.physicalSize = size;
    tester.view.devicePixelRatio = 1.0;
    addTearDown(tester.view.reset);

    final confirmations = <String>[];
    await tester.pumpWidget(
      MaterialApp(
        home: Scaffold(
          body: DeleteAccountDialog(
            username: username,
            onConfirm: confirmations.add,
            onCancel: () => events?.add('cancel'),
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

  testBothViewports('names the account it is about to delete', (
    tester,
    size,
  ) async {
    await pumpDialog(tester, size: size);

    expect(
      find.textContaining('The account ada will be deleted'),
      findsOneWidget,
    );
    expect(find.text('Delete my account'), findsOneWidget);
  });

  testBothViewports('offers nothing that reaches the appliance', (
    tester,
    size,
  ) async {
    await pumpDialog(tester, size: size);

    // Deleting an account and resetting a Quark are two intents. There is no
    // control here that can turn the first into the second.
    expect(find.byType(Checkbox), findsNothing);
    expect(find.byType(Switch), findsNothing);
  });

  testBothViewports('does not confirm until the username is typed', (
    tester,
    size,
  ) async {
    final confirmations = await pumpDialog(tester, size: size);

    expect(tester.widget<FilledButton>(submit).onPressed, isNull);
    await tester.tap(submit);
    await tester.pump();

    expect(confirmations, isEmpty);
  });

  testBothViewports('refuses a username that does not match', (
    tester,
    size,
  ) async {
    final confirmations = await pumpDialog(tester, size: size);

    await tester.enterText(confirmField, 'adam');
    await tester.pump();

    expect(tester.widget<FilledButton>(submit).onPressed, isNull);
    await tester.tap(submit);
    await tester.pump();

    expect(confirmations, isEmpty);
  });

  testBothViewports('confirms once the username matches', (tester, size) async {
    final confirmations = await pumpDialog(tester, size: size);

    await tester.enterText(confirmField, 'ada');
    await tester.pump();
    await tester.tap(submit);
    await tester.pump();

    expect(confirmations, ['ada']);
  });

  testBothViewports('warns that the files outlive the account', (
    tester,
    size,
  ) async {
    await pumpDialog(tester, size: size);

    expect(filesWarning, findsOneWidget);
    expect(find.text(kDeleteAccountFilesWarning), findsOneWidget);
    // Naming the remedy, since this dialog deliberately cannot apply it.
    expect(kDeleteAccountFilesWarning, contains('Reset this Quark'));
  });

  testWidgets('says the Quark returns to setup when this is the last account', (
    tester,
  ) async {
    await pumpDialog(tester, size: narrowViewport);

    expect(find.textContaining('the Quark returns to setup'), findsOneWidget);
  });

  testWidgets('leaves the check to the Quark when no username is known', (
    tester,
  ) async {
    final confirmations = await pumpDialog(
      tester,
      size: narrowViewport,
      username: null,
    );

    expect(find.text('Type your username to confirm.'), findsOneWidget);
    await tester.enterText(confirmField, 'whoever');
    await tester.pump();
    await tester.tap(submit);
    await tester.pump();

    expect(confirmations, ['whoever']);
  });

  testWidgets('cancels without confirming', (tester) async {
    final events = <String>[];
    final confirmations = await pumpDialog(
      tester,
      size: narrowViewport,
      events: events,
    );

    await tester.tap(cancel);
    await tester.pump();

    expect(events, ['cancel']);
    expect(confirmations, isEmpty);
  });
}
