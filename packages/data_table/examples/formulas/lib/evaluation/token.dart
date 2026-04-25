enum TokenKind {
  number,
  string,
  boolean,
  ident,
  cellRef,
  colon,
  plus,
  minus,
  star,
  slash,
  percent,
  caret,
  eqEq,
  neq,
  lt,
  lte,
  gt,
  gte,
  lparen,
  rparen,
  comma,
  eof
}

class Token {
  final TokenKind kind;
  final String value;

  Token({required this.kind, this.value = ""});

  double? asNumber() {
    if (kind != TokenKind.number) {
      return null;
    }
    return double.tryParse(value);
  }
}
