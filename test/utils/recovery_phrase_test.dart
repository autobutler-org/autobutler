import 'dart:io';
import 'dart:math';
import 'dart:typed_data';

import 'package:flutter_test/flutter_test.dart';
import 'package:quark/models/chat_keys.dart';
import 'package:quark/services/chat_crypto.dart';
import 'package:quark/utils/recovery_phrase.dart';

/// #2430: the app generates the recovery phrase now, so its word list and
/// normalization have to be the Quark's.
void main() {
  test('the word list is the Go one, word for word', () {
    // `flutter test` runs from the repository root.
    final go = File('pkg/util/authutil/wordlist.go').readAsStringSync();
    final body = go.substring(go.indexOf('wordlist = []string{'));
    final words = [
      for (final match in RegExp(r'"([a-z]+)"').allMatches(body))
        match.group(1)!,
    ];

    expect(words, hasLength(256));
    expect(recoveryWords, words);
    expect(recoveryWords.toSet(), hasLength(256));
  });

  test('normalizing trims and lowercases, as NormalizeRecoveryPhrase does', () {
    expect(normalizeRecoveryPhrase('  Apple-BREAD\n'), 'apple-bread');
  });

  // BIP39's rule: the first four letters identify a word, so no two differ
  // only in their endings (#2783).
  test('no two words share their first four letters', () {
    String prefix(String word) => word.length < 4 ? word : word.substring(0, 4);

    expect(recoveryWords.map(prefix).toSet(), hasLength(recoveryWords.length));
  });

  test('a new phrase carries at least 128 bits (#2783)', () {
    final bitsPerWord = log(recoveryWords.length) / ln2;

    expect(bitsPerWord, 8);
    expect(recoveryPhraseWords, 16);
    expect(recoveryPhraseWords * bitsPerWord, greaterThanOrEqualTo(128));
    expect(legacyRecoveryPhraseWords * bitsPerWord, 48);
  });

  test('a legacy six-word phrase still derives keys, the same ones '
      'however it is typed (#2783)', () async {
    final crypto = await ChatCrypto.load();
    final salt = Uint8List(16);
    const params = KdfParams(opsLimit: 1, memLimit: 8192);
    const legacy = 'abandon-ability-able-about-above-absent';
    expect(legacy.split('-'), hasLength(legacyRecoveryPhraseWords));

    final clean = crypto.deriveRecoveryKeys(legacy, salt, params);
    final typed = crypto.deriveRecoveryKeys(
      ' ABANDON-ability-able-about-above-absent\n',
      salt,
      params,
    );
    final longer = crypto.deriveRecoveryKeys(
      crypto.generateRecoveryPhrase(),
      salt,
      params,
    );
    addTearDown(clean.dispose);
    addTearDown(typed.dispose);
    addTearDown(longer.dispose);

    expect(typed.authKey, clean.authKey);
    expect(longer.authKey, isNot(clean.authKey));
  });

  test(
    'a generated phrase is sixteen listed words joined by hyphens',
    () async {
      final crypto = await ChatCrypto.load();
      final phrases = {
        for (var i = 0; i < 8; i++) crypto.generateRecoveryPhrase(),
      };

      expect(phrases, hasLength(8));
      for (final phrase in phrases) {
        final words = phrase.split('-');
        expect(words, hasLength(recoveryPhraseWords));
        expect(words.every(recoveryWords.contains), isTrue, reason: phrase);
        expect(normalizeRecoveryPhrase(phrase), phrase);
      }
    },
  );
}
