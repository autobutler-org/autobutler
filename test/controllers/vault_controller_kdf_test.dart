import 'dart:convert';
import 'dart:typed_data';

import 'package:flutter_test/flutter_test.dart';
import 'package:quark/controllers/vault_controller.dart';
import 'package:quark/models/account_keys.dart';
import 'package:quark/models/vault_key.dart';
import 'package:quark/services/account_crypto.dart';
import 'package:quark/utils/error_text.dart';

/// #3648: a vault key the Quark serves at a cost outside the app's bounds is
/// refused when it is read, so the vault never derives at it.
void main() {
  /// `GET /vault/key` as `VaultService.fetchKey` reads it, at [opsLimit]
  /// passes over [memLimit] bytes.
  Future<WrappedVaultKey?> served(int opsLimit, int memLimit) async =>
      WrappedVaultKey.fromJson({
        'wrappedByPassword': base64Encode(Uint8List(72)),
        'saltPw': base64Encode(Uint8List(16)),
        'kdfParams': {
          'alg': KdfParams.splitAlgorithm,
          'opsLimit': opsLimit,
          'memLimit': memLimit,
        },
      });

  for (final (name, opsLimit, memLimit) in [
    ('below the standard', 1, 8192),
    ('above the ceiling', KdfParams.ceiling.opsLimit, 4 << 30),
  ]) {
    test('a key $name is refused, and nothing is derived', () async {
      var cryptoLoaded = false;
      final c = VaultController(
        loadCrypto: () async {
          cryptoLoaded = true;
          return AccountCrypto.load();
        },
        fetchKey: () => served(opsLimit, memLimit),
        authSalt: () async => Uint8List(16),
      );
      addTearDown(c.dispose);

      await c.load();
      expect(c.status, VaultStatus.error);
      expect(
        Errors.message(c.error, 'load your vault'),
        Errors.keyCostOutOfRange,
      );

      await c.unlock('any-password');
      expect(c.status, VaultStatus.error);
      expect(cryptoLoaded, isFalse);
    });
  }
}
