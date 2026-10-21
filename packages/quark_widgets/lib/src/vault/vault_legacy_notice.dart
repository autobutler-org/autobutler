import 'package:flutter/material.dart';
import 'package:quark_icons/quark_icons.dart';

import '../models/legacy_vault_item.dart';
import '../theme/quark_tokens.dart';

/// A card on the vault page saying that the shared household vault from an
/// earlier Quark was kept, where it is, and that it is not in the account's
/// own vault, so an upgrade never loses it without anyone being told.
///
/// The household vault was one vault everyone on the Quark opened with one
/// master password. Each account's vault replaced it and starts empty: the
/// old one cannot be moved in, because only that master password opens it.
/// [vaults] lists every copy found, one line each ([describe]), with its file
/// path for whoever manages the Quark. The card renders nothing when
/// [vaults] is empty, and scrolls past a third of the screen's height.
///
/// Key prefixes: `vault_legacy_notice` on the card and
/// `vault_legacy_notice_row_<index>` on each line.
///
/// ```dart
/// VaultLegacyNotice(
///   vaults: [LegacyVaultItem(path: '/var/lib/quark/data/household-vault.db', entries: 12)],
/// );
/// ```
class VaultLegacyNotice extends StatelessWidget {
  /// Creates the notice for [vaults].
  const VaultLegacyNotice({required this.vaults, super.key});

  /// Every household vault found on the Quark.
  final List<LegacyVaultItem> vaults;

  /// The notice's heading.
  static const title = 'Your old shared vault was kept';

  /// What happened, and that nothing was deleted.
  static const body =
      'Before this update, everyone on this Quark shared one vault with one '
      'master password. Each account now has its own vault, which starts '
      'empty. The shared vault was not moved into it, because only that '
      'master password can open it. Nothing was deleted: it is still on this '
      'Quark. Ask whoever manages this Quark before anyone removes it.';

  /// The line [vaults] shows for [vault]: where it is and what it holds.
  static String describe(LegacyVaultItem vault) {
    final where = !vault.onDrive
        ? 'Kept on this Quark'
        : vault.deviceName.isEmpty
        ? 'On a drive'
        : 'On the drive "${vault.deviceName}"';
    if (!vault.readable) return '$where: a file Quark could not read';
    final entries = vault.entries == 1 ? '1 entry' : '${vault.entries} entries';
    final folders = switch (vault.folders) {
      0 => '',
      1 => ', 1 folder',
      final n => ', $n folders',
    };
    return '$where: $entries$folders';
  }

  @override
  Widget build(BuildContext context) {
    if (vaults.isEmpty) return const SizedBox.shrink();
    final tokens = QuarkTokens.of(context);

    return Padding(
      key: const ValueKey('vault_legacy_notice'),
      padding: EdgeInsets.fromLTRB(
        tokens.spacingSm,
        tokens.spacingSm,
        tokens.spacingSm,
        0,
      ),
      child: DecoratedBox(
        decoration: BoxDecoration(
          color: tokens.card,
          border: Border.all(color: tokens.warning),
          borderRadius: BorderRadius.circular(tokens.radiusMd),
        ),
        // At most a third of the screen, scrolling past that, so the vault
        // under it stays usable on a phone at large text.
        child: ConstrainedBox(
          constraints: BoxConstraints(
            maxHeight: MediaQuery.sizeOf(context).height / 3,
          ),
          child: SingleChildScrollView(
            padding: EdgeInsets.all(tokens.spacingMd),
            child: Row(
              crossAxisAlignment: CrossAxisAlignment.start,
              children: [
                Icon(QuarkIcons.info_outline, size: 20, color: tokens.warning),
                SizedBox(width: tokens.spacingSm),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(
                        title,
                        style: TextStyle(
                          color: tokens.foreground,
                          fontWeight: FontWeight.w600,
                        ),
                      ),
                      SizedBox(height: tokens.spacingXs),
                      Text(
                        body,
                        style: TextStyle(color: tokens.mutedForeground),
                      ),
                      for (final (index, vault) in vaults.indexed)
                        Padding(
                          key: ValueKey('vault_legacy_notice_row_$index'),
                          padding: EdgeInsets.only(top: tokens.spacingSm),
                          child: Column(
                            crossAxisAlignment: CrossAxisAlignment.start,
                            children: [
                              Text(
                                describe(vault),
                                style: TextStyle(color: tokens.foreground),
                              ),
                              SelectableText(
                                vault.path,
                                style: TextStyle(
                                  color: tokens.mutedForeground,
                                  fontSize: 12,
                                ),
                              ),
                            ],
                          ),
                        ),
                    ],
                  ),
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }
}
