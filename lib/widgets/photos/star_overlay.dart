import 'package:flutter/material.dart';
import 'package:quark_icons/quark_icons.dart';

/// The favorite star in the corner of a photo tile, or nothing when the photo
/// is not a favorite.
class StarOverlay extends StatelessWidget {
  final bool isFavorite;

  const StarOverlay({super.key, required this.isFavorite});

  @override
  Widget build(BuildContext context) {
    if (!isFavorite) return const SizedBox.shrink();
    return Positioned(
      bottom: 4,
      right: 4,
      child: Icon(
        QuarkIcons.star_rounded,
        size: 16,
        color: Colors.white,
        shadows: const [Shadow(blurRadius: 4, color: Colors.black54)],
      ),
    );
  }
}
