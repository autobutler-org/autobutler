import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:quark_widgets/quark_widgets.dart';

import '../support/pump.dart';

void main() {
  VaultPasswordForm form({
    String? phraseHelp,
    bool isBusy = false,
    String? error,
    List<String>? submits,
    List<String>? secondary,
  }) => VaultPasswordForm(
    title: 'Your vault is locked',
    message: 'Enter the password you sign in with.',
    submitLabel: 'Unlock',
    phraseHelp: phraseHelp,
    isBusy: isBusy,
    error: error,
    secondaryLabel: secondary == null ? null : 'Start over',
    onSecondary: secondary == null ? null : () => secondary.add('tap'),
    onSubmit: (password, phrase) => submits?.add('$password/$phrase'),
  );

  testBothViewports('lays out the prompt without an exception', (
    tester,
    size,
  ) async {
    await pumpAt(
      tester,
      form(phraseHelp: 'Lets a recovery keep the vault.', secondary: []),
      size: size,
    );

    expect(find.text('Your vault is locked'), findsOneWidget);
    expect(
      find.byKey(const ValueKey('vault_password_secondary')),
      findsOneWidget,
    );
    expect(tester.takeException(), isNull);
  });

  testBothViewports('asks for the password once', (tester, size) async {
    await pumpAt(
      tester,
      form(phraseHelp: 'Lets a recovery keep the vault.'),
      size: size,
    );

    expect(find.byKey(const ValueKey('vault_password_field')), findsOneWidget);
    expect(find.byKey(const ValueKey('vault_password_confirm')), findsNothing);
    expect(find.byType(TextField), findsNWidgets(2));
  });

  testBothViewports('submits the password once and clears the field', (
    tester,
    size,
  ) async {
    final submits = <String>[];
    await pumpAt(tester, form(submits: submits), size: size);

    await tester.enterText(
      find.byKey(const ValueKey('vault_password_field')),
      'hunter22',
    );
    await tester.tap(find.byKey(const ValueKey('vault_password_submit')));
    await tester.pump();

    expect(submits, ['hunter22/']);
    expect(find.text('hunter22'), findsNothing);
  });

  testBothViewports('with phraseHelp, offers an optional phrase and hands it '
      'over', (tester, size) async {
    final submits = <String>[];
    await pumpAt(
      tester,
      form(phraseHelp: 'Lets a recovery keep the vault.', submits: submits),
      size: size,
    );
    expect(find.text('Lets a recovery keep the vault.'), findsOneWidget);
    expect(tester.takeException(), isNull);

    await tester.enterText(
      find.byKey(const ValueKey('vault_password_field')),
      'one',
    );
    await tester.enterText(
      find.byKey(const ValueKey('vault_password_phrase')),
      'apple-bread',
    );
    final submit = find.byKey(const ValueKey('vault_password_submit'));
    await tester.ensureVisible(submit);
    await tester.tap(submit);
    await tester.pump();

    expect(submits, ['one/apple-bread']);
    expect(find.text('apple-bread'), findsNothing);
  });

  testBothViewports('without phraseHelp, has no phrase field', (
    tester,
    size,
  ) async {
    await pumpAt(tester, form(), size: size);

    expect(find.byKey(const ValueKey('vault_password_phrase')), findsNothing);
  });

  testBothViewports('an empty password submits nothing', (tester, size) async {
    final submits = <String>[];
    await pumpAt(tester, form(submits: submits), size: size);

    await tester.tap(find.byKey(const ValueKey('vault_password_submit')));
    await tester.pump();

    expect(submits, isEmpty);
  });

  testBothViewports('renders the error and holds still while busy', (
    tester,
    size,
  ) async {
    final secondary = <String>[];
    await pumpAt(
      tester,
      form(isBusy: true, error: 'Wrong password.', secondary: secondary),
      size: size,
    );

    expect(find.text('Wrong password.'), findsOneWidget);
    expect(find.byType(QuarkLoader), findsOneWidget);
    expect(
      tester
          .widget<FilledButton>(
            find.byKey(const ValueKey('vault_password_submit')),
          )
          .onPressed,
      isNull,
    );
    await tester.tap(find.byKey(const ValueKey('vault_password_secondary')));
    expect(secondary, isEmpty);
  });

  testBothViewports('reports the secondary action', (tester, size) async {
    final secondary = <String>[];
    await pumpAt(tester, form(secondary: secondary), size: size);

    await tester.tap(find.byKey(const ValueKey('vault_password_secondary')));
    await tester.pump();

    expect(secondary, ['tap']);
  });
}
