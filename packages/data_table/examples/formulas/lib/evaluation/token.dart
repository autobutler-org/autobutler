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

  /// Convert to JSON map. `kind` is serialized as its enum name.
  Map<String, dynamic> toJson() {
    return {
      'kind': kind.toString().split('.').last,
      'value': value,
    };
  }

  /// Create [Token] from JSON map. Unknown or missing kinds default to TokenKind.eof.
  factory Token.fromJson(Map<String, dynamic> json) {
    final kindStr = json['kind'] as String? ?? '';
    final kind = TokenKind.values.firstWhere(
      (k) => k.toString().split('.').last == kindStr,
      orElse: () => TokenKind.eof,
    );
    final value = json['value']?.toString() ?? '';
    return Token(kind: kind, value: value);
  }

  @override
  String toString() =>
      "Token(kind: ${kind.toString().split('.').last}, value: '$value')";

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
