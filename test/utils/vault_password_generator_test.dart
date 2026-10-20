import 'package:flutter_test/flutter_test.dart';
import 'package:quark/services/account_crypto.dart';
import 'package:quark/utils/vault_password_generator.dart';

/// #3461: the generator moved from the Quark's `POST /vault/generate` to the
/// client, on libsodium's secure random generator.
void main() {
  late AccountCrypto crypto;

  setUpAll(() async => crypto = await AccountCrypto.load());

  test('the default draws 20 characters from every class', () {
    const options = PasswordOptions();
    expect(options.pool, hasLength(26 + 26 + 10 + 26));

    final password = generatePassword(crypto.randomString);
    expect(password, hasLength(20));
    expect(password.split('').every(options.pool.contains), isTrue);
  });

  test('clamps the length', () {
    expect(const PasswordOptions(length: 2).clampedLength, 8);
    expect(const PasswordOptions(length: 1000).clampedLength, 128);
    expect(
      generatePassword(crypto.randomString, const PasswordOptions(length: 64)),
      hasLength(64),
    );
  });

  test('draws only from the classes asked for', () {
    const digits = PasswordOptions(
      uppercase: false,
      lowercase: false,
      symbols: false,
    );
    expect(digits.pool, '0123456789');
    expect(
      generatePassword(crypto.randomString, digits),
      matches(RegExp(r'^[0-9]{20}$')),
    );
  });

  test('every class off falls back to lowercase and digits', () {
    const none = PasswordOptions(
      uppercase: false,
      lowercase: false,
      digits: false,
      symbols: false,
    );
    expect(none.pool, 'abcdefghijklmnopqrstuvwxyz0123456789');
  });

  test('leaves out characters that read alike', () {
    final pool = const PasswordOptions(avoidAmbiguous: true).pool;
    for (final c in PasswordOptions.ambiguousChars.split('')) {
      expect(pool, isNot(contains(c)));
    }
  });
}
