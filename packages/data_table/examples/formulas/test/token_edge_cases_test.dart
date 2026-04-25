import 'package:flutter_test/flutter_test.dart';
import 'package:data_table_example_spreadsheet/evaluation/token.dart';

void main() {
  group('Token edge cases', () {
    test(
        'fromJson handles null kind by defaulting to eof, non-string kind throws',
        () {
      final jsonNullKind = {'kind': null, 'value': 'x'};
      final tNull = Token.fromJson(jsonNullKind);
      expect(tNull.kind, equals(TokenKind.eof));
      expect(tNull.value, equals('x'));

      final jsonNonStringKind = {'kind': 123, 'value': 'y'};
      // The implementation casts kind to String?; a non-string value will cause a TypeError.
      expect(
          () => Token.fromJson(jsonNonStringKind), throwsA(isA<TypeError>()));
    });

    test('fromJson treats unknown kind string as eof', () {
      final jsonUnknown = {'kind': 'UnknownKind', 'value': 'v'};
      final t = Token.fromJson(jsonUnknown);
      expect(t.kind, equals(TokenKind.eof));
      expect(t.value, equals('v'));
    });

    test('asNumber returns null for empty or invalid numeric strings', () {
      final empty = Token(kind: TokenKind.number, value: '');
      expect(empty.asNumber(), isNull);

      final invalid = Token(kind: TokenKind.number, value: 'not-a-number');
      expect(invalid.asNumber(), isNull);
    });

    test(
        'asNumber parses valid numeric formats (including negatives and exponents)',
        () {
      final a = Token(kind: TokenKind.number, value: '-42.5');
      expect(a.asNumber(), equals(-42.5));

      final b = Token(kind: TokenKind.number, value: '1E3');
      expect(b.asNumber(), equals(1000));

      final c = Token(kind: TokenKind.number, value: '6.022e23');
      final parsed = c.asNumber();
      expect(parsed, isNotNull);
      expect(parsed! > 1e23, isTrue);
    });

    test(
        'very long string values are preserved and participate in equality/hashCode',
        () {
      final longValue = List.filled(10000, 'x').join();
      final t1 = Token(kind: TokenKind.ident, value: longValue);
      final t2 = Token(kind: TokenKind.ident, value: longValue);

      expect(t1, equals(t2));
      expect(t1.hashCode, equals(t2.hashCode));

      final json = t1.toJson();
      expect((json['value'] as String).length, equals(longValue.length));
    });

    test('toJson/fromJson round-trip works for boundary values', () {
      final hugeNum = '1.2345e308';
      final t = Token(kind: TokenKind.number, value: hugeNum);
      final json = t.toJson();
      final t2 = Token.fromJson(json);
      expect(t2.kind, equals(TokenKind.number));
      expect(t2.value, equals(hugeNum));
      expect(t2.asNumber(), isNotNull);
    });
  });
}
