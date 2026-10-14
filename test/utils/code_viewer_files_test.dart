import 'package:flutter_test/flutter_test.dart';
import 'package:quark/utils/code_viewer_files.dart';

/// Which files Files opens in the read-only code viewer (#2991).
void main() {
  test('source, config and log files open in the code viewer', () {
    for (final name in [
      'main.go',
      'config.YAML',
      'data.json',
      'server.log',
      'Dockerfile',
      'src/Makefile',
      'deploy/run.sh',
      // Plain text reads like any other text file, with Edit a tap away
      // (#3186).
      'notes.txt',
      'makefile/notes.txt',
      'docs/guide.rst',
    ]) {
      expect(opensInCodeViewer(name), isTrue, reason: name);
    }
  });

  test('Markdown, documents and unknown files do not', () {
    for (final name in [
      'README.md',
      'notes/trip.markdown',
      'report.qdoc',
      'photo.jpg',
      'data.csv',
      'archive.zip',
      'LICENSE',
    ]) {
      expect(opensInCodeViewer(name), isFalse, reason: name);
    }
  });
}
