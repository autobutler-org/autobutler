import 'dart:convert';

import 'package:flutter_test/flutter_test.dart';
import 'package:quark/models/vault_item.dart';
import 'package:quark/utils/error_text.dart';
import 'package:quark/utils/vault_config.dart';
import 'package:quark/utils/vault_export.dart';
import 'package:quark/utils/vault_import.dart';

/// #3461: the import parsers moved from the Quark's `vaultutil` to the
/// client. These are its tests, ported.
void main() {
  const protonPassExport =
      'type,name,url,email,username,password,note,totp,createTime,modifyTime,vault\n'
      'login,GitHub,"https://github.com/login, https://gist.github.com",alice@example.com,alice,gh-pass,work account,JBSWY3DPEHPK3PXP,1700000000,1700000000,Work\n'
      'login,Bank,https://bank.example,bob@example.com,,bank-pass,,,1700000000,1700000000,Personal\n'
      'note,Wi-Fi code,,,,,door code 1234,,1700000000,1700000000,Personal\n'
      'alias,Newsletter alias,,news@alias.example,,,,,1700000000,1700000000,Personal\n'
      'creditCard,Visa,,,,,,,1700000000,1700000000,Personal\n'
      'login,Empty,https://empty.example,,,,,,1700000000,1700000000,Personal\n';

  const googleExport =
      'name,url,username,password,note\n'
      'github.com,https://github.com/login,alice,gh-pass,\n'
      ',https://accounts.example.com/,carol,pw,old laptop\n';

  const bitwardenExport =
      'folder,favorite,type,name,notes,fields,reprompt,login_uri,login_username,login_password,login_totp\n'
      'Social,,login,GitHub,some notes,,0,https://github.com/login,alice,gh-pass,JBSWY3DPEHPK3PXP\n'
      'Banking,,login,Chase,,,,https://chase.com,bob,chase-pw,\n';

  group('detects', () {
    for (final (name, text, format) in [
      ('a JSON object', '{"entries": []}', VaultImportFormat.json),
      ('a JSON array', '[{"name": "test"}]', VaultImportFormat.json),
      ('Bitwarden', bitwardenExport, VaultImportFormat.bitwarden),
      ('Proton Pass', protonPassExport, VaultImportFormat.proton),
      ('Google', googleExport, VaultImportFormat.csv),
      (
        'any other CSV',
        'url,username,password\nhttps://example.com,alice,secret\n',
        VaultImportFormat.csv,
      ),
    ]) {
      test(name, () => expect(detectVaultImportFormat(text), format));
    }
  });

  test('Bitwarden keeps every column', () {
    final parsed = parseVaultImport(bitwardenExport);

    expect(parsed.errors, isEmpty);
    expect(parsed.entries, hasLength(2));
    final gh = parsed.entries.first;
    expect(gh.name, 'GitHub');
    expect(gh.url, 'https://github.com/login');
    expect(gh.username, 'alice');
    expect(gh.password, 'gh-pass');
    expect(gh.notes, 'some notes');
    expect(gh.totpSecret, 'JBSWY3DPEHPK3PXP');
    expect(gh.folder, 'Social');
  });

  test('Proton Pass keeps logins and counts the rest', () {
    final parsed = parseVaultImport(protonPassExport);

    expect(parsed.ignored, 3);
    expect(parsed.errors, hasLength(1));
    expect(parsed.entries, hasLength(2));
    final gh = parsed.entries.first;
    expect(gh.url, 'https://github.com/login');
    expect(gh.username, 'alice');
    expect(gh.totpSecret, 'JBSWY3DPEHPK3PXP');
    expect(gh.folder, 'Work');
    expect(gh.notes, 'work account');
    expect(parsed.entries.last.username, 'bob@example.com');
  });

  test('a generic CSV names an entry by its host, and needs a password', () {
    final google = parseVaultImport(googleExport);
    expect(google.errors, isEmpty);
    expect(google.entries.first.name, 'github.com');
    expect(google.entries.last.name, 'accounts.example.com');
    expect(google.entries.last.notes, 'old laptop');

    final missing = parseVaultImport(
      'url,username,password\nhttps://example.com,alice,\n',
    );
    expect(missing.entries, isEmpty);
    expect(missing.errors, ['Row 2: missing password']);
  });

  test('Quark JSON reads the object and the bare array', () {
    final object = parseVaultImport(
      jsonEncode({
        'entries': [
          {'name': 'GitHub', 'password': 'p', 'folderName': 'Work'},
        ],
        'folders': ['Work'],
      }),
    );
    expect(object.entries.single.folder, 'Work');

    final array = parseVaultImport('[{"name": "A"}, {"name": "B"}]');
    expect(array.entries.map((e) => e.name), ['A', 'B']);

    expect(parseVaultImport('{not json').errors, hasLength(1));
  });

  test('a CSV with no rows, or an open quote, is one error', () {
    expect(parseVaultImport('url,username,password\n').errors, [
      'CSV has no data rows',
    ]);
    expect(
      parseVaultImport('name,password\n"open,pw\n').errors.single,
      startsWith('CSV parse error'),
    );
  });

  test('parseCsv reads quotes, escaped quotes, CRLF and line breaks', () {
    expect(parseCsv('a,"b,c","say ""hi"""\r\n"multi\nline",,x\n'), [
      ['a', 'b,c', 'say "hi"'],
      ['multi\nline', '', 'x'],
    ]);
  });

  test('hostOf reads URLs with and without a scheme', () {
    for (final (url, host) in [
      ('https://github.com/login', 'github.com'),
      ('http://localhost:8080/path', 'localhost'),
      ('', ''),
      ('not a url', ''),
      ('github.com', 'github.com'),
      ('github.com/login', 'github.com'),
      ('  www.amazon.com  ', 'www.amazon.com'),
      ('192.168.1.1', '192.168.1.1'),
      ('https://GitHub.com/login', 'github.com'),
    ]) {
      expect(VaultItemData.hostOf(url), host, reason: url);
    }
  });

  group('reading an import file', () {
    test('takes one in chunks', () async {
      final text = await readVaultImport(
        Stream.fromIterable([utf8.encode('pass'), utf8.encode('€')]),
      );
      expect(text, 'pass€');
    });

    test('refuses one over the limit as soon as it passes it', () async {
      var chunks = 0;
      Stream<List<int>> huge() async* {
        while (true) {
          chunks++;
          yield List.filled(1 << 20, 0x61);
        }
      }

      await expectLater(
        readVaultImport(huge()),
        throwsA(
          isA<MessageException>().having(
            (e) => e.message,
            'message',
            Errors.vaultImportTooLarge,
          ),
        ),
      );
      expect(chunks, (VaultConfig.maxImportBytes >> 20) + 1);
    });
  });

  test('a CSV export quotes what needs it and names folders', () {
    final csv = vaultExport(const [
      VaultItem(id: 1, data: VaultItemData.folder('Work')),
      VaultItem(
        id: 2,
        data: VaultItemData(
          kind: VaultItemKind.entry,
          name: 'GitHub, Inc',
          password: 'say "hi"',
          notes: 'two\nlines',
          folderId: 1,
        ),
      ),
    ], VaultExportFormat.csv);

    expect(
      csv,
      'name,url,username,password,notes,totp_secret,folder\n'
      '"GitHub, Inc",,,"say ""hi""","two\nlines",,Work',
    );
    final back = parseVaultImport(csv);
    expect(back.entries.single.name, 'GitHub, Inc');
    expect(back.entries.single.password, 'say "hi"');
  });
}
