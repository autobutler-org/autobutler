import 'dart:math';

import 'package:flutter_test/flutter_test.dart';
import 'package:quark_widgets/quark_widgets.dart';

/// `highlightCodeLine` is a small generic lexer. These pin what it colors in
/// a few languages, the state it carries across lines, and that no input
/// makes it throw or hand back tokens a renderer could not use.
void main() {
  /// The text of each token of [line] with its kind, for readable asserts.
  List<(String, CodeTokenKind)> lex(
    String line,
    CodeLanguage language, [
    CodeLineState state = CodeLineState.initial,
  ]) => [
    for (final token in highlightCodeLine(line, language, state).tokens)
      (line.substring(token.start, token.end), token.kind),
  ];

  test('colors keywords, strings, numbers and comments in Dart', () {
    expect(lex("final x = 42; var s = 'hi'; // done", CodeLanguage.dart), [
      ('final', CodeTokenKind.keyword),
      ('42', CodeTokenKind.number),
      ('var', CodeTokenKind.keyword),
      ("'hi'", CodeTokenKind.string),
      ('// done', CodeTokenKind.comment),
    ]);
  });

  test('colors Python, with # comments and no keyword inside a word', () {
    expect(lex('def format(x): return 0x1F  # if', CodeLanguage.python), [
      ('def', CodeTokenKind.keyword),
      ('return', CodeTokenKind.keyword),
      ('0x1F', CodeTokenKind.number),
      ('# if', CodeTokenKind.comment),
    ]);
    expect(lex('informal = classify', CodeLanguage.python), isEmpty);
  });

  test('colors JSON keys apart from string values', () {
    expect(
      lex('  "name": "quark", "port": 8080, "on": true', CodeLanguage.json),
      [
        ('"name"', CodeTokenKind.key),
        ('"quark"', CodeTokenKind.string),
        ('"port"', CodeTokenKind.key),
        ('8080', CodeTokenKind.number),
        ('"on"', CodeTokenKind.key),
        ('true', CodeTokenKind.keyword),
      ],
    );
  });

  test('colors YAML and INI keys', () {
    expect(lex('  - image: nginx # web', CodeLanguage.yaml), [
      ('image', CodeTokenKind.key),
      ('# web', CodeTokenKind.comment),
    ]);
    expect(lex('port = 8080 ; default', CodeLanguage.ini), [
      ('port', CodeTokenKind.key),
      ('8080', CodeTokenKind.number),
      ('; default', CodeTokenKind.comment),
    ]);
  });

  test('colors SQL keywords in any case, and log levels', () {
    expect(lex("Select id FROM t -- all", CodeLanguage.sql), [
      ('Select', CodeTokenKind.keyword),
      ('FROM', CodeTokenKind.keyword),
      ('-- all', CodeTokenKind.comment),
    ]);
    expect(lex('ERROR disk full', CodeLanguage.log), [
      ('ERROR', CodeTokenKind.keyword),
    ]);
  });

  test('colors markup tags, attribute values and comments', () {
    expect(lex('<a href="/x">hi</a><!-- c -->', CodeLanguage.html), [
      ('a', CodeTokenKind.keyword),
      ('"/x"', CodeTokenKind.string),
      ('a', CodeTokenKind.keyword),
      ('<!-- c -->', CodeTokenKind.comment),
    ]);
  });

  test('an apostrophe inside a word does not open a string', () {
    expect(lex("echo don't stop", CodeLanguage.shell), isEmpty);
    expect(lex("x = r'raw'", CodeLanguage.python), [
      ("'raw'", CodeTokenKind.string),
    ]);
  });

  test('a # only starts a comment at the start of a word', () {
    expect(lex(r'echo $# a#b', CodeLanguage.shell), isEmpty);
  });

  test('plain text has no tokens', () {
    final result = highlightCodeLine(
      'if "x" 42 // y',
      CodeLanguage.plain,
      CodeLineState.initial,
    );
    expect(result.tokens, isEmpty);
    expect(result.endState, CodeLineState.initial);
  });

  test('carries a block comment across lines', () {
    final first = highlightCodeLine(
      'int a; /* begins',
      CodeLanguage.c,
      CodeLineState.initial,
    );
    expect(first.tokens.last, const CodeToken(7, 16, CodeTokenKind.comment));
    expect(first.endState, isNot(CodeLineState.initial));

    final middle = highlightCodeLine(
      'return 1; // still comment',
      CodeLanguage.c,
      first.endState,
    );
    expect(middle.tokens, const [CodeToken(0, 26, CodeTokenKind.comment)]);
    expect(middle.endState, first.endState);

    expect(lex('ends */ return 2;', CodeLanguage.c, middle.endState), [
      ('ends */', CodeTokenKind.comment),
      ('return', CodeTokenKind.keyword),
      ('2', CodeTokenKind.number),
    ]);
    expect(
      highlightCodeLine('ends */ x', CodeLanguage.c, middle.endState).endState,
      CodeLineState.initial,
    );
  });

  test('carries a triple-quoted string and a backtick string', () {
    final doc = highlightCodeLine(
      'x = """first',
      CodeLanguage.python,
      CodeLineState.initial,
    );
    expect(lex('second""" + 1', CodeLanguage.python, doc.endState), [
      ('second"""', CodeTokenKind.string),
      ('1', CodeTokenKind.number),
    ]);

    final template = highlightCodeLine(
      'const s = `a',
      CodeLanguage.javascript,
      CodeLineState.initial,
    );
    expect(lex(r'b \` c` ; let', CodeLanguage.javascript, template.endState), [
      (r'b \` c`', CodeTokenKind.string),
      ('let', CodeTokenKind.keyword),
    ]);
  });

  test('an unterminated string ends with its line', () {
    final result = highlightCodeLine(
      'var s = "oops',
      CodeLanguage.dart,
      CodeLineState.initial,
    );
    expect(result.tokens.last, const CodeToken(8, 13, CodeTokenKind.string));
    expect(result.endState, CodeLineState.initial);
    expect(lex(r'"a \" b" 1', CodeLanguage.dart), [
      (r'"a \" b"', CodeTokenKind.string),
      ('1', CodeTokenKind.number),
    ]);
  });

  test('a line past the length limit is not lexed and keeps the state', () {
    final open = highlightCodeLine(
      '/*',
      CodeLanguage.c,
      CodeLineState.initial,
    ).endState;
    final result = highlightCodeLine(
      'x' * (maxHighlightedLineLength + 1),
      CodeLanguage.c,
      open,
    );
    expect(result.tokens, isEmpty);
    expect(result.endState, open);
  });

  test('odd input never throws, and tokens are ordered and in range', () {
    const alphabet = [
      '"',
      "'",
      '`',
      '/',
      '*',
      '#',
      '-',
      '[',
      ']',
      '<',
      '>',
      '!',
      r'\',
      ':',
      '=',
      ' ',
      '\t',
      'a',
      'if',
      '0',
      '.',
      '"""',
      "'''",
      'é',
      '中',
      '😀',
      '\ud83d',
      '\u0000',
      ';',
      r'$',
    ];
    final random = Random(2991);
    for (final language in CodeLanguage.values) {
      var state = CodeLineState.initial;
      for (var round = 0; round < 200; round++) {
        final line = [
          for (var i = random.nextInt(24); i > 0; i--)
            alphabet[random.nextInt(alphabet.length)],
        ].join();
        final result = highlightCodeLine(line, language, state);
        var previousEnd = 0;
        for (final token in result.tokens) {
          expect(token.start, greaterThanOrEqualTo(previousEnd), reason: line);
          expect(token.end, greaterThan(token.start), reason: line);
          expect(token.end, lessThanOrEqualTo(line.length), reason: line);
          expect(token.kind, isNot(CodeTokenKind.plain), reason: line);
          previousEnd = token.end;
        }
        state = result.endState;
      }
    }
  });
}
