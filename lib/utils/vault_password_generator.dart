import 'package:quark/utils/vault_config.dart';

/// What the vault's password generator draws from (#3461), ported from the
/// Quark's old `POST /vault/generate`.
///
/// Out-of-range lengths are clamped to
/// [VaultConfig.minGeneratedPasswordLength]..[VaultConfig.maxGeneratedPasswordLength],
/// and switching every character class off falls back to lowercase and
/// digits, so the generator always makes something usable.
class PasswordOptions {
  /// Builds options; the defaults are every class at
  /// [VaultConfig.generatedPasswordLength].
  const PasswordOptions({
    this.length = VaultConfig.generatedPasswordLength,
    this.uppercase = true,
    this.lowercase = true,
    this.digits = true,
    this.symbols = true,
    this.avoidAmbiguous = false,
  });

  /// How many characters.
  final int length;

  /// Whether `A`-`Z` may appear.
  final bool uppercase;

  /// Whether `a`-`z` may appear.
  final bool lowercase;

  /// Whether `0`-`9` may appear.
  final bool digits;

  /// Whether the symbols in [symbolChars] may appear.
  final bool symbols;

  /// Whether to leave out characters that read alike: [ambiguousChars].
  final bool avoidAmbiguous;

  /// The symbols the generator may use.
  static const String symbolChars = '!@#\$%^&*()-_=+[]{}|;:,.<>?';

  /// Characters left out under [avoidAmbiguous].
  static const String ambiguousChars = '0O1lI';

  /// [length], clamped.
  int get clampedLength => length.clamp(
    VaultConfig.minGeneratedPasswordLength,
    VaultConfig.maxGeneratedPasswordLength,
  );

  /// The characters a password is drawn from.
  String get pool {
    final none = !uppercase && !lowercase && !digits && !symbols;
    final chars = [
      if (lowercase || none) 'abcdefghijklmnopqrstuvwxyz',
      if (uppercase) 'ABCDEFGHIJKLMNOPQRSTUVWXYZ',
      if (digits || none) '0123456789',
      if (symbols) symbolChars,
    ].join();
    return avoidAmbiguous
        ? chars.split('').where((c) => !ambiguousChars.contains(c)).join()
        : chars;
  }
}

/// A password per [options], with [draw] picking `length` characters from a
/// pool uniformly at random: `AccountCrypto.randomString`, libsodium's
/// secure random generator.
String generatePassword(
  String Function(String pool, int length) draw, [
  PasswordOptions options = const PasswordOptions(),
]) => draw(options.pool, options.clampedLength);
