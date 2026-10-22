import 'dart:math' as math;
import 'dart:ui' show ImageFilter;

import 'package:flutter/rendering.dart';

import '../model/element_shadow.dart';
import '../theme/slide_theme.dart';

/// Where [shadow] falls in the space of an element turned [rotation]
/// degrees clockwise: the shadow's slide offset turned back by the
/// rotation, so a turned element's shadow still falls the slide's way.
Offset shadowOffset(ElementShadow shadow, double rotation) {
  final a = rotation * math.pi / 180;
  final cos = math.cos(a), sin = math.sin(a);
  return Offset(
    shadow.x * cos + shadow.y * sin,
    -shadow.x * sin + shadow.y * cos,
  );
}

/// The standard deviation of the Gaussian blur [shadow] is drawn with:
/// half its blur distance, so the edge has all but faded over it.
double shadowSigma(ElementShadow shadow) => shadow.blur / 2;

/// Paints [shadow] of what [silhouette] draws on [canvas], in the space of
/// an element turned [rotation] degrees: the silhouette in the shadow's
/// color, moved by its offset and blurred. Paint the element itself after.
void paintShadow(
  Canvas canvas,
  ElementShadow shadow,
  double rotation,
  SlideTheme theme,
  void Function(Canvas canvas) silhouette,
) {
  final sigma = shadowSigma(shadow);
  final offset = shadowOffset(shadow, rotation);
  canvas.saveLayer(
    null,
    Paint()
      ..colorFilter = ColorFilter.mode(
        Color(shadow.color.resolve(theme)),
        BlendMode.srcIn,
      )
      ..imageFilter =
          sigma > 0 ? ImageFilter.blur(sigmaX: sigma, sigmaY: sigma) : null,
  );
  canvas.translate(offset.dx, offset.dy);
  silhouette(canvas);
  canvas.restore();
}

/// Paints the [shadow] a rectangular element turned [rotation] degrees
/// casts from the box it is given — a picture's frame — under whatever its
/// child draws.
class SlideShadowPainter extends CustomPainter {
  /// Creates a painter for [shadow].
  const SlideShadowPainter(this.shadow, this.rotation, this.theme);

  /// The shadow to paint.
  final ElementShadow shadow;

  /// The element's rotation in degrees clockwise.
  final double rotation;

  /// The theme a role color resolves against.
  final SlideTheme theme;

  @override
  void paint(Canvas canvas, Size size) => paintShadow(
        canvas,
        shadow,
        rotation,
        theme,
        (canvas) => canvas.drawRect(Offset.zero & size, Paint()),
      );

  @override
  bool shouldRepaint(SlideShadowPainter oldDelegate) =>
      oldDelegate.shadow != shadow ||
      oldDelegate.rotation != rotation ||
      oldDelegate.theme != theme;
}
