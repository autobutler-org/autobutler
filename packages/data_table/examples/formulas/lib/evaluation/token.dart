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

  @override
  bool operator ==(Object other) =>
      identical(this, other) ||
      other is Token &&
          runtimeType == other.runtimeType &&
          kind == other.kind &&
          value == other.value;

  @override
  int get hashCode => kind.hashCode ^ value.hashCode;
}
