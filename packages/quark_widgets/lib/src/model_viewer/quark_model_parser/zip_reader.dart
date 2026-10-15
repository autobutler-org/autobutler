import 'dart:convert';
import 'dart:typed_data';

import '../quark_model_parser.dart';
import 'inflate.dart';

const int _endSignature = 0x06054b50;
const int _centralSignature = 0x02014b50;
const int _localSignature = 0x04034b50;

/// The most entries an archive may list. A 3MF holds a handful of parts;
/// a directory of millions is an attack on the reader, not a model.
const int maxZipEntries = 10000;

/// Random access to the entries of a zip held in memory, as a 3MF package
/// is: the central directory is read once, and an entry is decompressed
/// only when [read] asks for it, so the rest of the archive is never
/// inflated.
///
/// Only stored and DEFLATE entries are read; an encrypted or otherwise
/// compressed one, a ZIP64 archive, or a directory that runs off the end of
/// the bytes is `malformed`.
class ZipReader {
  /// Reads the central directory of [bytes].
  ZipReader(this.bytes) : _data = ByteData.sublistView(bytes) {
    final end = _findEnd();
    final count = _data.getUint16(end + 10, Endian.little);
    final offset = _data.getUint32(end + 16, Endian.little);
    if (count == 0xffff || offset == 0xffffffff) _bad('ZIP64 archive');
    if (count > maxZipEntries) _bad('$count entries');
    var at = offset;
    for (var i = 0; i < count; i++) {
      _need(at, 46);
      if (_data.getUint32(at, Endian.little) != _centralSignature) {
        _bad('central directory entry $i');
      }
      final nameLength = _data.getUint16(at + 28, Endian.little);
      final extraLength = _data.getUint16(at + 30, Endian.little);
      final commentLength = _data.getUint16(at + 32, Endian.little);
      _need(at + 46, nameLength);
      final name = utf8.decode(
        Uint8List.sublistView(bytes, at + 46, at + 46 + nameLength),
        allowMalformed: true,
      );
      _entries.putIfAbsent(name.toLowerCase(), () => at);
      at += 46 + nameLength + extraLength + commentLength;
    }
  }

  /// The archive.
  final Uint8List bytes;
  final ByteData _data;

  /// Each entry's central directory record, by lowercased name: OPC part
  /// names compare without case.
  final Map<String, int> _entries = {};

  /// Whether the archive has an entry called [name], ignoring case.
  bool contains(String name) => _entries.containsKey(name.toLowerCase());

  /// The contents of the entry called [name], ignoring case, or null when
  /// there is none. More than [maxLength] bytes of it throws `tooLarge`
  /// whatever size the archive declares; [tick] keeps the time cap.
  Uint8List? read(
    String name, {
    required int maxLength,
    required void Function() tick,
  }) {
    final at = _entries[name.toLowerCase()];
    if (at == null) return null;
    final flags = _data.getUint16(at + 8, Endian.little);
    final method = _data.getUint16(at + 10, Endian.little);
    final compressed = _data.getUint32(at + 20, Endian.little);
    final size = _data.getUint32(at + 24, Endian.little);
    final local = _data.getUint32(at + 42, Endian.little);
    if (flags & 1 != 0) _bad('$name is encrypted');
    if (compressed == 0xffffffff || size == 0xffffffff) _bad('ZIP64 entry');
    _need(local, 30);
    if (_data.getUint32(local, Endian.little) != _localSignature) {
      _bad('$name local header');
    }
    final start =
        local +
        30 +
        _data.getUint16(local + 26, Endian.little) +
        _data.getUint16(local + 28, Endian.little);
    _need(start, compressed);
    final content = Uint8List.sublistView(bytes, start, start + compressed);
    switch (method) {
      case 0:
        if (compressed > maxLength) {
          throw QuarkModelParseException(
            QuarkModelParseIssue.tooLarge,
            '$name is $compressed bytes',
          );
        }
        return content;
      case 8:
        return inflate(
          content,
          maxLength: maxLength,
          tick: tick,
          expectedLength: size,
        );
      default:
        _bad('$name uses compression method $method');
    }
  }

  /// The offset of the end of central directory record, which sits in the
  /// last 64 KiB, before the archive comment.
  int _findEnd() {
    final last = bytes.length - 22;
    final first = last - 0xffff < 0 ? 0 : last - 0xffff;
    for (var at = last; at >= first; at--) {
      if (_data.getUint32(at, Endian.little) == _endSignature) return at;
    }
    _bad('not a zip archive');
  }

  void _need(int at, int length) {
    if (at + length > bytes.length) _bad('runs past the end of the archive');
  }
}

Never _bad(String detail) => throw QuarkModelParseException(
  QuarkModelParseIssue.malformed,
  'zip: $detail',
);
