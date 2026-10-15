import 'package:flutter/material.dart';

import '../../theme/quark_tokens.dart';
import '../quark_model_mesh.dart';

/// The strip under a `QuarkModelViewer`'s canvas: the model's bounding-box
/// size along each axis, its triangle and vertex counts, how to move it, and
/// the caller's status line. The parts wrap onto more lines when the width
/// runs out.
///
/// Sizes are in the file's own unit, which a model file does not record
/// (STL is usually millimeters), so they carry no unit name.
///
/// Key prefixes: `model_viewer_size`, `model_viewer_triangles`,
/// `model_viewer_vertices`, `model_viewer_hint` and `model_viewer_status`.
///
/// ```dart
/// ModelViewerStats(mesh: mesh, statusLabel: 'part.stl');
/// ```
class ModelViewerStats extends StatelessWidget {
  /// Creates the strip for [mesh].
  const ModelViewerStats({required this.mesh, this.statusLabel, super.key});

  /// The model described.
  final QuarkModelMesh mesh;

  /// A line from the caller, shown after the counts; null shows nothing.
  final String? statusLabel;

  /// The touch and mouse gestures, spelled out for someone who has never
  /// used a 3D viewer.
  static const String hint =
      'Drag to turn. Two fingers or right-drag to move. Pinch or scroll to '
      'zoom.';

  @override
  Widget build(BuildContext context) {
    final tokens = QuarkTokens.of(context);
    final status = statusLabel;
    final size = mesh.size;
    final strong = TextStyle(fontSize: 12, color: tokens.foreground);
    final muted = TextStyle(fontSize: 12, color: tokens.mutedForeground);
    return DecoratedBox(
      decoration: BoxDecoration(
        color: tokens.sidebar,
        border: Border(top: BorderSide(color: tokens.border)),
      ),
      child: Padding(
        padding: EdgeInsets.symmetric(
          horizontal: tokens.spacingMd,
          vertical: tokens.spacingSm,
        ),
        child: Wrap(
          spacing: tokens.spacingLg,
          runSpacing: tokens.spacingXs,
          children: [
            Text(
              'X ${formatModelLength(size.x)} · '
              'Y ${formatModelLength(size.y)} · '
              'Z ${formatModelLength(size.z)}',
              key: const ValueKey('model_viewer_size'),
              style: strong,
            ),
            Text(
              _count(mesh.triangleCount, 'triangle'),
              key: const ValueKey('model_viewer_triangles'),
              style: strong,
            ),
            Text(
              _count(mesh.vertexCount, 'vertex', 'vertices'),
              key: const ValueKey('model_viewer_vertices'),
              style: strong,
            ),
            if (status != null)
              Text(
                status,
                key: const ValueKey('model_viewer_status'),
                style: muted,
              ),
            Text(hint, key: const ValueKey('model_viewer_hint'), style: muted),
          ],
        ),
      ),
    );
  }
}

/// [value] with at most three significant decimals and no trailing zeros:
/// `120`, `45.25`, `0.125`.
String formatModelLength(double value) {
  final magnitude = value.abs();
  final decimals = magnitude >= 100
      ? 1
      : magnitude >= 1
      ? 2
      : 3;
  var text = value.toStringAsFixed(decimals);
  if (text.contains('.')) {
    text = text
        .replaceFirst(RegExp(r'0+$'), '')
        .replaceFirst(RegExp(r'\.$'), '');
  }
  return text == '-0' ? '0' : text;
}

String _count(int count, String one, [String? many]) {
  final digits = count.toString();
  final grouped = StringBuffer();
  for (var i = 0; i < digits.length; i++) {
    if (i > 0 && (digits.length - i) % 3 == 0) grouped.write(',');
    grouped.write(digits[i]);
  }
  return '$grouped ${count == 1 ? one : many ?? '${one}s'}';
}
