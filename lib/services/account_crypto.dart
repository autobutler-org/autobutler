import 'dart:convert';
import 'dart:typed_data';

import 'package:quark/models/account_keys.dart';
import 'package:quark/utils/error_text.dart';
import 'package:quark/utils/recovery_phrase.dart';
import 'package:sodium/sodium_sumo.dart';

/// The account's key material, on libsodium: what the login password and the
/// recovery phrase derive, and the wraps those keys seal (#3457).
///
/// One Argon2id run over a secret is split by HKDF into per-purpose keys, so
/// everything the account encrypts on the client opens with one password and
/// one recovery phrase. Chat (`ChatCrypto`) wraps its identity seeds with
/// this. The Quark never runs any of it: it is sent [AuthKeys.authKey] and
/// stores what [wrapWithKey] produces.
class AccountCrypto {
  /// Wraps an initialized libsodium.
  AccountCrypto(this.sodium);

  /// The libsodium instance everything here runs on.
  final SodiumSumo sodium;

  static Future<AccountCrypto>? _loading;

  /// Loads libsodium once for the life of the app.
  ///
  /// On web this waits for `sodium.js`, which `web/index.html` loads.
  static Future<AccountCrypto> load() => _loading ??=
      Future(() async => AccountCrypto(await SodiumSumoInit.init())).catchError(
        (Object e) {
          _loading = null;
          throw e;
        },
      );

  Aead get _aead => sodium.crypto.aeadXChaCha20Poly1305IETF;

  /// Derives the two keys a password stands for (#2430), so the Quark is sent
  /// one and can open nothing with it.
  ///
  /// The construction is permanent once accounts exist:
  ///
  /// ```text
  /// master  = Argon2id13(password, salt, params.opsLimit, params.memLimit), 32 bytes
  /// authKey = HKDF-SHA256(ikm = master, salt = none, info = "auth"), 32 bytes
  /// wrapKey = HKDF-SHA256(ikm = master, salt = none, info = "chat-wrap"), 32 bytes
  /// ```
  ///
  /// HKDF is RFC 5869 extract-then-expand, with the empty salt the RFC reads
  /// as 32 zero bytes. [salt] is the account's 16-byte auth salt from
  /// `GET /auth/salt`. The caller disposes the result.
  AuthKeys deriveAuthKeys(String password, Uint8List salt, KdfParams params) =>
      _deriveSplit(password, salt, params, 'auth', 'chat-wrap');

  /// The same construction over a recovery phrase (#2430): `authKey` is the
  /// `recoveryKey` the Quark is sent in the phrase's place, and `wrapKey`
  /// opens the chat identity's phrase wrap.
  ///
  /// ```text
  /// phraseMaster = Argon2id13(normalize(phrase), salt, params), 32 bytes
  /// recoveryKey  = HKDF-SHA256(phraseMaster, info = "recovery-auth"), 32 bytes
  /// wrapKey      = HKDF-SHA256(phraseMaster, info = "recovery-wrap"), 32 bytes
  /// ```
  ///
  /// [phrase] is normalized here ([normalizeRecoveryPhrase]), so a phrase
  /// typed with capitals derives the same keys. [salt] is the account's auth
  /// salt. The caller disposes the result.
  AuthKeys deriveRecoveryKeys(
    String phrase,
    Uint8List salt,
    KdfParams params,
  ) => _deriveSplit(
    normalizeRecoveryPhrase(phrase),
    salt,
    params,
    'recovery-auth',
    'recovery-wrap',
  );

  /// One Argon2id run over [secret], split by HKDF-SHA256 into a key sent
  /// under [authInfo] and a wrap key under [wrapInfo].
  AuthKeys _deriveSplit(
    String secret,
    Uint8List salt,
    KdfParams params,
    String authInfo,
    String wrapInfo,
  ) {
    final hkdf = sodium.crypto.kdfHkdfSha256;
    final master = _derive(secret, salt, params);
    try {
      final prk = master.runUnlockedSync((ikm) => hkdf.extract(ikm: ikm));
      try {
        final authKey = hkdf.expand(
          masterKey: prk,
          context: authInfo,
          outLen: 32,
        );
        try {
          return AuthKeys(
            authKey: base64Encode(authKey.extractBytes()),
            wrapKey: hkdf.expand(
              masterKey: prk,
              context: wrapInfo,
              outLen: _aead.keyBytes,
            ),
          );
        } finally {
          authKey.dispose();
        }
      } finally {
        prk.dispose();
      }
    } finally {
      master.dispose();
    }
  }

  /// Encrypts [secret] under Argon2id([passphrase], a fresh salt).
  ///
  /// The result is `nonce || XChaCha20-Poly1305(secret)`. [passphrase] is the
  /// recovery phrase, which the caller normalizes first, or the login password
  /// of an account still on the first scheme ([KdfParams.algorithm]).
  WrappedSecret wrap(Uint8List secret, String passphrase, KdfParams params) {
    final salt = sodium.randombytes.buf(sodium.crypto.pwhash.saltBytes);
    final key = _derive(passphrase, salt, params);
    try {
      return wrapWithKey(secret, key, salt);
    } finally {
      key.dispose();
    }
  }

  /// Encrypts [secret] directly under [key], with no Argon2id of its own:
  /// `nonce || XChaCha20-Poly1305(secret)`. [salt] is recorded beside the
  /// wrap, for whoever re-derives [key]. The caller disposes [key].
  WrappedSecret wrapWithKey(Uint8List secret, SecureKey key, Uint8List salt) {
    final nonce = sodium.randombytes.buf(_aead.nonceBytes);
    final cipherText = _aead.encrypt(message: secret, nonce: nonce, key: key);
    return WrappedSecret(
      wrapped: Uint8List.fromList([...nonce, ...cipherText]),
      salt: salt,
    );
  }

  /// Opens what [wrap] produced.
  ///
  /// A wrong [passphrase] throws a [MessageException] carrying [wrongSecret],
  /// so [Errors.message] shows it as written.
  Uint8List unwrap(
    WrappedSecret wrapped,
    String passphrase,
    KdfParams params, {
    required String wrongSecret,
  }) {
    final key = _derive(passphrase, wrapped.salt, params);
    try {
      return unwrapWithKey(wrapped, key, wrongSecret: wrongSecret);
    } finally {
      key.dispose();
    }
  }

  /// Opens what [wrapWithKey] produced. A wrong [key] throws a
  /// [MessageException] carrying [wrongSecret]. The caller disposes [key].
  Uint8List unwrapWithKey(
    WrappedSecret wrapped,
    SecureKey key, {
    required String wrongSecret,
  }) {
    final nonceBytes = _aead.nonceBytes;
    if (wrapped.wrapped.length <= nonceBytes) {
      throw MessageException(wrongSecret);
    }
    try {
      return _aead.decrypt(
        cipherText: Uint8List.sublistView(wrapped.wrapped, nonceBytes),
        nonce: Uint8List.sublistView(wrapped.wrapped, 0, nonceBytes),
        key: key,
      );
    } on SodiumException {
      throw MessageException(wrongSecret);
    }
  }

  SecureKey _derive(String secret, Uint8List salt, KdfParams params) =>
      sodium.crypto.pwhash.callStr(
        outLen: _aead.keyBytes,
        password: secret,
        salt: salt,
        opsLimit: params.opsLimit,
        memLimit: params.memLimit,
        alg: CryptoPwhashAlgorithm.argon2id13,
      );
}

/// What [AccountCrypto.deriveAuthKeys] makes of a password, or
/// [AccountCrypto.deriveRecoveryKeys] of a recovery phrase (#2430): the key
/// the Quark is sent in the secret's place, and the key that wraps the chat
/// identity and never leaves the client.
///
/// Neither is to be logged or kept past the request it was derived for.
/// [dispose] it once both have been used.
class AuthKeys {
  /// Pairs the two keys.
  AuthKeys({required this.authKey, required this.wrapKey});

  /// The standard padded base64 of 32 bytes: `authKey`, or `recoveryKey`, on
  /// the wire.
  final String authKey;

  /// The XChaCha20-Poly1305 key of the chat identity's password wrap, or its
  /// phrase wrap.
  final SecureKey wrapKey;

  /// Frees the wrap key.
  void dispose() => wrapKey.dispose();
}
