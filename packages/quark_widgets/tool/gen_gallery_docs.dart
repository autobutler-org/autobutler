// Extracts class-level `///` docs from lib/src into the widget gallery's
// docs.g.dart, so the gallery can show a widget's documentation next to it.
//
// Flutter has no reflection, so the docs have to be baked in at build time.
// This is deliberately a line scanner and not `package:analyzer`: the package
// has no dev dependency beyond flutter_test and flutter_lints, and a class
// declaration is one line of text.
//
// Run with `make -C packages/quark_widgets generate/docs`.
library;

import 'dart:io';

/// Matches a public class declaration that extends or implements something —
/// which, in this package, means a widget rather than a value type.
final RegExp _classPattern = RegExp(
  r'^(?:(?:abstract|base|final|interface|sealed|mixin)\s+)*'
  r'class\s+([A-Z]\w*)(?:<[^>]*>)?\s+(?:extends|implements)\b',
);

/// Pulls the `///` block immediately above each public widget class in [source]
/// out into a map of class name to doc text.
///
/// Annotations between the doc block and the declaration (`@immutable`) are
/// skipped. A class with no doc block is left out entirely.
Map<String, String> extractClassDocs(String source) {
  final docs = <String, String>{};
  final buffer = <String>[];

  for (final line in source.split('\n')) {
    final trimmed = line.trim();

    if (trimmed.startsWith('///')) {
      // Strip the marker and the single space after it, but keep any further
      // indentation: doc blocks contain indented code samples.
      final content = trimmed.substring(3);
      buffer.add(content.startsWith(' ') ? content.substring(1) : content);
      continue;
    }

    // An annotation sits between the doc block and the declaration it belongs
    // to, so it must not break the association. Anything else does.
    if (trimmed.startsWith('@')) continue;

    final match = _classPattern.firstMatch(trimmed);
    if (match != null && buffer.isNotEmpty) {
      docs[match.group(1)!] = buffer.join('\n').trimRight();
    }
    buffer.clear();
  }

  return docs;
}

/// Collects the docs for every `.dart` file under [libSrc], reading the
/// directory in sorted order so the result never depends on the filesystem.
Map<String, String> collectDocs(Directory libSrc) {
  final files =
      libSrc
          .listSync(recursive: true)
          .whereType<File>()
          .where((f) => f.path.endsWith('.dart'))
          .toList()
        ..sort((a, b) => a.path.compareTo(b.path));

  return {
    for (final file in files) ...extractClassDocs(file.readAsStringSync()),
  };
}

/// Renders [docs] as the source of `docs.g.dart`, with keys sorted so
/// regenerating without a source change is a no-op.
String renderDocsSource(Map<String, String> docs) {
  final names = docs.keys.toList()..sort();
  final buffer = StringBuffer()
    ..writeln('// GENERATED FILE - do not edit by hand.')
    ..writeln('//')
    ..writeln('// Class docs from packages/quark_widgets/lib/src, for the')
    ..writeln('// gallery\'s documentation pane. Regenerate with:')
    ..writeln('//')
    ..writeln('//     make -C packages/quark_widgets generate/docs')
    ..writeln('//')
    ..writeln('// The formatter is off so that the generator, and the test')
    ..writeln('// that checks this file is current, agree byte for byte.')
    ..writeln('// dart format off')
    ..writeln()
    ..writeln(
      '/// Documentation for each exported widget, keyed by class name.',
    )
    ..writeln('const Map<String, String> widgetDocs = {');
  for (final name in names) {
    buffer.writeln("  '$name': ${_dartString(docs[name]!)},");
  }
  buffer.writeln('};');
  return buffer.toString();
}

/// Quotes [value] as a single-line Dart string literal, escaping everything
/// that would otherwise change its meaning.
String _dartString(String value) {
  final escaped = value
      .replaceAll(r'\', r'\\')
      .replaceAll("'", r"\'")
      .replaceAll(r'$', r'\$')
      .replaceAll('\n', r'\n');
  return "'$escaped'";
}

void main() {
  final packageRoot = Directory.current;
  final libSrc = Directory('${packageRoot.path}/lib/src');
  if (!libSrc.existsSync()) {
    stderr.writeln(
      'No lib/src at ${libSrc.path}. Run this from packages/quark_widgets, or '
      'use `make -C packages/quark_widgets generate/docs`.',
    );
    exitCode = 1;
    return;
  }

  final output = File(
    '${packageRoot.path}/examples/widget_gallery/lib/docs.g.dart',
  );
  final source = renderDocsSource(collectDocs(libSrc));
  output.writeAsStringSync(source);
  stdout.writeln('Wrote ${output.path}');
}
