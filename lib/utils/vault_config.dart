/// The vault's tuning values (#3461), now that the client does what the
/// Quark's vault did.
abstract final class VaultConfig {
  /// How long an unlocked vault sits untouched before it locks itself and
  /// forgets its key. The server-side vault's default was 900 seconds.
  static const Duration autoLockAfter = Duration(minutes: 15);

  /// The largest export file an import reads. Parsing works on the whole
  /// document, so it is read into memory and has to be bounded; an export of
  /// tens of thousands of logins is a few megabytes.
  static const int maxImportBytes = 10 << 20;

  /// The length the password generator uses unless asked for another.
  static const int generatedPasswordLength = 20;

  /// The shortest password the generator makes; a shorter request is raised.
  static const int minGeneratedPasswordLength = 8;

  /// The longest password the generator makes; a longer request is lowered.
  static const int maxGeneratedPasswordLength = 128;
}
