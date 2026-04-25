## Overview

Implement the standalone FormulaLexer as a Dart sync\* generator that tokenises formula expressions (leading '=' already removed). It yields Token objects (TokenKind + lexeme + offset) and must handle numbers, strings, booleans, identifiers, cell references, operators, punctuation, and always emit an eof token.

## Acceptance Criteria

- Lexer implemented as a sync\* generator matching the provided generator signature.
- Produces TokenKind variants: number, string, boolean, ident, cellRef, colon, plus, minus, star, slash, percent, caret, eqEq, neq, lt, lte, gt, gte, lparen, rparen, comma, eof.
- Skips whitespace and uppercases idents/cellRefs/booleans in lexeme.
- Strips '$' from cellRef lexemes and validates row/col format.
- Throws LexError on unrecognised characters or standalone '='.
- Unit tests exercising token streams, edge cases, escaped quotes, scientific notation.

## Out of Scope

- Any parser, evaluator, or integration logic. This spec only covers tokenisation.
- Multi-threading/isolate variants or performance micro-optimisations.

## Notes

- Keep numbers positive at lex time; unary '-' is a parser token.
- Use codeunit offset for error positions.
- Ensure generator always yields eof as last token.
