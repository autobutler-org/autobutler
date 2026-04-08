import 'package:flutter/material.dart';

class SpreadsheetEditorPage extends StatelessWidget {
  final String filePath;
  final String deviceSerial;

  const SpreadsheetEditorPage({
    required this.filePath,
    this.deviceSerial = '',
    super.key,
  });

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(
        title: Text(filePath.split('/').last),
      ),
      body: const Center(
        child: Text(
          'Editor coming soon',
          style: TextStyle(fontSize: 18),
        ),
      ),
    );
  }
}
