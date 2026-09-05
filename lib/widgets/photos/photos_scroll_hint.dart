import 'package:flutter/material.dart';
import 'package:quark_icons/quark_icons.dart';

/// The chevron that hints there is something above the photo grid.
///
/// The collapsed photos layout starts scrolled past its nav panel, so nothing
/// on screen says the panel is there. This fades in over the top edge until
/// the reader scrolls, and never takes a pointer.
///
/// ```dart
/// Stack(
///   children: [
///     grid,
///     if (showHint)
///       const Positioned(top: 0, left: 0, right: 0, child: PhotosScrollHint()),
///   ],
/// );
/// ```
class PhotosScrollHint extends StatelessWidget {
  /// Creates the hint.
  const PhotosScrollHint({super.key});

  @override
  Widget build(BuildContext context) {
    final colorScheme = Theme.of(context).colorScheme;

    return IgnorePointer(
      child: Container(
        height: 32,
        decoration: BoxDecoration(
          gradient: LinearGradient(
            begin: Alignment.topCenter,
            end: Alignment.bottomCenter,
            colors: [
              colorScheme.surface.withValues(alpha: 0.7),
              Colors.transparent,
            ],
          ),
        ),
        child: Center(
          child: Icon(
            QuarkIcons.keyboard_arrow_up_rounded,
            size: 20,
            color: colorScheme.onSurface.withValues(alpha: 0.4),
          ),
        ),
      ),
    );
  }
}
