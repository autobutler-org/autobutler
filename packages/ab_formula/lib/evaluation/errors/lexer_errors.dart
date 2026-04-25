class LexError implements Exception {
  final String message;
  final int offset;

  LexError(this.message, {required this.offset});

  @override
  String toString() => 'LexError(offset: $offset, message: $message)';
}
