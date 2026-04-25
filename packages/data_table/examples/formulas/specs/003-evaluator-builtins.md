## Overview

Implement FormulaEvaluator and the MVP built-in function registry. Evaluator walks the AST, produces FormulaValue tagged union results, and resolves cell references via a pure CellAccessor callback. Register built-ins in a Map<String, BuiltinFn> to allow extensibility.

## Acceptance Criteria

- FormulaValue types implemented: NumberValue, StringValue, BoolValue, ErrorValue.
- Evaluator supports arithmetic with correct precedence (PEMDAS), right-associative '^', unary +/-.
- Resolves CellRefNode by calling provided CellAccessor(row,col).
- Returns ErrorValue for division by zero (#DIV/0!), unknown function (#NAME?), type mismatch (#VALUE!), and circular reference detection hooks (#REF!).
- Built-in functions implemented for MVP categories: Math (SUM, AVERAGE, MIN, MAX, ABS, ROUND, FLOOR, CEILING, MOD, POWER, SQRT), String (CONCAT, LEN, UPPER, LOWER, TRIM, LEFT, RIGHT, MID, FIND, SUBSTITUTE), Logic (IF, AND, OR, NOT, IFERROR), Info (ISBLANK, ISNUMBER, ISTEXT), Count (COUNT, COUNTA, COUNTIF).
- Range arguments to aggregation/array-accepting functions handled (CellAccessor used per cell in range).
- Builtins registered in a Map and unit-tested individually.

## Out of Scope

- Sheet-level dependency ordering, cycle detection, or interpreter orchestration.
- Advanced Excel semantics beyond stated MVP set.

## Notes

- CellAccessor is typedef FormulaValue Function(int row,int col).
- Evaluator must be pure Dart with no Flutter/UI deps.
- ErrorValue should carry machine code and human message.