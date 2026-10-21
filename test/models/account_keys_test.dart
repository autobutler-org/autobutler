import 'dart:convert';
import 'dart:typed_data';

import 'package:flutter_test/flutter_test.dart';
import 'package:quark/models/account_keys.dart';
import 'package:quark/models/chat_keys.dart';
import 'package:quark/models/vault_key.dart';
import 'package:quark/utils/error_text.dart';

/// A `kdfParams` object at [opsLimit] passes over [memLimit] bytes.
Map<String, dynamic> _json(
  int opsLimit,
  int memLimit, {
  String alg = KdfParams.splitAlgorithm,
}) => {'alg': alg, 'opsLimit': opsLimit, 'memLimit': memLimit};

/// #3648: a stored cost below the app's standard or above its ceiling is
/// refused when it is read, before anything is derived at it.
void main() {
  const standard = KdfParams.standard;
  const ceiling = KdfParams.ceiling;

  group('KdfParams.fromJson', () {
    for (final (name, json) in [
      ('one pass too few', _json(standard.opsLimit - 1, standard.memLimit)),
      ('a byte too little', _json(standard.opsLimit, standard.memLimit - 1)),
      ("libsodium's floor", _json(1, 8192)),
      ('one pass too many', _json(ceiling.opsLimit + 1, standard.memLimit)),
      ('a byte too much', _json(standard.opsLimit, ceiling.memLimit + 1)),
    ]) {
      test('refuses $name', () {
        expect(
          () => KdfParams.fromJson(json),
          throwsA(isA<KdfParamsOutOfRange>()),
        );
      });
    }

    test('accepts the standard and the ceiling, in every scheme', () {
      for (final alg in [
        KdfParams.algorithm,
        KdfParams.splitAlgorithm,
        KdfParams.phraseSplitAlgorithm,
      ]) {
        for (final cost in [standard, ceiling]) {
          final read = KdfParams.fromJson(
            _json(cost.opsLimit, cost.memLimit, alg: alg),
          );
          expect(read.toJson(), _json(cost.opsLimit, cost.memLimit, alg: alg));
        }
      }
    });

    test('the refusal reads as the key-cost sentence, not the fallback', () {
      Object? error;
      try {
        KdfParams.fromJson(_json(1, 8192));
      } catch (e) {
        error = e;
      }
      expect(
        Errors.message(error, 'unlock your vault'),
        Errors.keyCostOutOfRange,
      );
    });
  });

  group('a stored key at a cost below the standard is refused', () {
    final wrap = base64Encode(Uint8List(72));
    final salt = base64Encode(Uint8List(16));

    test('for the vault', () {
      expect(
        () => WrappedVaultKey.fromJson({
          'wrappedByPassword': wrap,
          'saltPw': salt,
          'kdfParams': _json(1, 8192),
        }),
        throwsA(isA<KdfParamsOutOfRange>()),
      );
    });

    test('for chat', () {
      expect(
        () => WrappedChatKeys.fromJson({
          'boxPublicKey': base64Encode(Uint8List(32)),
          'signPublicKey': base64Encode(Uint8List(32)),
          'wrappedByPassword': wrap,
          'saltPw': salt,
          'kdfParams': _json(1, 8192),
        }),
        throwsA(isA<KdfParamsOutOfRange>()),
      );
    });
  });

  group('KdfParams.atLeast', () {
    test('raises each half that is below the floor, keeping the scheme', () {
      const weak = KdfParams(
        opsLimit: 1,
        memLimit: 128 << 20,
        alg: KdfParams.phraseSplitAlgorithm,
      );
      final raised = weak.atLeast(standard);
      expect(raised.opsLimit, standard.opsLimit);
      expect(raised.memLimit, 128 << 20);
      expect(raised.alg, KdfParams.phraseSplitAlgorithm);
    });

    test('leaves a cost at or above the floor as it is', () {
      const strong = KdfParams(opsLimit: 4, memLimit: 128 << 20);
      expect(strong.atLeast(standard).toJson(), strong.toJson());
    });
  });
}
