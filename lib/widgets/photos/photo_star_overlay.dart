import 'package:flutter/material.dart';
import 'package:quark_icons/quark_icons.dart';

/// The favorite star painted in the bottom-right corner of a photo tile.
///
/// Renders nothing when the photo is not a favorite, so a tile can drop it
/// into its [Stack] unconditionally. Meant to sit directly in a [Stack]: the
/// star positions itself.
class PhotoStarOverlay extends StatelessWidget {
  const PhotoStarOverlay({required this.isFavorite, super.key});

  /// Whether the photo underneath is marked as a favorite. When false this
  /// widget occupies no space and paints nothing.
  final bool isFavorite;

  @override
  Widget build(BuildContext context) {
    if (!isFavorite) return const SizedBox.shrink();
    return const Positioned(
      bottom: 4,
      right: 4,
      child: Icon(
        QuarkIcons.star_rounded,
        size: 16,
        color: Colors.white,
        shadows: [Shadow(blurRadius: 4, color: Colors.black54)],
      ),
    );
  }
}
