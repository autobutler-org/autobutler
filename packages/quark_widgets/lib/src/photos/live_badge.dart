import 'package:flutter/material.dart';

/// The small "LIVE" chip overlaid on the thumbnail of a live photo.
///
/// Its colors are deliberately fixed rather than themed: it is drawn on top of
/// a photograph, not on a surface, so a scrim and white text are what keep it
/// readable over whatever the image happens to be. Theme tokens would follow
/// the app into light mode and disappear against a bright picture.
///
/// Emits no `ValueKey`s; the thumbnail it sits on carries the tap target.
///
/// ```dart
/// Stack(
///   children: [
///     thumbnail,
///     if (photo.hasLiveVideo)
///       const Positioned(top: 4, left: 4, child: LiveBadge()),
///   ],
/// );
/// ```
class LiveBadge extends StatelessWidget {
  /// Creates the badge.
  const LiveBadge({super.key});

  @override
  Widget build(BuildContext context) {
    return Container(
      padding: const EdgeInsets.symmetric(horizontal: 4, vertical: 2),
      decoration: BoxDecoration(
        color: Colors.black54,
        borderRadius: BorderRadius.circular(3),
      ),
      child: const Text(
        'LIVE',
        style: TextStyle(
          color: Colors.white,
          fontSize: 9,
          fontWeight: FontWeight.w600,
          letterSpacing: 0.3,
        ),
      ),
    );
  }
}
