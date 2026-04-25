import 'package:ab_formula/evaluation/evaluation.dart';
import 'package:test/test.dart';

void main() {
  group('builtinFunctions', () {
    FormulaValue call(String name, List<ResolvedArgument> args) {
      final builtin = builtinFunctions[name];
      expect(builtin, isNotNull, reason: '$name should be registered');
      return builtin!(args);
    }

    ScalarArgument scalar(FormulaValue value) => ScalarArgument(value);
    RangeArgument range(List<FormulaValue> values) => RangeArgument(values);

    test('SUM', () {
      expect(
        call('SUM', [
          scalar(const NumberValue(2)),
          range([const NumberValue(3), blankValue, const NumberValue(5)]),
        ]),
        const NumberValue(10),
      );
    });

    test('AVERAGE', () {
      expect(
        call('AVERAGE', [
          range([const NumberValue(2), const NumberValue(4)])
        ]),
        const NumberValue(3),
      );
    });

    test('MIN', () {
      expect(
        call('MIN', [
          range([const NumberValue(2), const NumberValue(-1)])
        ]),
        const NumberValue(-1),
      );
    });

    test('MAX', () {
      expect(
        call('MAX', [
          range([const NumberValue(2), const NumberValue(9)])
        ]),
        const NumberValue(9),
      );
    });

    test('ABS', () {
      expect(
          call('ABS', [scalar(const NumberValue(-3))]), const NumberValue(3));
    });

    test('ROUND', () {
      expect(
        call('ROUND',
            [scalar(const NumberValue(3.14159)), scalar(const NumberValue(2))]),
        const NumberValue(3.14),
      );
    });

    test('FLOOR', () {
      expect(
        call('FLOOR', [scalar(const NumberValue(3.9))]),
        const NumberValue(3),
      );
    });

    test('CEILING', () {
      expect(
        call('CEILING', [scalar(const NumberValue(3.1))]),
        const NumberValue(4),
      );
    });

    test('MOD', () {
      expect(
        call('MOD',
            [scalar(const NumberValue(10)), scalar(const NumberValue(4))]),
        const NumberValue(2),
      );
    });

    test('POWER', () {
      expect(
        call('POWER',
            [scalar(const NumberValue(2)), scalar(const NumberValue(5))]),
        const NumberValue(32),
      );
    });

    test('SQRT', () {
      expect(
        call('SQRT', [scalar(const NumberValue(81))]),
        const NumberValue(9),
      );
    });

    test('CONCAT', () {
      expect(
        call('CONCAT', [
          scalar(const StringValue('a')),
          scalar(const NumberValue(2)),
          scalar(const BoolValue(true)),
        ]),
        const StringValue('a2.0TRUE'),
      );
    });

    test('LEN', () {
      expect(call('LEN', [scalar(const StringValue('hello'))]),
          const NumberValue(5));
    });

    test('UPPER', () {
      expect(
        call('UPPER', [scalar(const StringValue('hello'))]),
        const StringValue('HELLO'),
      );
    });

    test('LOWER', () {
      expect(
        call('LOWER', [scalar(const StringValue('HELLO'))]),
        const StringValue('hello'),
      );
    });

    test('TRIM', () {
      expect(
        call('TRIM', [scalar(const StringValue('  hi   there  '))]),
        const StringValue('hi there'),
      );
    });

    test('LEFT', () {
      expect(
        call('LEFT',
            [scalar(const StringValue('hello')), scalar(const NumberValue(2))]),
        const StringValue('he'),
      );
    });

    test('RIGHT', () {
      expect(
        call('RIGHT',
            [scalar(const StringValue('hello')), scalar(const NumberValue(2))]),
        const StringValue('lo'),
      );
    });

    test('MID', () {
      expect(
        call('MID', [
          scalar(const StringValue('hello')),
          scalar(const NumberValue(2)),
          scalar(const NumberValue(3)),
        ]),
        const StringValue('ell'),
      );
    });

    test('FIND', () {
      expect(
        call('FIND', [
          scalar(const StringValue('th')),
          scalar(const StringValue('hi there')),
        ]),
        const NumberValue(4),
      );
    });

    test('SUBSTITUTE', () {
      expect(
        call('SUBSTITUTE', [
          scalar(const StringValue('foo foo')),
          scalar(const StringValue('foo')),
          scalar(const StringValue('bar')),
          scalar(const NumberValue(2)),
        ]),
        const StringValue('foo bar'),
      );
    });

    test('IF', () {
      expect(
        call('IF', [
          scalar(const BoolValue(true)),
          scalar(const StringValue('yes')),
          scalar(const StringValue('no')),
        ]),
        const StringValue('yes'),
      );
    });

    test('AND', () {
      expect(
        call('AND', [
          scalar(const BoolValue(true)),
          range([const BoolValue(true), const BoolValue(false)]),
        ]),
        const BoolValue(false),
      );
    });

    test('OR', () {
      expect(
        call('OR', [
          scalar(const BoolValue(false)),
          range([const BoolValue(false), const BoolValue(true)]),
        ]),
        const BoolValue(true),
      );
    });

    test('NOT', () {
      expect(
          call('NOT', [scalar(const BoolValue(false))]), const BoolValue(true));
    });

    test('IFERROR', () {
      expect(
        call('IFERROR', [
          scalar(nameError()),
          scalar(const StringValue('fallback')),
        ]),
        const StringValue('fallback'),
      );
    });

    test('ISBLANK', () {
      expect(call('ISBLANK', [scalar(blankValue)]), const BoolValue(true));
    });

    test('ISNUMBER', () {
      expect(call('ISNUMBER', [scalar(const NumberValue(1))]),
          const BoolValue(true));
    });

    test('ISTEXT', () {
      expect(call('ISTEXT', [scalar(const StringValue('x'))]),
          const BoolValue(true));
    });

    test('COUNT', () {
      expect(
        call('COUNT', [
          scalar(const NumberValue(1)),
          range([const NumberValue(2), const StringValue('x')]),
        ]),
        const NumberValue(2),
      );
    });

    test('COUNTA', () {
      expect(
        call('COUNTA', [
          range([blankValue, const NumberValue(2), const StringValue('x')]),
        ]),
        const NumberValue(2),
      );
    });

    test('COUNTIF', () {
      expect(
        call('COUNTIF', [
          range([
            const NumberValue(1),
            const NumberValue(3),
            const NumberValue(5),
          ]),
          scalar(const StringValue('>2')),
        ]),
        const NumberValue(2),
      );
    });
  });
}
