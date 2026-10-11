import 'package:flutter/foundation.dart' show immutable;

import 'code_language.dart';

// Keywords of the languages below, not prose.
// cspell:ignore associatedtype constexpr deinit elif endef fileprivate ifdef
// cspell:ignore ifeq ifndef ifneq isset lateinit nonlocal ONBUILD STOPSIGNAL
// cspell:ignore typealias varchar

/// What a [CodeToken] is, which decides the color it is drawn in.
enum CodeTokenKind {
  /// Uncolored text. [highlightCodeLine] leaves it out of its tokens.
  plain,

  /// A line or block comment.
  comment,

  /// A string literal.
  string,

  /// A numeric literal.
  number,

  /// A language keyword, a markup tag name, or a log level.
  keyword,

  /// A key: a JSON object key, or the name before the `:` or `=` in YAML,
  /// TOML and INI.
  key,
}

/// A colored run of one line: UTF-16 offsets `[start, end)` and what the run
/// is.
@immutable
class CodeToken {
  /// Creates a token covering `[start, end)` of its line.
  const CodeToken(this.start, this.end, this.kind);

  /// The UTF-16 offset in the line where the token starts.
  final int start;

  /// The UTF-16 offset in the line just past the token.
  final int end;

  /// What the token is.
  final CodeTokenKind kind;

  @override
  bool operator ==(Object other) =>
      other is CodeToken &&
      other.start == start &&
      other.end == end &&
      other.kind == kind;

  @override
  int get hashCode => Object.hash(start, end, kind);

  @override
  String toString() => 'CodeToken($start-$end, ${kind.name})';
}

/// The lexer state carried from the end of one line to the start of the next:
/// whether the next line opens inside a block comment or a multi-line string.
///
/// Opaque. Start a file with [initial] and pass each line the `endState` of
/// the line before it.
@immutable
class CodeLineState {
  const CodeLineState._(this._closer, this._kind);

  /// The state at the top of a file: inside nothing.
  static const CodeLineState initial = CodeLineState._(
    null,
    CodeTokenKind.plain,
  );

  /// The delimiter that ends the open comment or string; null inside nothing.
  final String? _closer;

  /// Whether the open construct is a comment or a string.
  final CodeTokenKind _kind;

  @override
  bool operator ==(Object other) =>
      other is CodeLineState &&
      other._closer == _closer &&
      other._kind == _kind;

  @override
  int get hashCode => Object.hash(_closer, _kind);
}

/// The longest line [highlightCodeLine] will lex. A longer one, a minified
/// bundle say, comes back with no tokens and renders plain.
const int maxHighlightedLineLength = 5000;

/// What the generic lexer needs to know about one language.
class _Spec {
  _Spec({
    String keywords = '',
    this.lineComments = const [],
    this.blockComment,
    this.strings = const ['"', "'"],
    this.multiLineStrings = const [],
    this.ignoreCase = false,
    this.prefixedQuotes = false,
    this.keyStrings = false,
    this.tags = false,
    this.keyPattern,
  }) : keywords = keywords.isEmpty ? const {} : keywords.split(' ').toSet();

  final Set<String> keywords;
  final List<String> lineComments;
  final (String, String)? blockComment;
  final List<String> strings;
  final List<String> multiLineStrings;

  /// Keywords match in any case (SQL). [keywords] are then lowercase.
  final bool ignoreCase;

  /// A `'` straight after a letter still opens a string (`r'raw'`, `b'x'`).
  final bool prefixedQuotes;

  /// A string followed by `:` is a key (JSON).
  final bool keyStrings;

  /// A name after `<` is a tag (XML, HTML).
  final bool tags;

  /// Matches the key at the start of a line (YAML, TOML, INI).
  final RegExp? keyPattern;
}

const (String, String) _slashStar = ('/*', '*/');
const List<String> _slashes = ['//'];
const List<String> _hash = ['#'];
const List<String> _tripleQuotes = ['"""', "'''"];
final RegExp _colonKey = RegExp(r'[\w.\-/$]+(?=\s*:(?:\s|$))');
final RegExp _equalsKey = RegExp(r'[\w.\-]+(?=\s*=)');

const String _javascriptKeywords =
    'async await break case catch class const continue debugger default '
    'delete do else export extends false finally for function get if import '
    'in instanceof let new null of return set static super switch this throw '
    'true try typeof undefined var void while with yield';

const String _javaKeywords =
    'abstract assert boolean break byte case catch char class const continue '
    'default do double else enum extends false final finally float for if '
    'implements import instanceof int interface long native new null package '
    'private protected public record return short static super switch '
    'synchronized this throw throws transient true try var void volatile while';

final _Spec _markup = _Spec(blockComment: ('<!--', '-->'), tags: true);

final Map<CodeLanguage, _Spec> _specs = {
  CodeLanguage.json: _Spec(
    keywords: 'true false null',
    lineComments: _slashes,
    blockComment: _slashStar,
    strings: const ['"'],
    keyStrings: true,
  ),
  CodeLanguage.yaml: _Spec(
    keywords: 'true false null',
    lineComments: _hash,
    keyPattern: _colonKey,
  ),
  CodeLanguage.toml: _Spec(
    keywords: 'true false',
    lineComments: _hash,
    multiLineStrings: _tripleQuotes,
    keyPattern: _equalsKey,
  ),
  CodeLanguage.ini: _Spec(
    keywords: 'true false',
    lineComments: const ['#', ';'],
    keyPattern: _equalsKey,
  ),
  CodeLanguage.xml: _markup,
  CodeLanguage.html: _markup,
  CodeLanguage.css: _Spec(blockComment: _slashStar),
  CodeLanguage.javascript: _Spec(
    keywords: _javascriptKeywords,
    lineComments: _slashes,
    blockComment: _slashStar,
    multiLineStrings: const ['`'],
  ),
  CodeLanguage.typescript: _Spec(
    keywords:
        '$_javascriptKeywords abstract any as boolean declare enum implements '
        'interface is keyof namespace never number private protected public '
        'readonly string type unknown',
    lineComments: _slashes,
    blockComment: _slashStar,
    multiLineStrings: const ['`'],
  ),
  CodeLanguage.go: _Spec(
    keywords:
        'break case chan const continue default defer else fallthrough false '
        'for func go goto if import interface iota map nil package range '
        'return select struct switch true type var',
    lineComments: _slashes,
    blockComment: _slashStar,
    multiLineStrings: const ['`'],
  ),
  CodeLanguage.c: _Spec(
    keywords:
        'auto bool break case catch char class const constexpr continue '
        'default delete do double else enum extern false final float for '
        'friend goto if inline int long namespace new NULL nullptr operator '
        'override private protected public register return short signed sizeof '
        'static struct switch template this throw true try typedef typename '
        'union unsigned using virtual void volatile while',
    lineComments: _slashes,
    blockComment: _slashStar,
  ),
  CodeLanguage.csharp: _Spec(
    keywords:
        'abstract as async await base bool break byte case catch char class '
        'const continue decimal default delegate do double else enum event '
        'explicit extern false finally fixed float for foreach if implicit in '
        'int interface internal is lock long namespace new null object '
        'operator out override params private protected public readonly ref '
        'return sealed short sizeof static string struct switch this throw '
        'true try typeof uint ulong using var virtual void volatile while',
    lineComments: _slashes,
    blockComment: _slashStar,
    multiLineStrings: const ['"""'],
  ),
  CodeLanguage.rust: _Spec(
    keywords:
        'as async await break const continue crate dyn else enum extern false '
        'fn for if impl in let loop match mod move mut pub ref return self '
        'Self static struct super trait true type unsafe use where while',
    lineComments: _slashes,
    blockComment: _slashStar,
    // No `'`: a lifetime (`'a`) is far more common than a char literal.
    strings: const ['"'],
  ),
  CodeLanguage.java: _Spec(
    keywords: _javaKeywords,
    lineComments: _slashes,
    blockComment: _slashStar,
    multiLineStrings: const ['"""'],
  ),
  CodeLanguage.kotlin: _Spec(
    keywords:
        'abstract as break by catch class companion constructor continue data '
        'do else enum false finally for fun if import in init interface '
        'internal is lateinit null object open override package private '
        'protected public return sealed super suspend this throw true try '
        'typealias val var when while',
    lineComments: _slashes,
    blockComment: _slashStar,
    multiLineStrings: const ['"""'],
  ),
  CodeLanguage.python: _Spec(
    keywords:
        'False None True and as assert async await break case class continue '
        'def del elif else except finally for from global if import in is '
        'lambda match nonlocal not or pass raise return self try while with '
        'yield',
    lineComments: _hash,
    multiLineStrings: _tripleQuotes,
    prefixedQuotes: true,
  ),
  CodeLanguage.ruby: _Spec(
    keywords:
        'alias and begin break case class def do else elsif end ensure false '
        'for if in module next nil not or redo require rescue retry return '
        'self super then true undef unless until when while yield',
    lineComments: _hash,
  ),
  CodeLanguage.php: _Spec(
    keywords:
        'abstract and array as break case catch class clone const continue '
        'declare default do echo else elseif empty extends false final finally '
        'fn for foreach function global if implements include instanceof '
        'interface isset namespace new null or print private protected public '
        'require return static switch throw trait true try unset use var while '
        'yield',
    lineComments: const ['//', '#'],
    blockComment: _slashStar,
  ),
  CodeLanguage.lua: _Spec(
    keywords:
        'and break do else elseif end false for function goto if in local nil '
        'not or repeat return then true until while',
    lineComments: const ['--'],
    blockComment: ('--[[', ']]'),
  ),
  CodeLanguage.shell: _Spec(
    keywords:
        'alias case declare do done elif else esac exit export fi for function '
        'if in local readonly return set source then unset until while',
    lineComments: _hash,
  ),
  CodeLanguage.swift: _Spec(
    keywords:
        'actor as associatedtype async await break case catch class continue '
        'default defer deinit do else enum extension fallthrough false '
        'fileprivate for func guard if import in init inout internal is let '
        'nil open operator private protocol public repeat return self Self '
        'static struct subscript super switch throw throws true try typealias '
        'var where while',
    lineComments: _slashes,
    blockComment: _slashStar,
    strings: const ['"'],
    multiLineStrings: const ['"""'],
  ),
  CodeLanguage.dart: _Spec(
    keywords:
        'abstract as assert async await base break case catch class const '
        'continue covariant default deferred do dynamic else enum export '
        'extends extension external factory false final finally for get hide '
        'if implements import in interface is late library mixin new null on '
        'operator part required rethrow return sealed set show static super '
        'switch sync this throw true try typedef var void when while with '
        'yield',
    lineComments: _slashes,
    blockComment: _slashStar,
    multiLineStrings: _tripleQuotes,
    prefixedQuotes: true,
  ),
  CodeLanguage.sql: _Spec(
    keywords:
        'all alter and as begin between blob boolean by case check commit '
        'constraint create default delete distinct drop else end exists '
        'foreign from full group having if in index inner insert integer into '
        'is join key left like limit not null offset on or order outer '
        'primary real references right rollback select set table text then '
        'timestamp transaction union unique update values varchar view when '
        'where',
    lineComments: const ['--'],
    blockComment: _slashStar,
    ignoreCase: true,
  ),
  CodeLanguage.dockerfile: _Spec(
    keywords:
        'ADD ARG AS CMD COPY ENTRYPOINT ENV EXPOSE FROM HEALTHCHECK LABEL '
        'ONBUILD RUN SHELL STOPSIGNAL USER VOLUME WORKDIR',
    lineComments: _hash,
  ),
  CodeLanguage.makefile: _Spec(
    keywords:
        'define else endef endif export ifdef ifeq ifndef ifneq include '
        'override unexport',
    lineComments: _hash,
  ),
  CodeLanguage.log: _Spec(
    keywords:
        'CRITICAL DEBUG ERR ERROR FATAL INFO TRACE WARN WARNING critical debug '
        'error fatal info trace warn warning',
    strings: const ['"'],
  ),
};

bool _isSpace(int unit) => unit == 0x20 || unit == 0x09;

bool _isDigit(int unit) => unit >= 0x30 && unit <= 0x39;

/// Letters, digits, `_`, `$`, and everything outside ASCII.
bool _isWord(int unit) =>
    _isDigit(unit) ||
    (unit >= 0x41 && unit <= 0x5A) ||
    (unit >= 0x61 && unit <= 0x7A) ||
    unit == 0x5F ||
    unit == 0x24 ||
    unit > 0x7F;

/// The offset just past the first [closer] at or after [from] in [line], or
/// -1 when the line ends first. With [escapes], a backslash hides the
/// character after it.
int _closeIndex(String line, int from, String closer, {required bool escapes}) {
  final first = closer.codeUnitAt(0);
  for (var i = from; i < line.length; i++) {
    final unit = line.codeUnitAt(i);
    if (escapes && unit == 0x5C) {
      i++;
    } else if (unit == first && line.startsWith(closer, i)) {
      return i + closer.length;
    }
  }
  return -1;
}

/// Colors one [line] of [language], given the [startState] the line before it
/// ended in (`CodeLineState.initial` for the first line).
///
/// Returns the colored runs, in order, never overlapping and always inside the
/// line, and the state to hand the next line. Text between tokens is plain.
///
/// A small generic lexer driven by a per-language table of keywords, comment
/// markers and string delimiters. It aims to be useful, not exact: it knows
/// nothing of a language's grammar, never throws, and always terminates. A
/// line longer than [maxHighlightedLineLength] is not lexed at all.
({List<CodeToken> tokens, CodeLineState endState}) highlightCodeLine(
  String line,
  CodeLanguage language,
  CodeLineState startState,
) {
  final length = line.length;
  if (length > maxHighlightedLineLength) {
    return (tokens: const [], endState: startState);
  }
  final spec = _specs[language];
  if (spec == null) {
    return (tokens: const [], endState: CodeLineState.initial);
  }

  final tokens = <CodeToken>[];
  void add(int start, int end, CodeTokenKind kind) {
    if (end > start) tokens.add(CodeToken(start, end, kind));
  }

  var i = 0;
  final openCloser = startState._closer;
  if (openCloser != null) {
    // The line opens inside the comment or string the last one left open.
    final end = _closeIndex(
      line,
      0,
      openCloser,
      escapes: startState._kind == CodeTokenKind.string,
    );
    if (end < 0) {
      add(0, length, startState._kind);
      return (tokens: tokens, endState: startState);
    }
    add(0, end, startState._kind);
    i = end;
  } else if (spec.keyPattern case final keyPattern?) {
    var lead = 0;
    while (lead < length &&
        (_isSpace(line.codeUnitAt(lead)) || line.codeUnitAt(lead) == 0x2D)) {
      lead++;
    }
    final match = keyPattern.matchAsPrefix(line, lead);
    if (match != null) {
      add(match.start, match.end, CodeTokenKind.key);
      i = match.end;
    }
  }

  var endState = CodeLineState.initial;
  // Every pass through this loop either leaves it or moves `i` forward.
  scan:
  while (i < length) {
    final unit = line.codeUnitAt(i);
    if (_isSpace(unit)) {
      i++;
      continue;
    }

    final block = spec.blockComment;
    if (block != null && line.startsWith(block.$1, i)) {
      final end = _closeIndex(
        line,
        i + block.$1.length,
        block.$2,
        escapes: false,
      );
      if (end < 0) {
        add(i, length, CodeTokenKind.comment);
        endState = CodeLineState._(block.$2, CodeTokenKind.comment);
        break;
      }
      add(i, end, CodeTokenKind.comment);
      i = end;
      continue;
    }

    for (final prefix in spec.lineComments) {
      // A one-character marker only counts at the start of a word, so the
      // `#` in `$#` or `a#b` and the `;` in `a;b` stay code.
      if (line.startsWith(prefix, i) &&
          (prefix.length > 1 || i == 0 || _isSpace(line.codeUnitAt(i - 1)))) {
        add(i, length, CodeTokenKind.comment);
        break scan;
      }
    }

    for (final delimiter in spec.multiLineStrings) {
      if (!line.startsWith(delimiter, i)) continue;
      final end = _closeIndex(
        line,
        i + delimiter.length,
        delimiter,
        escapes: true,
      );
      if (end < 0) {
        add(i, length, CodeTokenKind.string);
        endState = CodeLineState._(delimiter, CodeTokenKind.string);
        break scan;
      }
      add(i, end, CodeTokenKind.string);
      i = end;
      continue scan;
    }

    for (final delimiter in spec.strings) {
      if (!line.startsWith(delimiter, i)) continue;
      // An apostrophe inside a word (don't) is not a string.
      if (delimiter == "'" &&
          !spec.prefixedQuotes &&
          i > 0 &&
          _isWord(line.codeUnitAt(i - 1))) {
        continue;
      }
      final close = _closeIndex(line, i + 1, delimiter, escapes: true);
      // An unterminated string runs to the end of its line and no further.
      final end = close < 0 ? length : close;
      var kind = CodeTokenKind.string;
      if (spec.keyStrings) {
        var next = end;
        while (next < length && _isSpace(line.codeUnitAt(next))) {
          next++;
        }
        if (next < length && line.codeUnitAt(next) == 0x3A) {
          kind = CodeTokenKind.key;
        }
      }
      add(i, end, kind);
      i = end;
      continue scan;
    }

    if (spec.tags && unit == 0x3C) {
      // `<name`, `</name`, `<?name`, `<!name`.
      var start = i + 1;
      if (start < length && '/?!'.contains(line[start])) start++;
      var end = start;
      while (end < length) {
        final next = line.codeUnitAt(end);
        if (!_isWord(next) && next != 0x3A && next != 0x2D && next != 0x2E) {
          break;
        }
        end++;
      }
      add(start, end, CodeTokenKind.keyword);
      i = end > i ? end : i + 1;
      continue;
    }

    if (_isDigit(unit)) {
      var end = i + 1;
      while (end < length &&
          (_isWord(line.codeUnitAt(end)) || line.codeUnitAt(end) == 0x2E)) {
        end++;
      }
      add(i, end, CodeTokenKind.number);
      i = end;
      continue;
    }

    if (_isWord(unit)) {
      var end = i + 1;
      while (end < length && _isWord(line.codeUnitAt(end))) {
        end++;
      }
      final word = line.substring(i, end);
      if (spec.keywords.contains(spec.ignoreCase ? word.toLowerCase() : word)) {
        add(i, end, CodeTokenKind.keyword);
      }
      i = end;
      continue;
    }

    i++;
  }

  return (tokens: tokens, endState: endState);
}
