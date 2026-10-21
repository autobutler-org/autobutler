import 'dart:math';
import 'dart:typed_data';

/// The Argon2id cost a wrapped account secret was derived with, stored beside
/// it as `kdfParams` so the cost can rise later without breaking older wraps.
///
/// [alg] names the chat identity's wrap schemes, which existing accounts
/// depend on and which never change.
class KdfParams {
  /// Builds explicit parameters. [memLimit] is in bytes.
  const KdfParams({
    required this.opsLimit,
    required this.memLimit,
    this.alg = algorithm,
  });

  /// Argon2id passes.
  final int opsLimit;

  /// Argon2id memory, in bytes.
  final int memLimit;

  /// What new wraps use: 3 passes over 64 MiB.
  ///
  /// Measured with the shipped sodium.js (sumo 0.8.4) in Chrome on an Apple
  /// M2 Pro: 100 ms. A mid-range phone's browser is roughly five times slower,
  /// which keeps an unlock under the one-second budget in #2416. 128 MiB took
  /// 220 ms and 256 MiB 456 ms on the same machine.
  static const standard = KdfParams(opsLimit: 3, memLimit: 64 << 20);

  /// The most a stored cost may ask for (#3648): 10 passes over 1 GiB, so a
  /// Quark cannot make an unlock hang or run a phone out of memory.
  static const ceiling = KdfParams(opsLimit: 10, memLimit: 1 << 30);

  /// How the wraps' keys come out of Argon2id: [algorithm],
  /// [splitAlgorithm] or [phraseSplitAlgorithm].
  final String alg;

  /// The first scheme (#2416): each wrap's key is Argon2id of its secret and
  /// its own random salt, used directly.
  static const algorithm = 'argon2id13';

  /// The split-key scheme (#2430). The phrase wrap is as in [algorithm]. The
  /// password wrap's key is the `wrapKey` of `AccountCrypto.deriveAuthKeys`, and
  /// its salt is the account's auth salt, so the password alone re-derives it.
  static const splitAlgorithm = 'argon2id13+hkdf-sha256';

  /// [splitAlgorithm] for the phrase wrap too (#2430): its key is the
  /// `wrapKey` of `AccountCrypto.deriveRecoveryKeys`, and its salt is the auth
  /// salt. The password wrap is as in [splitAlgorithm].
  static const phraseSplitAlgorithm = 'argon2id13+hkdf-sha256+recovery';

  /// Whether the password wrap is under the split-key scheme's `wrapKey`.
  bool get isSplit => alg == splitAlgorithm || isPhraseSplit;

  /// Whether the phrase wrap is under `AccountCrypto.deriveRecoveryKeys`'s
  /// `wrapKey`.
  bool get isPhraseSplit => alg == phraseSplitAlgorithm;

  /// The same cost under [splitAlgorithm].
  KdfParams get split => _as(splitAlgorithm);

  /// The same cost under [phraseSplitAlgorithm].
  KdfParams get phraseSplit => _as(phraseSplitAlgorithm);

  KdfParams _as(String alg) =>
      KdfParams(opsLimit: opsLimit, memLimit: memLimit, alg: alg);

  /// Whether this cost is at least [floor]'s, in passes and in memory.
  bool isAtLeast(KdfParams floor) =>
      opsLimit >= floor.opsLimit && memLimit >= floor.memLimit;

  /// This cost under the same [alg], raised to [floor]'s wherever it is
  /// below it: what a re-wrap uses, so it never copies a weaker stored cost
  /// than the app's own (#3648).
  KdfParams atLeast(KdfParams floor) => KdfParams(
    opsLimit: max(opsLimit, floor.opsLimit),
    memLimit: max(memLimit, floor.memLimit),
    alg: alg,
  );

  /// Reads `kdfParams`. Anything but the three algorithms above is refused
  /// rather than guessed at, and so, with a [KdfParamsOutOfRange], is a cost
  /// below [standard] or above [ceiling]: every wrap the app has made is at
  /// [standard], so a weaker one is a Quark asking for a cheaper wrap.
  factory KdfParams.fromJson(Map<String, dynamic> json) {
    final alg = json['alg'];
    if (alg != algorithm &&
        alg != splitAlgorithm &&
        alg != phraseSplitAlgorithm) {
      throw FormatException('unsupported chat key KDF: $alg');
    }
    final params = KdfParams(
      opsLimit: (json['opsLimit'] as num).toInt(),
      memLimit: (json['memLimit'] as num).toInt(),
      alg: alg as String,
    );
    if (!params.isAtLeast(standard) || !ceiling.isAtLeast(params)) {
      throw KdfParamsOutOfRange(params);
    }
    return params;
  }

  /// The `kdfParams` object.
  Map<String, dynamic> toJson() => {
    'alg': alg,
    'opsLimit': opsLimit,
    'memLimit': memLimit,
  };
}

/// A stored `kdfParams` whose cost is outside [KdfParams.standard] to
/// [KdfParams.ceiling], refused before anything is derived at it (#3648).
/// `Errors.message` reads it as `Errors.keyCostOutOfRange`.
class KdfParamsOutOfRange extends FormatException {
  /// Refuses [params].
  KdfParamsOutOfRange(this.params)
    : super(
        'key derivation cost out of range: ${params.opsLimit} passes over '
        '${params.memLimit} bytes',
      );

  /// The refused cost.
  final KdfParams params;
}

/// One wrap of an account secret (a chat identity's seeds): `nonce ||
/// ciphertext` and the Argon2id salt its key was derived with.
class WrappedSecret {
  /// Pairs a wrap with its salt.
  const WrappedSecret({required this.wrapped, required this.salt});

  /// `nonce || XChaCha20-Poly1305(secret)`.
  final Uint8List wrapped;

  /// The Argon2id salt.
  final Uint8List salt;
}
