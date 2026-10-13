import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:quark/widgets/storage_devices/role_dialog.dart';

/// #3194: the roles Set Role offers. The Quark refuses snapshot-backup for
/// built-in storage, so the dialog does not offer it there.
void main() {
  Future<List<String?>> pumpDialog(
    WidgetTester tester, {
    required bool isInternal,
  }) async {
    final popped = <String?>[];
    await tester.pumpWidget(
      MaterialApp(
        home: Builder(
          builder: (context) => TextButton(
            onPressed: () async => popped.add(
              await showDialog<String>(
                context: context,
                builder: (_) => RoleDialog(
                  currentRole: 'unassigned',
                  isInternal: isInternal,
                ),
              ),
            ),
            child: const Text('open'),
          ),
        ),
      ),
    );
    await tester.tap(find.text('open'));
    await tester.pumpAndSettle();
    return popped;
  }

  testWidgets('a USB drive is offered every role', (tester) async {
    await pumpDialog(tester, isInternal: false);

    for (final role in ['Default Storage', 'Snapshot Backup', 'Unassigned']) {
      expect(find.text(role), findsOneWidget);
    }
    expect(find.textContaining('copied onto it'), findsOneWidget);
  });

  testWidgets('built-in storage is offered default storage and unassigned, '
      'not snapshot backup', (tester) async {
    final popped = await pumpDialog(tester, isInternal: true);

    expect(find.text('Snapshot Backup'), findsNothing);
    expect(find.text('Unassigned'), findsOneWidget);
    expect(find.textContaining('copied onto it'), findsNothing);

    await tester.tap(find.text('Default Storage'));
    await tester.pumpAndSettle();

    expect(popped, ['default-storage']);
  });
}
