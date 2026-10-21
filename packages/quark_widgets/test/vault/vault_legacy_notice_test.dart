import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:quark_widgets/quark_widgets.dart';

import '../support/pump.dart';

void main() {
  const kept = LegacyVaultItem(
    path: '/var/lib/quark/data/household-vault-20261021T120000Z.db',
    entries: 12,
    folders: 3,
  );
  const onDrive = LegacyVaultItem(
    path: '/media/backup/quark/data/vault.db',
    onDrive: true,
    deviceName: 'Backup drive',
    entries: 1,
  );

  testBothViewports('says the shared vault was kept, and where', (
    tester,
    size,
  ) async {
    await pumpAt(
      tester,
      const SingleChildScrollView(
        child: VaultLegacyNotice(vaults: [kept, onDrive]),
      ),
      size: size,
    );

    expect(find.text(VaultLegacyNotice.title), findsOneWidget);
    expect(find.textContaining('Nothing was deleted'), findsOneWidget);
    expect(
      find.text('Kept on this Quark: 12 entries, 3 folders'),
      findsOneWidget,
    );
    expect(find.text('On the drive "Backup drive": 1 entry'), findsOneWidget);
    expect(find.text(kept.path), findsOneWidget);
    expect(
      find.byKey(const ValueKey('vault_legacy_notice_row_1')),
      findsOneWidget,
    );
  });

  testWidgets('renders nothing when no vault was found', (tester) async {
    await pumpAt(tester, const VaultLegacyNotice(vaults: []));

    expect(find.byKey(const ValueKey('vault_legacy_notice')), findsNothing);
  });

  test('describes an unreadable file and an unnamed drive', () {
    expect(
      VaultLegacyNotice.describe(
        const LegacyVaultItem(path: '/x', onDrive: true, readable: false),
      ),
      'On a drive: a file Quark could not read',
    );
    expect(
      VaultLegacyNotice.describe(const LegacyVaultItem(path: '/x')),
      'Kept on this Quark: 0 entries',
    );
  });
}
