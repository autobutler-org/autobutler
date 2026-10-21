import 'dart:math' as math;
import 'dart:typed_data';

import 'package:flutter_test/flutter_test.dart';
import 'package:quark_widgets/quark_widgets.dart';

/// A wavy square sheet of [side] x [side] vertices: 2 (side - 1)² triangles.
QuarkModelMesh sheet(int side) {
  final positions = Float32List(side * side * 3);
  for (var y = 0; y < side; y++) {
    for (var x = 0; x < side; x++) {
      final i = (y * side + x) * 3;
      positions[i] = x / side;
      positions[i + 1] = y / side;
      positions[i + 2] = 0.05 * math.sin(x / 7) * math.cos(y / 5);
    }
  }
  final indices = Uint32List((side - 1) * (side - 1) * 6);
  var k = 0;
  for (var y = 0; y < side - 1; y++) {
    for (var x = 0; x < side - 1; x++) {
      final a = y * side + x;
      indices
        ..[k++] = a
        ..[k++] = a + 1
        ..[k++] = a + side
        ..[k++] = a + 1
        ..[k++] = a + side + 1
        ..[k++] = a + side;
    }
  }
  return QuarkModelMesh.fromTriangles(positions: positions, indices: indices);
}

int bytesOf(QuarkModelMesh mesh) =>
    mesh.positions.lengthInBytes +
    mesh.indices.lengthInBytes +
    mesh.normals.lengthInBytes;

/// `QuarkModelMesh.withPreview` (#3231): the coarse copy the viewer draws
/// while a detailed model moves.
void main() {
  test('a mesh within the budget carries no preview', () {
    final mesh = sheet(20);
    final previewed = mesh.withPreview(mesh.triangleCount);
    expect(previewed.preview, isNull);
    expect(previewed.positions, same(mesh.positions));
    expect(previewed.indices, same(mesh.indices));
  });

  test('a detailed mesh gets a preview within the budget and its bounds', () {
    // About two million triangles, four times the viewer's old cap.
    final mesh = sheet(1001);
    expect(mesh.triangleCount, 2000000);
    const budget = 200000;
    final previewed = mesh.withPreview(budget);
    final preview = previewed.preview!;

    expect(previewed.triangleCount, mesh.triangleCount);
    expect(previewed.positions, same(mesh.positions));
    expect(preview.triangleCount, lessThanOrEqualTo(budget));
    // Coarser, but still the shape: at least a quarter of the budget.
    expect(preview.triangleCount, greaterThan(budget ~/ 4));
    expect(preview.min, mesh.min);
    expect(preview.max, mesh.max);
    expect(preview.isZUp, mesh.isZUp);
    expect(preview.preview, isNull);
    for (final index in preview.indices) {
      expect(index, lessThan(preview.vertexCount));
    }
    for (var t = 0; t < preview.indices.length; t += 3) {
      final a = preview.indices[t];
      final b = preview.indices[t + 1];
      final c = preview.indices[t + 2];
      expect(a != b && b != c && a != c, isTrue, reason: 'triangle ${t ~/ 3}');
    }
  });

  test('the preview costs memory bounded by its budget, not the mesh', () {
    final mesh = sheet(1001);
    const budget = 50000;
    final preview = mesh.withPreview(budget).preview!;
    // Positions, indices and normals: at most three vertices and 24 bytes
    // of index and normal per triangle, whatever the mesh was.
    expect(bytesOf(preview), lessThanOrEqualTo(budget * (3 * 12 + 24)));
    expect(bytesOf(preview), lessThan(bytesOf(mesh) ~/ 20));
  });

  test('a mesh with no extent still gets a preview within the budget', () {
    final positions = Float32List(3 * 1000);
    final indices = Uint32List.fromList([
      for (var t = 0; t < 900; t++) ...[t, t + 1, t + 2],
    ]);
    final mesh = QuarkModelMesh.fromTriangles(
      positions: positions,
      indices: indices,
    );
    final preview = mesh.withPreview(100).preview!;
    expect(preview.triangleCount, lessThanOrEqualTo(100));
    expect(preview.triangleCount, greaterThan(0));
  });
}
