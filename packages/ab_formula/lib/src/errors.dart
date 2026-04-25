// Formula error definitions

class FormulaError implements Exception {
  final String message;
  FormulaError(this.message);
  @override
  String toString() => 'FormulaError: $message';
}
