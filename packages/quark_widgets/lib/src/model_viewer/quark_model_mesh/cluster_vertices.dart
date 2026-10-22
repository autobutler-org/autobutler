import 'dart:math' as math;
import 'dart:typed_data';

import '../quark_model_mesh.dart';

/// [mesh] cut to at most [maxTriangles] triangles by vertex clustering, for
/// `QuarkModelMesh.withPreview`: every vertex inside one cell of a cubic
/// grid over the bounds becomes their average, and a triangle with two
/// corners in one cell is dropped. The grid starts near the resolution a
/// flat sheet would need and coarsens until few enough triangles survive.
///
/// The result keeps [mesh]'s bounds, so the viewer frames both the same.
/// A mesh with no extent, which every grid collapses, keeps every k-th
/// triangle instead.
QuarkModelMesh clusterVertices(QuarkModelMesh mesh, int maxTriangles) {
  final size = mesh.size;
  final extent = math.max(size.x, math.max(size.y, size.z));
  // A sheet across the grid keeps about two triangles per cell.
  var cells = math.sqrt(maxTriangles / 2).floor();
  while (extent > 0 && cells > 1) {
    final preview = _cluster(mesh, cells, extent);
    final count = preview.triangleCount;
    if (count <= maxTriangles) return preview;
    // Triangles scale with the square of the grid on a surface.
    cells = math.min(
      cells - 1,
      (cells * math.sqrt(maxTriangles / count) * 0.95).floor(),
    );
  }
  return _everyKth(mesh, maxTriangles);
}

QuarkModelMesh _cluster(QuarkModelMesh mesh, int cells, double extent) {
  final positions = mesh.positions;
  final vertices = mesh.vertexCount;
  final scale = cells / extent;
  final clusterOf = Int32List(vertices);
  final cellToCluster = <int, int>{};
  final sums = Float64List(vertices * 3);
  final counts = Int32List(vertices);
  var clusters = 0;
  for (var v = 0, p = 0; v < vertices; v++, p += 3) {
    var cell = 0;
    for (var axis = 0; axis < 3; axis++) {
      final at = ((positions[p + axis] - mesh.min[axis]) * scale).floor();
      cell = cell * cells + at.clamp(0, cells - 1);
    }
    final cluster = cellToCluster.putIfAbsent(cell, () => clusters++);
    clusterOf[v] = cluster;
    sums[cluster * 3] += positions[p];
    sums[cluster * 3 + 1] += positions[p + 1];
    sums[cluster * 3 + 2] += positions[p + 2];
    counts[cluster]++;
  }

  final source = mesh.indices;
  final kept = Uint32List(source.length);
  var length = 0;
  for (var i = 0; i < source.length; i += 3) {
    final a = clusterOf[source[i]];
    final b = clusterOf[source[i + 1]];
    final c = clusterOf[source[i + 2]];
    if (a == b || b == c || a == c) continue;
    kept
      ..[length++] = a
      ..[length++] = b
      ..[length++] = c;
  }

  // Only the clusters a kept triangle names, renumbered in order.
  final renumber = Int32List(clusters)..fillRange(0, clusters, -1);
  final indices = Uint32List(length);
  var used = 0;
  for (var i = 0; i < length; i++) {
    final cluster = kept[i];
    if (renumber[cluster] < 0) renumber[cluster] = used++;
    indices[i] = renumber[cluster];
  }
  final averaged = Float32List(used * 3);
  for (var cluster = 0; cluster < clusters; cluster++) {
    final at = renumber[cluster];
    if (at < 0) continue;
    for (var axis = 0; axis < 3; axis++) {
      averaged[at * 3 + axis] = sums[cluster * 3 + axis] / counts[cluster];
    }
  }
  return _framedLike(
    mesh,
    QuarkModelMesh.fromTriangles(positions: averaged, indices: indices),
  );
}

QuarkModelMesh _everyKth(QuarkModelMesh mesh, int maxTriangles) {
  final step = (mesh.triangleCount / maxTriangles).ceil();
  final count = (mesh.triangleCount / step).ceil();
  final indices = Uint32List(count * 3);
  for (var t = 0; t < count; t++) {
    indices.setRange(t * 3, t * 3 + 3, mesh.indices, t * step * 3);
  }
  return _framedLike(
    mesh,
    QuarkModelMesh.fromTriangles(positions: mesh.positions, indices: indices),
  );
}

/// [preview] with [mesh]'s bounds and orientation.
QuarkModelMesh _framedLike(QuarkModelMesh mesh, QuarkModelMesh preview) =>
    QuarkModelMesh(
      positions: preview.positions,
      indices: preview.indices,
      normals: preview.normals,
      min: mesh.min,
      max: mesh.max,
      isZUp: mesh.isZUp,
    );
