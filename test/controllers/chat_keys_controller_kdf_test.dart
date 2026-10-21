import 'dart:typed_data';

import 'package:flutter_test/flutter_test.dart';
import 'package:quark/controllers/chat_keys_controller.dart';
import 'package:quark/models/account_keys.dart';
import 'package:quark/models/chat_keys.dart';
import 'package:quark/services/chat_crypto.dart';

/// Argon2id's floor: the cost a tampered Quark lowered the stored keys to.
const _tampered = KdfParams(opsLimit: 1, memLimit: 8192);

/// The cost the controller re-wraps at, standing in for
/// [KdfParams.standard] so the test stays fast.
const _floor = KdfParams(opsLimit: 2, memLimit: 16384);

final _salt = Uint8List.fromList(List.generate(16, (i) => i));

/// #3648: a re-wrap of the chat identity never copies a stored cost below
/// the one the app wraps at.
void main() {
  late ChatCrypto crypto;
  WrappedChatKeys? stored;

  setUpAll(() async => crypto = await ChatCrypto.load());

  ChatKeysController controller(KdfParams kdfParams) => ChatKeysController(
    loadCrypto: () async => crypto,
    fetchMine: ({String? sessionToken}) async => stored,
    putMine: (keys, {String? sessionToken}) async => stored = keys,
    activeHost: () => 'one',
    persist: false,
    kdfParams: kdfParams,
  );

  setUp(() async {
    stored = null;
    // Keys made at the tampered cost, as a first sign-in at it would.
    await controller(_tampered).signedIn(password: 'pw-one', authSalt: _salt);
    expect(stored!.kdfParams.opsLimit, _tampered.opsLimit);
  });

  test(
    'a password change re-wraps at the floor, not the stored cost',
    () async {
      final box = stored!.publicKeys.boxPublicKey;

      final rewrapped = (await controller(_floor).keysForPasswordChange(
        currentPassword: 'pw-one',
        newPassword: 'pw-two',
        authSalt: _salt,
      ))!;

      expect(rewrapped.kdfParams.opsLimit, _floor.opsLimit);
      expect(rewrapped.kdfParams.memLimit, _floor.memLimit);
      expect(rewrapped.kdfParams.alg, KdfParams.splitAlgorithm);
      stored = rewrapped;
      final after = controller(_floor);
      await after.signedIn(password: 'pw-two', authSalt: _salt);
      expect(after.identity!.box.publicKey, box);
    },
  );

  test('a first-scheme upgrade at sign-in re-wraps at the floor', () async {
    final identity = crypto.generateIdentity();
    stored = WrappedChatKeys(
      publicKeys: identity.publicKeys,
      byPassword: crypto.wrap(identity, 'pw-one', _tampered),
      byPhrase: null,
      kdfParams: _tampered,
    );

    await controller(_floor).signedIn(password: 'pw-one', authSalt: _salt);

    expect(stored!.kdfParams.opsLimit, _floor.opsLimit);
    expect(stored!.kdfParams.memLimit, _floor.memLimit);
    expect(stored!.kdfParams.alg, KdfParams.splitAlgorithm);
  });
}
