## Overview

Complete repository-level deliverables: EBNF grammar README, public API barrel, and comprehensive unit tests for lexer, parser, evaluator, built-ins, interpreter, and error cases.

## Acceptance Criteria

- Add lib/src/formula/README.md documenting the EBNF grammar (from task) and operator precedence.
- Expose public API via formula.dart: parse(String) → ParsedFormula; evaluate(ParsedFormula, CellAccessor, {originRow,originCol}) → FormulaValue; all value/error types exported.
- Unit tests covering: lexer token streams, parser AST shapes & metadata, evaluator arithmetic/logic/string semantics, each built-in function, division-by-zero, unknown function, type-mismatch, circular reference handling, and DataSheetInterpreter end-to-end scenarios.
- Tests run via existing project test runner; CI-friendly, no Flutter widget dependencies.

## Out of Scope

- Integration into DataSheetController or UI wiring (follow-up ticket).
- Performance microbenchmarks or isolate-based parallel parsing.

## Notes

- Tests should avoid relying on a running sheet controller; use pure-dart CellAccessor mocks.
- README must include example formulas and notes about case-insensitivity and '$' stripping.