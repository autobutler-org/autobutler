## Overview

Create FormulaParser: a recursive-descent parser that consumes the lexer's token stream and emits a typed AST (FormulaNode) plus ParsedFormula metadata: cellRefs, rangeRefs, calledFunctions, hasRangeArgs.

## Acceptance Criteria

- Parser implements the full grammar precedence (comparison → addition → term → unary → power → primary).
- Constructs AST node types: NumberNode, StringNode, BoolNode, CellRefNode, RangeNode, UnaryNode, BinaryNode, CallNode.
- Validates range-ref usage: range-ref parsed only inside function argument context; bare range-ref outside calls is a parse error.
- Populates ParsedFormula fields (cellRefs uppercased A1 format, rangeRefs in A1:C3 format, calledFunctions uppercased, hasRangeArgs boolean).
- Emits FormulaParseError with offset/message on syntax errors.
- Unit tests asserting AST shapes, metadata, and error offsets.

## Out of Scope

- Evaluation semantics or built-in functions implementation.
- Dependency graph or sheet-level analysis.

## Notes

- Lexer yields tokens lazily; parser should iterate tokens without materialising a list.
- Treat '$' as stripped at lex time; parser receives normalized cellRef lexemes.
- Identifier without '(' (except TRUE/FALSE) should be an error.