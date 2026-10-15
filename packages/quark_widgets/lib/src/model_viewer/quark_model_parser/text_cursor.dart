import 'dart:math' as math;
import 'dart:typed_data';

import '../quark_model_parser.dart';

/// Reads the words and numbers of a text model file (ASCII STL, OBJ, a PLY
/// header or ASCII body) straight from its bytes, so a 60 MB file is never
/// decoded into a string twice its size.
class TextCursor {
  /// Creates a cursor at [position] in [bytes].
  TextCursor(this.bytes, [this.position = 0]);

  /// The file.
  final Uint8List bytes;

  /// The offset of the next byte to read.
  int position;

  static const int _space = 0x20, _tab = 0x09, _cr = 0x0d, _lf = 0x0a;

  /// Whether every byte has been read.
  bool get atEnd => position >= bytes.length;

  /// Whether the cursor sits at the end of a line, or of the file, after
  /// skipping spaces.
  bool get atLineEnd {
    skipSpaces();
    return atEnd || bytes[position] == _lf;
  }

  /// Skips spaces and tabs, and a carriage return, but not a line break.
  void skipSpaces() {
    while (position < bytes.length) {
      final byte = bytes[position];
      if (byte != _space && byte != _tab && byte != _cr) return;
      position++;
    }
  }

  /// Skips every kind of whitespace, line breaks included.
  void skipWhitespace() {
    while (position < bytes.length && _isWhitespace(bytes[position])) {
      position++;
    }
  }

  /// Moves past the next line break, or to the end of the file.
  void nextLine() {
    while (position < bytes.length && bytes[position] != _lf) {
      position++;
    }
    if (position < bytes.length) position++;
  }

  /// Whether the byte under the cursor is [byte].
  bool at(int byte) => position < bytes.length && bytes[position] == byte;

  /// Moves past the word under the cursor, after skipping spaces.
  void skipWord() {
    skipSpaces();
    skipRestOfWord();
  }

  /// Moves to the end of the word the cursor is in the middle of, or stays
  /// put on whitespace.
  void skipRestOfWord() {
    while (position < bytes.length && !_isWhitespace(bytes[position])) {
      position++;
    }
  }

  /// Whether the word under the cursor, after spaces, is exactly [word]
  /// (ASCII, case-sensitive). Moves past it when it is; stays put otherwise.
  bool word(String word) {
    skipSpaces();
    final end = position + word.length;
    if (end > bytes.length) return false;
    for (var i = 0; i < word.length; i++) {
      if (bytes[position + i] != word.codeUnitAt(i)) return false;
    }
    if (end < bytes.length && !_isWhitespace(bytes[end])) return false;
    position = end;
    return true;
  }

  /// The word under the cursor, after spaces, as a string. Only for short
  /// header words; the body is read with [word] and the number readers.
  String readWord() {
    skipSpaces();
    final start = position;
    skipWord();
    return String.fromCharCodes(bytes, start, position);
  }

  /// Reads a decimal number such as `-1.5e-3`, after spaces.
  double readDouble() {
    skipSpaces();
    final start = position;
    var negative = false;
    if (position < bytes.length &&
        (bytes[position] == 0x2d || bytes[position] == 0x2b)) {
      negative = bytes[position] == 0x2d;
      position++;
    }
    var mantissa = 0.0;
    var digits = 0;
    var exponent = 0;
    while (position < bytes.length && _isDigit(bytes[position])) {
      mantissa = mantissa * 10 + (bytes[position++] - 0x30);
      digits++;
    }
    if (position < bytes.length && bytes[position] == 0x2e) {
      position++;
      while (position < bytes.length && _isDigit(bytes[position])) {
        mantissa = mantissa * 10 + (bytes[position++] - 0x30);
        exponent--;
        digits++;
      }
    }
    if (digits == 0) {
      position = start;
      throw QuarkModelParseException(
        QuarkModelParseIssue.malformed,
        'no number at $start',
      );
    }
    if (position < bytes.length &&
        (bytes[position] == 0x65 || bytes[position] == 0x45)) {
      position++;
      var exponentNegative = false;
      if (position < bytes.length &&
          (bytes[position] == 0x2d || bytes[position] == 0x2b)) {
        exponentNegative = bytes[position] == 0x2d;
        position++;
      }
      var value = 0;
      while (position < bytes.length && _isDigit(bytes[position])) {
        value = math.min(value * 10 + (bytes[position++] - 0x30), 1000);
      }
      exponent += exponentNegative ? -value : value;
    }
    final magnitude = exponent == 0
        ? mantissa
        : exponent > 0
        ? mantissa * math.pow(10, exponent)
        : mantissa / math.pow(10, -exponent);
    return negative ? -magnitude : magnitude;
  }

  /// Reads a whole number, such as an OBJ index (which may be negative),
  /// after spaces. Stops at the first byte that is not a digit, so `7/2/3`
  /// reads 7 and leaves the cursor on the slash.
  int readInt() {
    skipSpaces();
    final start = position;
    var negative = false;
    if (position < bytes.length &&
        (bytes[position] == 0x2d || bytes[position] == 0x2b)) {
      negative = bytes[position] == 0x2d;
      position++;
    }
    var value = 0;
    final digitsStart = position;
    while (position < bytes.length && _isDigit(bytes[position])) {
      value = value * 10 + (bytes[position++] - 0x30);
      // Past any index a capped mesh could have; keeps it a small integer.
      if (value > 0x7fffffff) value = 0x7fffffff;
    }
    if (position == digitsStart) {
      position = start;
      throw QuarkModelParseException(
        QuarkModelParseIssue.malformed,
        'no integer at $start',
      );
    }
    return negative ? -value : value;
  }

  static bool _isDigit(int byte) => byte >= 0x30 && byte <= 0x39;

  static bool _isWhitespace(int byte) =>
      byte == _space || byte == _tab || byte == _cr || byte == _lf;
}
