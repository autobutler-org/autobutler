import 'dart:typed_data';

import 'package:flutter/material.dart';

class ImageViewerPage extends StatelessWidget {
  final Uint8List bytes;
  final String name;

  const ImageViewerPage({super.key, required this.bytes, required this.name});

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: Text(name)),
      body: Center(
        child: InteractiveViewer(
          child: Image.memory(
            bytes,
            fit: BoxFit.contain,
            errorBuilder: (c, e, st) =>
                const Icon(Icons.broken_image, size: 64),
          ),
        ),
      ),
    );
  }
}
