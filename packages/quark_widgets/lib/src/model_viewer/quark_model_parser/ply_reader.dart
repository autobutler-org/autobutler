import 'dart:typed_data';

import '../quark_model_mesh.dart';
import '../quark_model_parser.dart';
import 'mesh_builder.dart';
import 'text_cursor.dart';

/// Reads a Stanford PLY file in any of its three encodings.
///
/// The header declares elements in order, each a count of rows with typed
/// properties; the body holds them in that order. The `vertex` element's
/// `x`, `y` and `z`, and the `face` element's index list (`vertex_indices`,
/// or `vertex_index`), are kept; every other element and property is read
/// past. A face is split into a triangle fan.
QuarkModelMesh readPly(Uint8List bytes, QuarkModelLimits limits) {
  final header = _readHeader(bytes);
  final builder = MeshBuilder(limits);
  for (final element in header.elements) {
    // Every row takes a byte at least, so a count past the file's length is
    // a lie, and is refused before anything is sized from it.
    if (element.count > bytes.length) _malformed('${element.count} rows');
    if (element.name == 'vertex' && element.count > builder.maxVertices ||
        element.name == 'face' && element.count > limits.maxTriangles) {
      throw QuarkModelParseException(
        QuarkModelParseIssue.tooManyTriangles,
        '${element.count} ${element.name}',
      );
    }
  }
  final body = header.encoding == _Encoding.ascii
      ? _AsciiBody(TextCursor(bytes, header.bodyStart))
      : _BinaryBody(
          ByteData.sublistView(bytes),
          header.bodyStart,
          header.encoding == _Encoding.littleEndian
              ? Endian.little
              : Endian.big,
        );
  final corners = <int>[];
  for (final element in header.elements) {
    final isVertex = element.name == 'vertex';
    final isFace = element.name == 'face';
    final listIndex = isFace ? _faceListIndex(element) : -1;
    final xyz = isVertex
        ? [
            for (final axis in ['x', 'y', 'z']) _propertyIndex(element, axis),
          ]
        : const <int>[];
    if (isVertex && xyz.contains(-1)) _malformed('vertex without x, y, z');
    final values = Float64List(element.properties.length);
    for (var row = 0; row < element.count; row++) {
      builder.tick();
      body.startRow();
      for (var p = 0; p < element.properties.length; p++) {
        final property = element.properties[p];
        final listType = property.listType;
        if (listType == null) {
          values[p] = body.read(property.type);
          continue;
        }
        final length = body.read(listType).toInt();
        if (length < 0) _malformed('list of $length');
        if (p == listIndex) corners.clear();
        for (var i = 0; i < length; i++) {
          final value = body.read(property.type);
          if (p == listIndex) {
            if (value < 0) _malformed('face index $value');
            corners.add(value.toInt());
          }
        }
      }
      if (isVertex) builder.add(values[xyz[0]], values[xyz[1]], values[xyz[2]]);
      if (isFace) builder.polygon(corners);
    }
  }
  return builder.build(isZUp: false);
}

Never _malformed(String detail) =>
    throw QuarkModelParseException(QuarkModelParseIssue.malformed, detail);

enum _Encoding { ascii, littleEndian, bigEndian }

/// A property's scalar type: its size in bytes, and how to read it.
enum _Type {
  int8(1),
  uint8(1),
  int16(2),
  uint16(2),
  int32(4),
  uint32(4),
  float32(4),
  float64(8);

  const _Type(this.size);

  final int size;

  static _Type parse(String name) => switch (name) {
    'char' || 'int8' => int8,
    'uchar' || 'uint8' => uint8,
    'short' || 'int16' => int16,
    'ushort' || 'uint16' => uint16,
    'int' || 'int32' => int32,
    'uint' || 'uint32' => uint32,
    'float' || 'float32' => float32,
    'double' || 'float64' => float64,
    _ => _malformed('property type $name'),
  };
}

/// A property: a scalar of [type], or with [listType] a list whose length is
/// a [listType] and whose items are [type]s.
typedef _Property = ({String name, _Type type, _Type? listType});

typedef _Element = ({String name, int count, List<_Property> properties});

typedef _Header = ({
  _Encoding encoding,
  List<_Element> elements,
  int bodyStart,
});

_Header _readHeader(Uint8List bytes) {
  final cursor = TextCursor(bytes);
  if (!cursor.word('ply')) _malformed('no ply magic');
  cursor.nextLine();
  _Encoding? encoding;
  final elements = <_Element>[];
  while (!cursor.atEnd) {
    final keyword = cursor.readWord();
    switch (keyword) {
      case 'format':
        encoding = switch (cursor.readWord()) {
          'ascii' => _Encoding.ascii,
          'binary_little_endian' => _Encoding.littleEndian,
          'binary_big_endian' => _Encoding.bigEndian,
          final other => _malformed('format $other'),
        };
      case 'element':
        elements.add((
          name: cursor.readWord(),
          count: cursor.readInt(),
          properties: <_Property>[],
        ));
      case 'property':
        if (elements.isEmpty) _malformed('property before element');
        final first = cursor.readWord();
        final listType = first == 'list'
            ? _Type.parse(cursor.readWord())
            : null;
        final type = _Type.parse(listType == null ? first : cursor.readWord());
        final _Property property = (
          name: cursor.readWord(),
          type: type,
          listType: listType,
        );
        elements.last.properties.add(property);
      case 'end_header':
        cursor.nextLine();
        if (encoding == null) _malformed('no format line');
        return (
          encoding: encoding,
          elements: elements,
          bodyStart: cursor.position,
        );
    }
    // `comment` and `obj_info` lines, and the rest of any line read above.
    cursor.nextLine();
  }
  _malformed('no end_header');
}

int _propertyIndex(_Element element, String name) =>
    element.properties.indexWhere((p) => p.name == name && p.listType == null);

int _faceListIndex(_Element element) {
  final named = element.properties.indexWhere(
    (p) =>
        p.listType != null &&
        (p.name == 'vertex_indices' || p.name == 'vertex_index'),
  );
  return named >= 0
      ? named
      : element.properties.indexWhere((p) => p.listType != null);
}

/// The rows after the header, read one value at a time.
sealed class _Body {
  /// Called at the start of every row.
  void startRow() {}

  /// Reads the next value, which the header says is a [type].
  double read(_Type type);
}

class _AsciiBody extends _Body {
  _AsciiBody(this.cursor);

  final TextCursor cursor;

  @override
  void startRow() => cursor.skipWhitespace();

  @override
  double read(_Type type) {
    // A row may wrap; numbers are separated by any whitespace.
    cursor.skipWhitespace();
    if (cursor.atEnd) _malformed('body ends early');
    return cursor.readDouble();
  }
}

class _BinaryBody extends _Body {
  _BinaryBody(this.data, this.offset, this.endian);

  final ByteData data;
  int offset;
  final Endian endian;

  @override
  double read(_Type type) {
    final at = offset;
    offset += type.size;
    return switch (type) {
      _Type.int8 => data.getInt8(at),
      _Type.uint8 => data.getUint8(at),
      _Type.int16 => data.getInt16(at, endian),
      _Type.uint16 => data.getUint16(at, endian),
      _Type.int32 => data.getInt32(at, endian),
      _Type.uint32 => data.getUint32(at, endian),
      _Type.float32 => data.getFloat32(at, endian),
      _Type.float64 => data.getFloat64(at, endian),
    }.toDouble();
  }
}
