import 'dart:typed_data';

import '../quark_model_parser.dart';

/// Decompresses raw DEFLATE (RFC 1951) [input], the way a zip stores an
/// entry, into at most [maxLength] bytes.
///
/// Output past [maxLength] throws a `tooLarge` [QuarkModelParseException],
/// so a zip bomb costs no more than the cap; [expectedLength], the size the
/// archive declares, only sizes the first buffer and is never trusted.
/// [tick] is called as symbols are decoded, so the caller's time cap holds.
/// Anything that is not valid DEFLATE is `malformed`.
Uint8List inflate(
  Uint8List input, {
  required int maxLength,
  required void Function() tick,
  int expectedLength = 0,
}) => _Inflater(input, maxLength, tick, expectedLength).run();

/// Code lengths are sent in this order in a dynamic block's header.
const List<int> _codeLengthOrder = [
  16, 17, 18, 0, 8, 7, 9, 6, 10, 5, 11, 4, 12, 3, 13, 2, 14, 1, 15, //
];

const List<int> _lengthBase = [
  3, 4, 5, 6, 7, 8, 9, 10, 11, 13, 15, 17, 19, 23, 27, 31, 35, 43, 51, 59, //
  67, 83, 99, 115, 131, 163, 195, 227, 258,
];

const List<int> _lengthExtra = [
  0, 0, 0, 0, 0, 0, 0, 0, 1, 1, 1, 1, 2, 2, 2, 2, 3, 3, 3, 3, 4, 4, 4, 4, //
  5, 5, 5, 5, 0,
];

const List<int> _distanceBase = [
  1, 2, 3, 4, 5, 7, 9, 13, 17, 25, 33, 49, 65, 97, 129, 193, 257, 385, 513, //
  769, 1025, 1537, 2049, 3073, 4097, 6145, 8193, 12289, 16385, 24577,
];

const List<int> _distanceExtra = [
  0, 0, 0, 0, 1, 1, 2, 2, 3, 3, 4, 4, 5, 5, 6, 6, 7, 7, 8, 8, 9, 9, 10, 10, //
  11, 11, 12, 12, 13, 13,
];

/// A canonical Huffman code as a lookup table indexed by the next
/// [maxBits] input bits (least significant first, as DEFLATE packs them).
/// Each entry is `symbol << 4 | length`; 0 marks bits that start no code.
class _Huffman {
  _Huffman(List<int> lengths) {
    final counts = Int32List(16);
    for (final length in lengths) {
      counts[length]++;
      if (length > maxBits) maxBits = length;
    }
    counts[0] = 0;
    final next = Int32List(16);
    var code = 0;
    for (var bits = 1; bits < 16; bits++) {
      code = (code + counts[bits - 1]) << 1;
      next[bits] = code;
    }
    table = Int32List(1 << maxBits);
    for (var symbol = 0; symbol < lengths.length; symbol++) {
      final length = lengths[symbol];
      if (length == 0) continue;
      final code = next[length]++;
      if (code >= 1 << length) _bad('oversubscribed code');
      var reversed = 0;
      for (var i = 0; i < length; i++) {
        reversed |= (code >> i & 1) << (length - 1 - i);
      }
      for (var i = reversed; i < table.length; i += 1 << length) {
        table[i] = symbol << 4 | length;
      }
    }
  }

  int maxBits = 0;
  late final Int32List table;
}

class _Inflater {
  _Inflater(this.input, this.maxLength, this.tick, int expectedLength)
    : _out = Uint8List(
        expectedLength.clamp(1024, maxLength.clamp(1024, 1 << 24)),
      );

  final Uint8List input;
  final int maxLength;
  final void Function() tick;

  Uint8List _out;
  int _length = 0;
  int _position = 0;
  int _bitBuffer = 0;
  int _bitCount = 0;

  static final _Huffman _fixedLiterals = _Huffman([
    for (var i = 0; i < 288; i++)
      i < 144 ? 8 : (i < 256 ? 9 : (i < 280 ? 7 : 8)),
  ]);
  static final _Huffman _fixedDistances = _Huffman(List.filled(30, 5));

  Uint8List run() {
    var last = false;
    while (!last) {
      last = _bits(1) == 1;
      switch (_bits(2)) {
        case 0:
          _stored();
        case 1:
          _codes(_fixedLiterals, _fixedDistances);
        case 2:
          _dynamic();
        default:
          _bad('block type 3');
      }
    }
    return Uint8List.sublistView(_out, 0, _length);
  }

  int _bits(int need) {
    while (_bitCount < need) {
      if (_position >= input.length) _bad('ran out of input');
      _bitBuffer |= input[_position++] << _bitCount;
      _bitCount += 8;
    }
    final value = _bitBuffer & ((1 << need) - 1);
    _bitBuffer >>= need;
    _bitCount -= need;
    return value;
  }

  void _stored() {
    _bitBuffer = 0;
    _bitCount = 0;
    if (_position + 4 > input.length) _bad('ran out of input');
    final length = input[_position] | input[_position + 1] << 8;
    final check = input[_position + 2] | input[_position + 3] << 8;
    if (length != (~check & 0xffff)) _bad('stored block length');
    _position += 4;
    if (_position + length > input.length) _bad('ran out of input');
    _reserve(length);
    _out.setRange(_length, _length + length, input, _position);
    _length += length;
    _position += length;
    tick();
  }

  void _dynamic() {
    final literalCount = _bits(5) + 257;
    final distanceCount = _bits(5) + 1;
    final codeLengthCount = _bits(4) + 4;
    if (literalCount > 286 || distanceCount > 30) _bad('too many codes');
    final codeLengths = List.filled(19, 0);
    for (var i = 0; i < codeLengthCount; i++) {
      codeLengths[_codeLengthOrder[i]] = _bits(3);
    }
    final codeLengthCode = _Huffman(codeLengths);
    final lengths = List.filled(literalCount + distanceCount, 0);
    for (var i = 0; i < lengths.length;) {
      final symbol = _decode(codeLengthCode);
      if (symbol < 16) {
        lengths[i++] = symbol;
        continue;
      }
      var repeat = 0;
      var value = 0;
      if (symbol == 16) {
        if (i == 0) _bad('repeat with no previous length');
        value = lengths[i - 1];
        repeat = 3 + _bits(2);
      } else if (symbol == 17) {
        repeat = 3 + _bits(3);
      } else {
        repeat = 11 + _bits(7);
      }
      if (i + repeat > lengths.length) _bad('too many lengths');
      while (repeat-- > 0) {
        lengths[i++] = value;
      }
    }
    if (lengths[256] == 0) _bad('no end-of-block code');
    _codes(
      _Huffman(lengths.sublist(0, literalCount)),
      _Huffman(lengths.sublist(literalCount)),
    );
  }

  void _codes(_Huffman literals, _Huffman distances) {
    while (true) {
      tick();
      var symbol = _decode(literals);
      if (symbol < 256) {
        _reserve(1);
        _out[_length++] = symbol;
        continue;
      }
      if (symbol == 256) return;
      symbol -= 257;
      if (symbol >= 29) _bad('length symbol $symbol');
      final length = _lengthBase[symbol] + _bits(_lengthExtra[symbol]);
      final distanceSymbol = _decode(distances);
      if (distanceSymbol >= 30) _bad('distance symbol $distanceSymbol');
      final distance =
          _distanceBase[distanceSymbol] + _bits(_distanceExtra[distanceSymbol]);
      if (distance > _length) _bad('distance $distance past the start');
      _reserve(length);
      for (var i = 0; i < length; i++) {
        _out[_length] = _out[_length - distance];
        _length++;
      }
    }
  }

  int _decode(_Huffman huffman) {
    // Past the end of the input the missing bits read as zeros; a code
    // that needs them is caught below.
    while (_bitCount < huffman.maxBits && _position < input.length) {
      _bitBuffer |= input[_position++] << _bitCount;
      _bitCount += 8;
    }
    final entry = huffman.table[_bitBuffer & ((1 << huffman.maxBits) - 1)];
    final length = entry & 15;
    if (length == 0) _bad('no such code');
    if (length > _bitCount) _bad('ran out of input');
    _bitBuffer >>= length;
    _bitCount -= length;
    return entry >> 4;
  }

  /// Makes room for [more] bytes, throwing once the output would pass the
  /// cap.
  void _reserve(int more) {
    final need = _length + more;
    if (need > maxLength) {
      throw QuarkModelParseException(
        QuarkModelParseIssue.tooLarge,
        'inflates past $maxLength bytes',
      );
    }
    if (need <= _out.length) return;
    var size = _out.length * 2;
    while (size < need) {
      size *= 2;
    }
    _out = Uint8List(size.clamp(need, maxLength))..setAll(0, _out);
  }
}

Never _bad(String detail) => throw QuarkModelParseException(
  QuarkModelParseIssue.malformed,
  'deflate: $detail',
);
