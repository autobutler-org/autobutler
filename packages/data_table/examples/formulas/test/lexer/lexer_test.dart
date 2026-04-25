import 'package:data_table_example_formulas/evaluation/lexer/lexer.dart';
import 'package:data_table_example_formulas/evaluation/token.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('lex', () {
    test('tokenizes a representative formula and emits eof', () {
      final tokens =
          lex('=SUM(\$a\$1:B2, "he""llo", true, 1.25E+3) >= 5').toList();

      expect(
        tokens,
        equals([
          Token(kind: TokenKind.ident, value: 'SUM', offset: 0),
          Token(kind: TokenKind.lparen, offset: 3),
          Token(kind: TokenKind.cellRef, value: 'A1', offset: 4),
          Token(kind: TokenKind.colon, offset: 8),
          Token(kind: TokenKind.cellRef, value: 'B2', offset: 9),
          Token(kind: TokenKind.comma, offset: 11),
          Token(kind: TokenKind.string, value: 'he"llo', offset: 13),
          Token(kind: TokenKind.comma, offset: 22),
          Token(kind: TokenKind.boolean, value: 'TRUE', offset: 24),
          Token(kind: TokenKind.comma, offset: 28),
          Token(kind: TokenKind.number, value: '1.25E+3', offset: 30),
          Token(kind: TokenKind.rparen, offset: 37),
          Token(kind: TokenKind.gte, offset: 39),
          Token(kind: TokenKind.number, value: '5', offset: 42),
          Token(kind: TokenKind.eof, offset: 43),
        ]),
      );
    });

    test('skips whitespace and uppercases identifiers', () {
      final tokens = lex('  foo + false ').toList();

      expect(
        tokens,
        equals([
          Token(kind: TokenKind.ident, value: 'FOO', offset: 2),
          Token(kind: TokenKind.plus, offset: 6),
          Token(kind: TokenKind.boolean, value: 'FALSE', offset: 8),
          Token(kind: TokenKind.eof, offset: 14),
        ]),
      );
    });

    test('supports scientific notation and leading-dot decimals', () {
      final tokens = lex('.5 + 2e3').toList();

      expect(
        tokens,
        equals([
          Token(kind: TokenKind.number, value: '.5', offset: 0),
          Token(kind: TokenKind.plus, offset: 3),
          Token(kind: TokenKind.number, value: '2e3', offset: 5),
          Token(kind: TokenKind.eof, offset: 8),
        ]),
      );
    });

    test('rejects standalone equals', () {
      expect(
        () => lex('=').toList(),
        throwsA(
          isA<LexError>().having((error) => error.offset, 'offset', 0).having(
              (error) => error.message, 'message', contains('Standalone =')),
        ),
      );

      expect(
        () => lex('A1 = 2').toList(),
        throwsA(
          isA<LexError>().having((error) => error.offset, 'offset', 3).having(
              (error) => error.message, 'message', contains('Standalone =')),
        ),
      );
    });

    test('rejects invalid cell refs and unterminated strings', () {
      expect(
        () => lex(r'$foo').toList(),
        throwsA(
          isA<LexError>().having((error) => error.offset, 'offset', 0).having(
              (error) => error.message,
              'message',
              contains('Invalid cell reference')),
        ),
      );

      expect(
        () => lex('"abc').toList(),
        throwsA(
          isA<LexError>().having((error) => error.offset, 'offset', 0).having(
              (error) => error.message,
              'message',
              contains('Unterminated string')),
        ),
      );
    });
  });
}
