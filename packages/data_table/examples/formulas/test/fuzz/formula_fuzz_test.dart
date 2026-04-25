import 'dart:math';

import 'package:data_table_example_formulas/evaluation/lexer/lexer.dart';
import 'package:data_table_example_formulas/evaluation/parser/parser.dart';
import 'package:data_table_example_formulas/evaluation/token.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  group('formula fuzz', () {
    test('lexer tolerates deterministic random ascii inputs', () {
      final random = Random(1337);

      for (var index = 0; index < 250; index++) {
        final candidate = _randomAsciiFormula(random, maxLength: 36);

        try {
          final tokens = lex(candidate).toList();
          expect(tokens, isNotEmpty, reason: 'candidate: $candidate');
          expect(tokens.last.kind, TokenKind.eof,
              reason: 'candidate: $candidate');
          expect(
            tokens.map((token) => token.offset),
            orderedEquals([...tokens.map((token) => token.offset)]..sort()),
            reason: 'candidate: $candidate',
          );
        } on LexError {
          // Expected for invalid random inputs.
        }
      }
    });

    test('parser accepts deterministic random valid formulas', () {
      final random = Random(424242);

      for (var index = 0; index < 200; index++) {
        final formula = '=${_generateExpression(random, 0, false)}';
        final parsed = parseTokens(lex(formula));

        expect(parsed.root, isA<FormulaNode>(), reason: 'formula: $formula');
        expect(
          parsed.calledFunctions.every((name) => name == name.toUpperCase()),
          isTrue,
          reason: 'formula: $formula',
        );
        expect(
          parsed.cellRefs
              .every((ref) => RegExp(r'^[A-Z]+[1-9][0-9]*$').hasMatch(ref)),
          isTrue,
          reason: 'formula: $formula',
        );
        expect(
          parsed.rangeRefs.every(
            (ref) =>
                RegExp(r'^[A-Z]+[1-9][0-9]*:[A-Z]+[1-9][0-9]*$').hasMatch(ref),
          ),
          isTrue,
          reason: 'formula: $formula',
        );
      }
    });

    test('parser random smoke test only yields parsed results or LexError', () {
      final random = Random(9001);

      for (var index = 0; index < 200; index++) {
        final candidate = _randomAsciiFormula(random, maxLength: 28);

        try {
          parseTokens(lex(candidate));
        } on LexError {
          // Expected for invalid inputs.
        }
      }
    });
  });
}

String _randomAsciiFormula(Random random, {required int maxLength}) {
  const alphabet =
      'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789()<>!=:+-*/%^,.:_\$ "';
  final length = random.nextInt(maxLength + 1);
  final buffer = StringBuffer();
  if (random.nextBool()) {
    buffer.write('=');
  }
  for (var index = 0; index < length; index++) {
    buffer.write(alphabet[random.nextInt(alphabet.length)]);
  }
  return buffer.toString();
}

String _generateExpression(
    Random random, int depth, bool insideFunctionArgument) {
  if (depth >= 3) {
    return _generateTerminal(random, insideFunctionArgument);
  }

  final choice = random.nextInt(8);
  switch (choice) {
    case 0:
      return _generateTerminal(random, insideFunctionArgument);
    case 1:
      final operator = random.nextBool() ? '+' : '-';
      return '$operator${_generateExpression(random, depth + 1, insideFunctionArgument)}';
    case 2:
      return '(${_generateExpression(random, depth + 1, insideFunctionArgument)})';
    case 3:
      return '${_generateExpression(random, depth + 1, insideFunctionArgument)} ${_pick(random, [
            '+',
            '-'
          ])} ${_generateExpression(random, depth + 1, insideFunctionArgument)}';
    case 4:
      return '${_generateExpression(random, depth + 1, insideFunctionArgument)} ${_pick(random, [
            '*',
            '/',
            '%'
          ])} ${_generateExpression(random, depth + 1, insideFunctionArgument)}';
    case 5:
      return '${_generateExpression(random, depth + 1, insideFunctionArgument)} ^ ${_generateExpression(random, depth + 1, insideFunctionArgument)}';
    case 6:
      return '${_generateExpression(random, depth + 1, insideFunctionArgument)} ${_pick(random, [
            '==',
            '!=',
            '<',
            '<=',
            '>',
            '>='
          ])} ${_generateExpression(random, depth + 1, insideFunctionArgument)}';
    default:
      return _generateFunctionCall(random, depth + 1);
  }
}

String _generateTerminal(Random random, bool insideFunctionArgument) {
  final choices = <String Function()>[
    () => _generateNumber(random),
    () => '"${_pick(random, ['a', 'b', 'c', 'he""llo', ' spaced '])}"',
    () => random.nextBool() ? 'TRUE' : 'FALSE',
    () => _generateCellRef(random),
  ];

  if (insideFunctionArgument) {
    choices
        .add(() => '${_generateCellRef(random)}:${_generateCellRef(random)}');
  }

  return choices[random.nextInt(choices.length)]();
}

String _generateFunctionCall(Random random, int depth) {
  final functionName = _pick(random, ['SUM', 'IF', 'MAX', 'MIN', 'AND', 'OR']);
  final argumentCount = 1 + random.nextInt(3);
  final arguments = List.generate(
    argumentCount,
    (_) => _generateExpression(random, depth, true),
  );
  return '$functionName(${arguments.join(', ')})';
}

String _generateNumber(Random random) {
  final whole = 1 + random.nextInt(999);
  if (!random.nextBool()) {
    return '$whole';
  }

  final fraction = 1 + random.nextInt(999);
  return '$whole.$fraction';
}

String _generateCellRef(Random random) {
  final columnLength = 1 + random.nextInt(2);
  final column = List.generate(
    columnLength,
    (_) => String.fromCharCode(65 + random.nextInt(26)),
  ).join();
  final row = 1 + random.nextInt(99);
  return '$column$row';
}

T _pick<T>(Random random, List<T> options) {
  return options[random.nextInt(options.length)];
}
