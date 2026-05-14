import 'package:ab_formula/evaluation/token.dart';
import 'package:test/test.dart';

void main() {
  group('Token', () {
    test('construction and properties', () {
      final t = Token(kind: TokenKind.number, value: '3.14', offset: 2);

      expect(t.kind, TokenKind.number);
      expect(t.value, '3.14');
      expect(t.offset, 2);
      expect(t.asNumber(), 3.14);
    });

    test('asNumber returns null for non-number kinds', () {
      final t = Token(kind: TokenKind.ident, value: 'abc');
      expect(t.asNumber(), isNull);
    });

    test('equality and hashCode for equal tokens', () {
      final a = Token(kind: TokenKind.ident, value: 'foo');
      final b = Token(kind: TokenKind.ident, value: 'foo');

      expect(a, equals(b));
      expect(a.hashCode, equals(b.hashCode));
    });

    test('inequality for different tokens', () {
      final a = Token(kind: TokenKind.ident, value: 'foo');
      final b = Token(kind: TokenKind.ident, value: 'bar');
      final c = Token(kind: TokenKind.number, value: 'foo');

      expect(a == b, isFalse);
      expect(a == c, isFalse);
    });

    test('JSON round-trip preserves Token', () {
      final t = Token(kind: TokenKind.number, value: '123.45', offset: 7);
      final json = t.toJson();
      final t2 = Token.fromJson(json);
      expect(t2, equals(t));
    });

    test('toString is stable and readable', () {
      final t = Token(kind: TokenKind.ident, value: 'MyVar', offset: 4);
      expect(
        t.toString(),
        equals("Token(kind: ident, value: 'MyVar', offset: 4)"),
      );
    });

    test(
      'fromJson handles missing kind (defaults to eof) and missing value',
      () {
        final jsonMissingKind = {'value': 'x'};
        final t = Token.fromJson(jsonMissingKind);
        expect(t.kind, equals(TokenKind.eof));
        expect(t.value, equals('x'));
        expect(t.offset, equals(0));

        final jsonMissingValue = {'kind': 'string', 'offset': '9'};
        final t2 = Token.fromJson(jsonMissingValue);
        expect(t2.kind, equals(TokenKind.string));
        expect(t2.value, equals(''));
        expect(t2.offset, equals(9));
      },
    );

    test('fromJson ignores extra fields', () {
      final json = {
        'kind': 'boolean',
        'value': 'true',
        'offset': 3,
        'extra': 42,
      };
      final t = Token.fromJson(json);
      expect(t.kind, equals(TokenKind.boolean));
      expect(t.value, equals('true'));
      expect(t.offset, equals(3));
    });
  });
}
