// Interpreter that ties lexer, parser and evaluator together

import 'lexer.dart';
import 'parser.dart';
import 'evaluator.dart';

class Interpreter {
  final String expr;
  Interpreter(this.expr);

  dynamic run() {
    final lexer = Lexer(expr);
    final tokens = lexer.tokenize();
    final ast = Parser(tokens).parse();
    return Evaluator().evaluate(ast);
  }
}
