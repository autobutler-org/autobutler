import 'dart:convert';
import 'dart:typed_data';

import 'package:quark/models/account_keys.dart';

/// A user's published chat keys: what any signed-in user may fetch.
class ChatPublicKeys {
  /// Pairs the two public keys.
  const ChatPublicKeys({
    required this.boxPublicKey,
    required this.signPublicKey,
  });

  /// X25519, what key grants are sealed to.
  final Uint8List boxPublicKey;

  /// Ed25519, what the user's signatures verify against.
  final Uint8List signPublicKey;

  /// Reads `GET /api/v0/chat/keys/:userId`.
  factory ChatPublicKeys.fromJson(Map<String, dynamic> json) => ChatPublicKeys(
    boxPublicKey: base64Decode(json['boxPublicKey'] as String),
    signPublicKey: base64Decode(json['signPublicKey'] as String),
  );
}

/// Everything the Quark stores for one user's chat identity (#2416): the
/// public keys, and the private seeds wrapped under the login password and,
/// when the client had it, the recovery phrase.
class WrappedChatKeys {
  /// Builds the stored form.
  const WrappedChatKeys({
    required this.publicKeys,
    required this.byPassword,
    required this.byPhrase,
    required this.kdfParams,
  });

  /// The public halves.
  final ChatPublicKeys publicKeys;

  /// Wrapped under the login password: directly under its Argon2id, or under
  /// the `wrapKey` derived from it when [kdfParams] is [KdfParams.isSplit].
  final WrappedSecret byPassword;

  /// Wrapped under the recovery phrase: directly under its Argon2id, or under
  /// the `wrapKey` derived from it when [kdfParams] is
  /// [KdfParams.isPhraseSplit]. Null for keys made at a sign-in that had no
  /// phrase to hand, which recovery cannot open.
  final WrappedSecret? byPhrase;

  /// The Argon2id cost both wraps used, and the scheme of each.
  final KdfParams kdfParams;

  /// Reads `GET /api/v0/chat/keys/me`.
  factory WrappedChatKeys.fromJson(Map<String, dynamic> json) {
    final byPhrase = json['wrappedByPhrase'] as String?;
    final saltRp = json['saltRp'] as String?;
    return WrappedChatKeys(
      publicKeys: ChatPublicKeys.fromJson(json),
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

  /// The body of `PUT /api/v0/chat/keys/me` and of `chatKeys` in
  /// `POST /api/v0/auth/recover`.
  Map<String, dynamic> toJson() => {
    'boxPublicKey': base64Encode(publicKeys.boxPublicKey),
    'signPublicKey': base64Encode(publicKeys.signPublicKey),
    'wrappedByPassword': base64Encode(byPassword.wrapped),
    'saltPw': base64Encode(byPassword.salt),
    if (byPhrase != null) ...{
      'wrappedByPhrase': base64Encode(byPhrase!.wrapped),
      'saltRp': base64Encode(byPhrase!.salt),
    },
    'kdfParams': kdfParams.toJson(),
  };
}
