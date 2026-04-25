import 'package:data_table_example_formulas/evaluation/lexer/lexer.dart';
import 'package:data_table_example_formulas/evaluation/parser/parser.dart';
import 'package:data_table_example_formulas/evaluation/token.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('FormulaParser', () {
    test('builds AST metadata from a token stream', () {
      final parsed = FormulaParser(
        lex('=SUM(A1:B2, -C3^2, "x", FALSE) >= 10'),
      ).parse();

      expect(parsed.root, isA<BinaryNode>());

      final comparison = parsed.root as BinaryNode;
      expect(comparison.operatorKind, TokenKind.gte);
      expect(comparison.left, isA<CallNode>());
      expect(comparison.right, isA<NumberNode>());

      final call = comparison.left as CallNode;
      expect(call.functionName, 'SUM');
      expect(call.arguments, hasLength(4));
      expect(call.arguments[0], isA<RangeNode>());

      final unary = call.arguments[1] as UnaryNode;
      expect(unary.operatorKind, TokenKind.minus);
      expect(unary.operand, isA<BinaryNode>());

      final power = unary.operand as BinaryNode;
      expect(power.operatorKind, TokenKind.caret);
      expect((power.left as CellRefNode).ref, 'C3');
      expect((power.right as NumberNode).value, 2);

      expect(parsed.cellRefs, orderedEquals(['C3']));
      expect(parsed.rangeRefs, orderedEquals(['A1:B2']));
      expect(parsed.calledFunctions, orderedEquals(['SUM']));
      expect(parsed.hasRangeArgs, isTrue);
    });

    test('implements precedence with right-associative power', () {
      final parsed = parseTokens(lex('=1 + 2 * 3 ^ 4 == 163'));

      final comparison = parsed.root as BinaryNode;
      expect(comparison.operatorKind, TokenKind.eqEq);

      final addition = comparison.left as BinaryNode;
      expect(addition.operatorKind, TokenKind.plus);
      expect((addition.left as NumberNode).value, 1);

      final multiplication = addition.right as BinaryNode;
      expect(multiplication.operatorKind, TokenKind.star);
      expect((multiplication.left as NumberNode).value, 2);

      final power = multiplication.right as BinaryNode;
      expect(power.operatorKind, TokenKind.caret);
      expect((power.left as NumberNode).value, 3);
      expect((power.right as NumberNode).value, 4);
    });

    test('allows ranges only inside function argument context', () {
      expect(
        () => parseTokens(lex('=A1:B2')),
        throwsA(
          isA<LexError>().having((error) => error.offset, 'offset', 2).having(
                (error) => error.message,
                'message',
                contains('Range references are only allowed'),
              ),
        ),
      );
    });

    test('rejects identifiers that are not function calls', () {
      expect(
        () => parseTokens(lex('=foo + 1')),
        throwsA(
          isA<LexError>().having((error) => error.offset, 'offset', 0).having(
                (error) => error.message,
                'message',
                contains('Expected "(" after identifier FOO'),
              ),
        ),
      );
    });

    test('tracks nested function metadata in encounter order', () {
      final parsed = parseTokens(lex('=IF(TRUE, SUM(A1:B2), A1)'));

      expect(parsed.calledFunctions, orderedEquals(['IF', 'SUM']));
      expect(parsed.rangeRefs, orderedEquals(['A1:B2']));
      expect(parsed.cellRefs, orderedEquals(['A1']));
      expect(parsed.hasRangeArgs, isTrue);
    });
  });
}
