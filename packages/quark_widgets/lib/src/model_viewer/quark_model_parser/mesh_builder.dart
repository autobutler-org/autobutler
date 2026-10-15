import 'dart:typed_data';

import '../quark_model_mesh.dart';
import '../quark_model_parser.dart';

/// Collects vertices and triangles as a reader finds them, holding the
/// reader to its `QuarkModelLimits`: it throws once the triangle or vertex
/// cap is passed, and [tick] throws once the time cap is.
///
/// [weld] merges a vertex with an identical one already added, through an
/// open-addressing table on the coordinates' float bits, so an STL's three
/// copies of every corner become one.
class MeshBuilder {
  /// Creates a builder that starts timing now.
  MeshBuilder(this.limits) : _clock = Stopwatch()..start();

  /// The caps this builder enforces.
  final QuarkModelLimits limits;

  final Stopwatch _clock;
  int _ticks = 0;

  Float32List _positions = Float32List(3 * 1024);
  int _positionLength = 0;
  Uint32List _indices = Uint32List(3 * 1024);
  int _indexLength = 0;

  // The weld table: vertex numbers, -1 for an empty slot.
  Int32List _table = Int32List(0);
  final Float32List _scratch = Float32List(3);
  late final Uint32List _bits = Uint32List.view(_scratch.buffer);

  /// How many vertices have been added.
  int get vertexCount => _positionLength ~/ 3;

  /// How many triangles have been added.
  int get triangleCount => _indexLength ~/ 3;

  /// The most vertices the limits allow.
  int get maxVertices => limits.maxTriangles * 3;

  /// Throws when reading has run past the time cap. Cheap to call per item:
  /// it reads the clock once every 1024 calls.
  void tick() {
    if (++_ticks & 1023 != 0) return;
    if (_clock.elapsed > limits.maxDuration) {
      throw QuarkModelParseException(
        QuarkModelParseIssue.tooSlow,
        '${_clock.elapsedMilliseconds} ms',
      );
    }
  }

  /// Adds a vertex and returns its number.
  int add(double x, double y, double z) {
    if (vertexCount >= maxVertices) {
      throw QuarkModelParseException(
        QuarkModelParseIssue.tooManyTriangles,
        'over $maxVertices vertices',
      );
    }
    if (_positionLength + 3 > _positions.length) {
      _positions = _grow(_positions, _positions.length * 2) as Float32List;
    }
    _positions[_positionLength++] = x;
    _positions[_positionLength++] = y;
    _positions[_positionLength++] = z;
    return vertexCount - 1;
  }

  /// Adds a vertex unless an identical one is already there, and returns the
  /// number of whichever it is.
  int weld(double x, double y, double z) {
    if (vertexCount * 2 >= _table.length) _rehash();
    _scratch[0] = x;
    _scratch[1] = y;
    _scratch[2] = z;
    final mask = _table.length - 1;
    var slot = _hash() & mask;
    while (true) {
      final existing = _table[slot];
      if (existing < 0) break;
      final at = existing * 3;
      if (_positions[at] == _scratch[0] &&
          _positions[at + 1] == _scratch[1] &&
          _positions[at + 2] == _scratch[2]) {
        return existing;
      }
      slot = (slot + 1) & mask;
    }
    final index = add(x, y, z);
    _table[slot] = index;
    return index;
  }

  /// Adds the triangle [a], [b], [c], by vertex number.
  void triangle(int a, int b, int c) {
    if (triangleCount >= limits.maxTriangles) {
      throw QuarkModelParseException(
        QuarkModelParseIssue.tooManyTriangles,
        'over ${limits.maxTriangles} triangles',
      );
    }
    if (_indexLength + 3 > _indices.length) {
      _indices = _grow(_indices, _indices.length * 2) as Uint32List;
    }
    _indices[_indexLength++] = a;
    _indices[_indexLength++] = b;
    _indices[_indexLength++] = c;
  }

  /// Splits the polygon [corners] (vertex numbers, in order) into a fan of
  /// triangles. Fewer than three corners add nothing.
  void polygon(List<int> corners) {
    for (var i = 2; i < corners.length; i++) {
      triangle(corners[0], corners[i - 1], corners[i]);
    }
  }

  /// The finished mesh. Throws when there are no triangles or a triangle
  /// names a vertex that does not exist.
  QuarkModelMesh build({required bool isZUp}) {
    if (_indexLength == 0) {
      throw const QuarkModelParseException(QuarkModelParseIssue.noSurfaces);
    }
    final count = vertexCount;
    for (var i = 0; i < _indexLength; i++) {
      if (_indices[i] >= count) {
        throw QuarkModelParseException(
          QuarkModelParseIssue.malformed,
          'vertex ${_indices[i]} of $count',
        );
      }
    }
    return QuarkModelMesh.fromTriangles(
      positions: Float32List.fromList(
        Float32List.sublistView(_positions, 0, _positionLength),
      ),
      indices: Uint32List.fromList(
        Uint32List.sublistView(_indices, 0, _indexLength),
      ),
      isZUp: isZUp,
    );
  }

  // Every product stays under 2^53, so the hash is the same, and as well
  // spread, when ints are JavaScript doubles on the web.
  int _hash() {
    var h = 0;
    for (var i = 0; i < 3; i++) {
      final bits = _bits[i];
      h = (h * 31 + (bits & 0xffff)) & 0x0fffffff;
      h = (h * 31 + (bits >> 16)) & 0x0fffffff;
    }
    h ^= h >> 13;
    h = (h * 0x5bd1) & 0x0fffffff;
    return h ^ (h >> 15);
  }

  void _rehash() {
    final length = _table.isEmpty ? 4096 : _table.length * 2;
    _table = Int32List(length)..fillRange(0, length, -1);
    final mask = _table.length - 1;
    for (var v = 0; v < vertexCount; v++) {
      _scratch[0] = _positions[v * 3];
      _scratch[1] = _positions[v * 3 + 1];
      _scratch[2] = _positions[v * 3 + 2];
      var slot = _hash() & mask;
      while (_table[slot] >= 0) {
        slot = (slot + 1) & mask;
      }
      _table[slot] = v;
    }
  }

  static TypedData _grow(TypedData list, int length) => switch (list) {
    Float32List() => Float32List(length)..setAll(0, list),
    Uint32List() => Uint32List(length)..setAll(0, list),
    _ => throw ArgumentError.value(list),
  };
}
