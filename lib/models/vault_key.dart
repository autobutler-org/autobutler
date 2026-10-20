import 'dart:convert';

import 'package:quark/models/account_keys.dart';

/// What the Quark stores of an account's vault key (#3459): the random key
/// every vault item is encrypted under, wrapped on the client under the
/// login password and, when the client had it, the recovery phrase. The
/// Quark never holds the key itself, nor anything that opens a wrap.
///
/// Both wraps are in the split scheme (`AccountCrypto.wrapVaultKey`): the
/// password wrap's key is the `vaultWrapKey` of
/// `AccountCrypto.deriveAuthKeys`, the phrase wrap's that of
/// `AccountCrypto.deriveRecoveryKeys`, and each salt is the account's auth
/// salt.
class WrappedVaultKey {
  /// Builds the stored form.
  const WrappedVaultKey({
    required this.byPassword,
    required this.byPhrase,
    required this.kdfParams,
  });

  /// Wrapped under the login password's `vaultWrapKey`.
  final WrappedSecret byPassword;

  /// Wrapped under the recovery phrase's `vaultWrapKey`. Null for a key made
  /// with no phrase to hand, which a recovery cannot open.
  final WrappedSecret? byPhrase;

  /// The Argon2id cost both wraps' keys were derived with.
  final KdfParams kdfParams;

  /// Reads `GET /api/v0/vault/key` and `POST /api/v0/auth/recover/vault-key`.
  /// The timestamps beside the wraps are not kept.
  factory WrappedVaultKey.fromJson(Map<String, dynamic> json) {
    final byPhrase = json['wrappedByPhrase'] as String?;
    final saltRp = json['saltRp'] as String?;
    return WrappedVaultKey(
      byPassword: WrappedSecret(
        wrapped: base64Decode(json['wrappedByPassword'] as String),
        salt: base64Decode(json['saltPw'] as String),
      ),
      byPhrase: byPhrase == null || saltRp == null
          ? null
          : WrappedSecret(
              wrapped: base64Decode(byPhrase),
              salt: base64Decode(saltRp),
            ),
      kdfParams: KdfParams.fromJson(json['kdfParams'] as Map<String, dynamic>),
    );
  }

  /// The body of `PUT /api/v0/vault/key`, and of `vaultKey` in the
  /// credential requests (`PUT /auth/password`, `PUT /auth/recovery-key`,
  /// `POST /auth/recover`).
  Map<String, dynamic> toJson() => {
    'wrappedByPassword': base64Encode(byPassword.wrapped),
    'saltPw': base64Encode(byPassword.salt),
    if (byPhrase case final phrase?) ...{
      'wrappedByPhrase': base64Encode(phrase.wrapped),
      'saltRp': base64Encode(phrase.salt),
    },
    'kdfParams': kdfParams.toJson(),
  };
}
