import 'dart:convert';

import 'package:quark/models/vault_item.dart';
import 'package:quark/utils/error_text.dart';
import 'package:quark/utils/vault_config.dart';

/// One login read from another password manager's export, before it is
/// encrypted into the vault (#2543, #3461). [folder] is a folder name, made
/// on demand.
typedef ImportedEntry = ({
  String name,
  String url,
  String username,
  String password,
  String notes,
  String totpSecret,
  String folder,
});

/// What an export file held: the logins it could read, one line per row it
/// could not, and how many items were left out because the vault has no
/// place for them (Proton Pass notes, aliases, cards and identities).
typedef ParsedImport = ({
  List<ImportedEntry> entries,
  List<String> errors,
  int ignored,
});

/// The export formats [parseVaultImport] reads, ported from the Quark's old
/// `vaultutil` import.
enum VaultImportFormat {
  /// Quark's own JSON export, as an object or a bare array of entries.
  json,

  /// A Bitwarden CSV export.
  bitwarden,

  /// A Proton Pass CSV export.
  proton,

  /// Any other CSV with a header row: Google Password Manager, browsers.
  csv,
}

/// Reads an import file from [bytes], refusing anything over
/// [VaultConfig.maxImportBytes] with [Errors.vaultImportTooLarge] as soon as
/// the stream passes it, so an oversized file is never read whole.
Future<String> readVaultImport(Stream<List<int>> bytes) async {
  final chunks = <List<int>>[];
  var total = 0;
  await for (final chunk in bytes) {
    total += chunk.length;
    if (total > VaultConfig.maxImportBytes) {
      throw const MessageException(Errors.vaultImportTooLarge);
    }
    chunks.add(chunk);
  }
  return utf8.decode(chunks.expand((c) => c).toList(), allowMalformed: true);
}

/// Guesses [text]'s format: JSON by its opening brace or bracket, then
/// Bitwarden and Proton Pass by their CSV headers, and any other CSV.
VaultImportFormat detectVaultImportFormat(String text) {
  final trimmed = text.trim();
  if (trimmed.startsWith('{') || trimmed.startsWith('[')) {
    return VaultImportFormat.json;
  }
  final firstLine = trimmed.split('\n').first.toLowerCase();
  if (firstLine.contains('login_uri') || firstLine.contains('login_username')) {
    return VaultImportFormat.bitwarden;
  }
  final columns = _headerIndex(firstLine.split(','));
  if (['type', 'email', 'totp', 'vault'].every(columns.containsKey)) {
    return VaultImportFormat.proton;
  }
  return VaultImportFormat.csv;
}

/// Parses an export file, sniffing its format with [detectVaultImportFormat]
/// unless [format] is given.
ParsedImport parseVaultImport(String text, [VaultImportFormat? format]) =>
    switch (format ?? detectVaultImportFormat(text)) {
      VaultImportFormat.json => _parseQuarkJson(text),
      VaultImportFormat.bitwarden => _parseBitwarden(text),
      VaultImportFormat.proton => _parseProtonPass(text),
      VaultImportFormat.csv => _parseGenericCsv(text),
    };

/// Splits [text] into CSV records (RFC 4180): commas, double-quoted fields
/// with `""` for a quote, and line breaks inside quotes. A trailing empty
/// line is not a record. Throws a [FormatException] on an unclosed quote.
List<List<String>> parseCsv(String text) {
  final records = <List<String>>[];
  var record = <String>[];
  final field = StringBuffer();
  var quoted = false;
  var i = 0;
  void endField() {
    record.add(field.toString());
    field.clear();
  }

  void endRecord() {
    endField();
    if (record.length > 1 || record.first.isNotEmpty) records.add(record);
    record = <String>[];
  }

  while (i < text.length) {
    final c = text[i];
    if (quoted) {
      if (c == '"') {
        if (i + 1 < text.length && text[i + 1] == '"') {
          field.write('"');
          i++;
        } else {
          quoted = false;
        }
      } else {
        field.write(c);
      }
    } else if (c == '"' && field.isEmpty) {
      quoted = true;
    } else if (c == ',') {
      endField();
    } else if (c == '\n' || c == '\r') {
      if (c == '\r' && i + 1 < text.length && text[i + 1] == '\n') i++;
      endRecord();
    } else {
      field.write(c);
    }
    i++;
  }
  if (quoted) throw const FormatException('unclosed quote');
  if (field.isNotEmpty || record.isNotEmpty) endRecord();
  return records;
}

/// One CSV export's records, with its header row read into column positions.
/// Throws a [FormatException] carrying the line [_parseCsvRows] reports when
/// there is nothing to import.
({Map<String, int> header, List<List<String>> rows}) _csvRows(String text) {
  final List<List<String>> records;
  try {
    records = parseCsv(text);
  } on FormatException catch (e) {
    throw FormatException('CSV parse error: ${e.message}');
  }
  if (records.length < 2) throw const FormatException('CSV has no data rows');
  return (header: _headerIndex(records.first), rows: records.sublist(1));
}

/// Runs [rowToEntry] over every data row of a CSV export. A row it returns a
/// string for is reported as that error, and null skips it as ignored.
ParsedImport _parseCsvRows(
  String text,
  Object? Function(List<String> row, Map<String, int> header, int line)
  rowToEntry,
) {
  final ({Map<String, int> header, List<List<String>> rows}) csv;
  try {
    csv = _csvRows(text);
  } on FormatException catch (e) {
    return (entries: const [], errors: [e.message], ignored: 0);
  }
  final entries = <ImportedEntry>[];
  final errors = <String>[];
  var ignored = 0;
  for (final (i, row) in csv.rows.indexed) {
    switch (rowToEntry(row, csv.header, i + 2)) {
      case final ImportedEntry entry:
        entries.add(entry);
      case final String error:
        errors.add(error);
      default:
        ignored++;
    }
  }
  return (entries: entries, errors: errors, ignored: ignored);
}

ParsedImport _parseQuarkJson(String text) {
  final Object? decoded;
  try {
    decoded = jsonDecode(text);
  } on FormatException catch (e) {
    return (
      entries: const [],
      errors: ['invalid JSON: ${e.message}'],
      ignored: 0,
    );
  }
  final list = switch (decoded) {
    {'entries': final List entries} => entries,
    final List entries => entries,
    _ => const [],
  };
  return (
    entries: [
      for (final e in list.whereType<Map>())
        (
          name: '${e['name'] ?? ''}',
          url: '${e['url'] ?? ''}',
          username: '${e['username'] ?? ''}',
          password: '${e['password'] ?? ''}',
          notes: '${e['notes'] ?? ''}',
          totpSecret: '${e['totpSecret'] ?? ''}',
          folder: '${e['folderName'] ?? ''}',
        ),
    ],
    errors: const [],
    ignored: 0,
  );
}

ParsedImport _parseBitwarden(String text) =>
    _parseCsvRows(text, (row, header, line) {
      final url = _col(row, header, ['login_uri']);
      var name = _col(row, header, ['name']);
      if (name.isEmpty && url.isEmpty) return 'Row $line: missing name and URL';
      if (name.isEmpty) name = _orElse(VaultItemData.hostOf(url), url);
      return (
        name: name,
        url: url,
        username: _col(row, header, ['login_username']),
        password: _col(row, header, ['login_password']),
        notes: _col(row, header, ['notes']),
        totpSecret: _col(row, header, ['login_totp']),
        folder: _col(row, header, ['folder']),
      );
    });

/// Only logins become entries. A login saved with an email and no username
/// keeps the email, the first of several URLs is used, and the Proton vault
/// becomes the folder.
ParsedImport _parseProtonPass(String text) =>
    _parseCsvRows(text, (row, header, line) {
      final kind = _col(row, header, ['type']).toLowerCase();
      if (kind.isNotEmpty && kind != 'login') return null;
      final url = _firstUrl(_col(row, header, ['url']));
      final username = _col(row, header, ['username', 'email']);
      final password = _col(row, header, ['password']);
      if (username.isEmpty && password.isEmpty) {
        return 'Row $line: login has no username or password';
      }
      return (
        name: _orElse(
          _col(row, header, ['name']),
          _orElse(VaultItemData.hostOf(url), 'Entry $line'),
        ),
        url: url,
        username: username,
        password: password,
        notes: _col(row, header, ['note']),
        totpSecret: _col(row, header, ['totp']),
        folder: _col(row, header, ['vault']),
      );
    });

/// Accepts the several spellings each column goes by across browsers and
/// managers.
ParsedImport _parseGenericCsv(String text) => _parseCsvRows(text, (
  row,
  header,
  line,
) {
  final url = _col(row, header, ['url', 'website', 'uri', 'login_uri']);
  final password = _col(row, header, ['password', 'pass', 'login_password']);
  if (password.isEmpty) return 'Row $line: missing password';
  return (
    name: _orElse(
      _col(row, header, ['name', 'title']),
      _orElse(VaultItemData.hostOf(url), 'Entry $line'),
    ),
    url: url,
    username: _col(row, header, [
      'username',
      'user',
      'login',
      'login_username',
    ]),
    password: password,
    notes: _col(row, header, ['notes', 'note', 'comments']),
    totpSecret: '',
    folder: '',
  );
});

Map<String, int> _headerIndex(List<String> header) => {
  for (final (i, h) in header.indexed) h.trim().toLowerCase(): i,
};

/// The first non-empty column of [keys] in [row].
String _col(List<String> row, Map<String, int> header, List<String> keys) {
  for (final key in keys) {
    final i = header[key];
    if (i != null && i < row.length && row[i].trim().isNotEmpty) {
      return row[i].trim();
    }
  }
  return '';
}

String _orElse(String value, String fallback) =>
    value.isEmpty ? fallback : value;

/// The first URL of a list Proton Pass joins with commas or spaces.
String _firstUrl(String value) => value
    .split(RegExp('[, ]'))
    .firstWhere((s) => s.isNotEmpty, orElse: () => '');
