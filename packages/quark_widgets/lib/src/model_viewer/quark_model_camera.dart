import 'dart:math' as math;

/// Where a `QuarkModelViewer` looks at its model from: an orbit around the
/// model's center, a distance, and a sideways shift.
///
/// Immutable; every gesture makes a new one. [initial] is the three-quarter
/// view from above the viewer opens on and returns to on reset.
///
/// ```dart
/// final turned = QuarkModelCamera.initial.orbit(0.2, 0);
/// ```
class QuarkModelCamera {
  /// Creates a camera. Out-of-range values are clamped.
  const QuarkModelCamera({
    this.yaw = -math.pi / 5,
    this.pitch = math.pi / 7,
    this.zoom = 1,
    this.panX = 0,
    this.panY = 0,
  });

  /// The view the viewer opens on.
  static const QuarkModelCamera initial = QuarkModelCamera();

  /// The closest the camera gets, as a multiple of the framing distance.
  static const double maxZoom = 40;

  /// The farthest the camera gets, as a fraction of the framing distance.
  static const double minZoom = 0.2;

  /// How far the pitch goes either way, just short of straight down or up so
  /// "up" never flips.
  static const double maxPitch = math.pi / 2 - 0.01;

  /// Turn around the model's up axis, in radians.
  final double yaw;

  /// Tilt toward looking down on the model, in radians; positive looks from
  /// above.
  final double pitch;

  /// How much closer than the framing distance the camera is; 1 fits the
  /// whole model in view.
  final double zoom;

  /// Sideways shift of the model on screen, in model radii.
  final double panX;

  /// Upward shift of the model on screen, in model radii.
  final double panY;

  /// Turned by [dYaw] and tilted by [dPitch] radians.
  QuarkModelCamera orbit(double dYaw, double dPitch) => QuarkModelCamera(
    yaw: yaw + dYaw,
    pitch: (pitch + dPitch).clamp(-maxPitch, maxPitch),
    zoom: zoom,
    panX: panX,
    panY: panY,
  );

  /// Shifted by [dx] and [dy] model radii.
  QuarkModelCamera pan(double dx, double dy) => QuarkModelCamera(
    yaw: yaw,
    pitch: pitch,
    zoom: zoom,
    panX: panX + dx,
    panY: panY + dy,
  );

  /// Moved closer by [factor]; under 1 moves away.
  QuarkModelCamera zoomBy(double factor) => QuarkModelCamera(
    yaw: yaw,
    pitch: pitch,
    zoom: (zoom * factor).clamp(minZoom, maxZoom),
    panX: panX,
    panY: panY,
  );

  @override
  bool operator ==(Object other) =>
      other is QuarkModelCamera &&
      other.yaw == yaw &&
      other.pitch == pitch &&
      other.zoom == zoom &&
      other.panX == panX &&
      other.panY == panY;

  @override
  int get hashCode => Object.hash(yaw, pitch, zoom, panX, panY);

  @override
  String toString() =>
      'QuarkModelCamera(yaw: $yaw, pitch: $pitch, zoom: $zoom, '
      'pan: $panX, $panY)';
}
