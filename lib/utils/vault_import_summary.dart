/// What a vault import did (#2543, #3461). [skipped] counts entries already
/// in the vault, [ignored] items the vault has no place for (Proton Pass
/// notes, aliases and cards), and each of [errors] is a row the import could
/// not take.
typedef VaultImportResult = ({
  int imported,
  int skipped,
  int ignored,
  List<String> errors,
});

/// The one-line result of a vault import, shown in a snack bar. A count of
/// zero says nothing, so only the ones that happened are mentioned.
String vaultImportSummary(VaultImportResult result) => [
  'Imported ${result.imported} entries',
  if (result.skipped > 0) '${result.skipped} skipped (duplicates)',
  if (result.ignored > 0) '${result.ignored} non-login items left out',
  if (result.errors.isNotEmpty) '${result.errors.length} errors',
].join(', ');
