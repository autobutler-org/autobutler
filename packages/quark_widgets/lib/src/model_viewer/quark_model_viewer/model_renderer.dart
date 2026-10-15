import 'dart:math' as math;
import 'dart:typed_data';
import 'dart:ui';

import '../quark_model_camera.dart';
import '../quark_model_mesh.dart';

/// Draws a `QuarkModelMesh` on a [Canvas] with no GPU API of its own, so it
/// works the same on every platform Flutter paints on.
///
/// Every frame it projects the vertices in perspective, sorts the triangles
/// back to front with a bucket sort (the painter's algorithm), flat-shades
/// each one from a light fixed beside the camera, lighting both sides so an
/// open or inside-out mesh still reads, and hands the lot to one
/// [Canvas.drawVertices]. Wireframe draws each triangle's edges with one
/// [Canvas.drawRawPoints] instead and skips the sort.
///
/// Its scratch buffers are sized for [mesh] once and reused frame to frame.
class ModelRenderer {
  /// Creates a renderer for [mesh].
  ModelRenderer(this.mesh)
    : _screen = Float32List(mesh.vertexCount * 2),
      _depth = Float32List(mesh.vertexCount),
      _faceDepth = Float32List(mesh.triangleCount),
      _order = Int32List(mesh.triangleCount),
      _faceBucket = Int32List(mesh.triangleCount) {
    final size = mesh.size;
    _radius = math.max(
      math.sqrt(size.x * size.x + size.y * size.y + size.z * size.z) / 2,
      1e-9,
    );
    for (var axis = 0; axis < 3; axis++) {
      _center[axis] = (mesh.min[axis] + mesh.max[axis]) / 2;
    }
  }

  /// The mesh drawn.
  final QuarkModelMesh mesh;

  /// The vertical field of view.
  static const double fieldOfView = math.pi / 4;

  static const int _buckets = 1 << 14;

  final Float64List _center = Float64List(3);
  late final double _radius;
  final Float32List _screen;
  final Float32List _depth;
  final Float32List _faceDepth;
  final Int32List _order;
  final Int32List _faceBucket;
  final Int32List _bucketStarts = Int32List(_buckets + 1);
  Float32List? _fill;
  Int32List? _colors;
  Float32List? _lines;

  /// The model's bounding radius, which the pan offsets are measured in.
  double get radius => _radius;

  /// Paints the mesh into [size] as seen by [camera]: shaded in [color], or
  /// with [wireframe] as edges in [edgeColor]. Returns how many triangles
  /// were drawn, which is every one in front of the camera.
  int paint(
    Canvas canvas,
    Size size,
    QuarkModelCamera camera, {
    required Color color,
    required Color edgeColor,
    bool wireframe = false,
  }) {
    if (mesh.isEmpty || size.isEmpty) return 0;
    final rotation = _rotation(camera);
    final distance = _radius / math.sin(fieldOfView / 2) / camera.zoom;
    final near = distance * 0.01;
    final focal = (size.shortestSide / 2) / math.tan(fieldOfView / 2);
    final cx = size.width / 2, cy = size.height / 2;
    final panX = camera.panX * _radius, panY = camera.panY * _radius;

    final positions = mesh.positions;
    for (var v = 0, p = 0; v < mesh.vertexCount; v++, p += 3) {
      final x = positions[p] - _center[0];
      final y = positions[p + 1] - _center[1];
      final z = positions[p + 2] - _center[2];
      final vx = rotation[0] * x + rotation[1] * y + rotation[2] * z + panX;
      final vy = rotation[3] * x + rotation[4] * y + rotation[5] * z + panY;
      final vz = rotation[6] * x + rotation[7] * y + rotation[8] * z;
      final w = distance - vz;
      _depth[v] = w;
      final scale = focal / math.max(w, near);
      _screen[v * 2] = cx + vx * scale;
      _screen[v * 2 + 1] = cy - vy * scale;
    }

    return wireframe
        ? _paintEdges(canvas, edgeColor, near)
        : _paintFaces(canvas, rotation, color, near);
  }

  int _paintFaces(
    Canvas canvas,
    Float64List rotation,
    Color color,
    double near,
  ) {
    final indices = mesh.indices;
    final triangles = mesh.triangleCount;
    var nearest = double.infinity, farthest = 0.0;
    for (var t = 0, i = 0; t < triangles; t++, i += 3) {
      final a = _depth[indices[i]];
      final b = _depth[indices[i + 1]];
      final c = _depth[indices[i + 2]];
      // A triangle reaching behind the near plane would project inside out.
      final depth = a < near || b < near || c < near ? -1.0 : a + b + c;
      _faceDepth[t] = depth;
      if (depth < 0) continue;
      if (depth < nearest) nearest = depth;
      if (depth > farthest) farthest = depth;
    }
    if (nearest == double.infinity) return 0;

    // Counting sort into buckets, farthest first.
    final counts = _bucketStarts..fillRange(0, _buckets + 1, 0);
    final span = farthest - nearest;
    final bucketScale = span > 0 ? (_buckets - 1) / span : 0.0;
    final buckets = _faceBucket;
    for (var t = 0; t < triangles; t++) {
      final depth = _faceDepth[t];
      final bucket = depth < 0
          ? -1
          : _buckets - 1 - ((depth - nearest) * bucketScale).toInt();
      buckets[t] = bucket;
      if (bucket >= 0) counts[bucket]++;
    }
    var drawn = 0;
    for (var b = 0; b < _buckets; b++) {
      final count = counts[b];
      counts[b] = drawn;
      drawn += count;
    }
    for (var t = 0; t < triangles; t++) {
      final bucket = buckets[t];
      if (bucket >= 0) _order[counts[bucket]++] = t;
    }

    final fill = _fill ??= Float32List(triangles * 6);
    final colors = _colors ??= Int32List(triangles * 3);
    final normals = mesh.normals;
    final screen = _screen;
    final order = _order;
    // The light turned back into model space, so each face's brightness is
    // one dot product with its own normal.
    final lx =
        rotation[0] * _light[0] +
        rotation[3] * _light[1] +
        rotation[6] * _light[2];
    final ly =
        rotation[1] * _light[0] +
        rotation[4] * _light[1] +
        rotation[7] * _light[2];
    final lz =
        rotation[2] * _light[0] +
        rotation[5] * _light[1] +
        rotation[8] * _light[2];
    final shades = _shades(color);
    for (var k = 0, f = 0, c = 0; k < drawn; k++, f += 6, c += 3) {
      final i = order[k] * 3;
      final light =
          (normals[i] * lx + normals[i + 1] * ly + normals[i + 2] * lz).abs();
      final argb = shades[(light * 255).toInt()];
      final a = indices[i] * 2, b = indices[i + 1] * 2, d = indices[i + 2] * 2;
      fill[f] = screen[a];
      fill[f + 1] = screen[a + 1];
      fill[f + 2] = screen[b];
      fill[f + 3] = screen[b + 1];
      fill[f + 4] = screen[d];
      fill[f + 5] = screen[d + 1];
      colors[c] = argb;
      colors[c + 1] = argb;
      colors[c + 2] = argb;
    }
    canvas.drawVertices(
      Vertices.raw(
        VertexMode.triangles,
        Float32List.sublistView(fill, 0, drawn * 6),
        colors: Int32List.sublistView(colors, 0, drawn * 3),
      ),
      // Paint the vertex colors as they are.
      BlendMode.dst,
      Paint(),
    );
    return drawn;
  }

  int _paintEdges(Canvas canvas, Color edgeColor, double near) {
    final indices = mesh.indices;
    final lines = _lines ??= Float32List(mesh.triangleCount * 12);
    var length = 0;
    for (var i = 0; i < indices.length; i += 3) {
      final a = indices[i], b = indices[i + 1], c = indices[i + 2];
      if (_depth[a] < near || _depth[b] < near || _depth[c] < near) continue;
      final ax = _screen[a * 2], ay = _screen[a * 2 + 1];
      final bx = _screen[b * 2], by = _screen[b * 2 + 1];
      final cx = _screen[c * 2], cy = _screen[c * 2 + 1];
      lines
        ..[length] = ax
        ..[length + 1] = ay
        ..[length + 2] = bx
        ..[length + 3] = by
        ..[length + 4] = bx
        ..[length + 5] = by
        ..[length + 6] = cx
        ..[length + 7] = cy
        ..[length + 8] = cx
        ..[length + 9] = cy
        ..[length + 10] = ax
        ..[length + 11] = ay;
      length += 12;
    }
    canvas.drawRawPoints(
      PointMode.lines,
      Float32List.sublistView(lines, 0, length),
      Paint()
        ..color = edgeColor
        ..strokeWidth = 0,
    );
    return length ~/ 12;
  }

  /// The model-to-view rotation, row-major: Z-up files turned Y-up, then
  /// the camera's yaw about Y, then its pitch about X.
  Float64List _rotation(QuarkModelCamera camera) {
    final cy = math.cos(camera.yaw), sy = math.sin(camera.yaw);
    final cp = math.cos(camera.pitch), sp = math.sin(camera.pitch);
    // pitch × yaw.
    final m = Float64List.fromList([
      cy, 0, sy, //
      sp * sy, cp, -sp * cy,
      -cp * sy, sp, cp * cy,
    ]);
    if (!mesh.isZUp) return m;
    // × (x, y, z) → (x, z, -y), which stands a Z-up model upright.
    return Float64List.fromList([
      m[0], -m[2], m[1], //
      m[3], -m[5], m[4],
      m[6], -m[8], m[7],
    ]);
  }

  static const double _ambient = 0.28;
  static final Float64List _light = () {
    const x = -0.35, y = 0.55, z = 0.75;
    final length = math.sqrt(x * x + y * y + z * z);
    return Float64List.fromList([x / length, y / length, z / length]);
  }();

  Color? _shadesColor;
  final Int32List _shadeTable = Int32List(256);

  /// [color] at each of 256 brightnesses, from the ambient floor to full,
  /// built once per color.
  Int32List _shades(Color color) {
    if (_shadesColor == color) return _shadeTable;
    _shadesColor = color;
    int channel(double value) => math.min(255, (value * 255).round());
    for (var level = 0; level < 256; level++) {
      final shade = _ambient + (1 - _ambient) * level / 255;
      _shadeTable[level] =
          0xff000000 |
          (channel(color.r * shade) << 16) |
          (channel(color.g * shade) << 8) |
          channel(color.b * shade);
    }
    return _shadeTable;
  }
}
