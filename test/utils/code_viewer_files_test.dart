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
    ]) {
      expect(opensInCodeViewer(name), isTrue, reason: name);
    }
  });

  test('prose, documents and unknown files do not', () {
    for (final name in [
      'notes.txt',
      'README.md',
      'report.qdoc',
      'photo.jpg',
      'data.csv',
      'archive.zip',
      'LICENSE',
      // A folder called Makefile.d holding a text file.
      'makefile/notes.txt',
    ]) {
      expect(opensInCodeViewer(name), isFalse, reason: name);
    }
  });
}
