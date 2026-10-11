// cspell:ignore containerfile gnumakefile rakefile

/// The languages `highlightCodeLine` knows how to color.
///
/// [plain] is the fallback for anything else and renders uncolored.
enum CodeLanguage {
  /// No highlighting.
  plain,

  /// JSON, with keys colored apart from string values.
  json,

  /// YAML.
  yaml,

  /// TOML.
  toml,

  /// INI, `.conf`, `.properties` and `.env` style `key = value` files.
  ini,

  /// XML and its dialects, SVG among them.
  xml,

  /// HTML, lexed as [xml].
  html,

  /// CSS, SCSS and Less.
  css,

  /// JavaScript and JSX.
  javascript,

  /// TypeScript and TSX.
  typescript,

  /// Go.
  go,

  /// C, C++ and Objective-C.
  c,

  /// C#.
  csharp,

  /// Rust.
  rust,

  /// Java.
  java,

  /// Kotlin.
  kotlin,

  /// Python.
  python,

  /// Ruby.
  ruby,

  /// PHP.
  php,

  /// Lua.
  lua,

  /// POSIX shell, Bash, Zsh and Fish.
  shell,

  /// Swift.
  swift,

  /// Dart.
  dart,

  /// SQL.
  sql,

  /// A Dockerfile.
  dockerfile,

  /// A Makefile.
  makefile,

  /// A log file: levels and numbers are colored.
  log,
}

const Map<String, CodeLanguage> _byExtension = {
  'json': CodeLanguage.json,
  'jsonc': CodeLanguage.json,
  'json5': CodeLanguage.json,
  'geojson': CodeLanguage.json,
  'webmanifest': CodeLanguage.json,
  'yaml': CodeLanguage.yaml,
  'yml': CodeLanguage.yaml,
  'toml': CodeLanguage.toml,
  'ini': CodeLanguage.ini,
  'cfg': CodeLanguage.ini,
  'conf': CodeLanguage.ini,
  'properties': CodeLanguage.ini,
  'env': CodeLanguage.ini,
  'editorconfig': CodeLanguage.ini,
  'xml': CodeLanguage.xml,
  'svg': CodeLanguage.xml,
  'plist': CodeLanguage.xml,
  'xsd': CodeLanguage.xml,
  'xsl': CodeLanguage.xml,
  'html': CodeLanguage.html,
  'htm': CodeLanguage.html,
  'xhtml': CodeLanguage.html,
  'css': CodeLanguage.css,
  'scss': CodeLanguage.css,
  'less': CodeLanguage.css,
  'js': CodeLanguage.javascript,
  'mjs': CodeLanguage.javascript,
  'cjs': CodeLanguage.javascript,
  'jsx': CodeLanguage.javascript,
  'ts': CodeLanguage.typescript,
  'tsx': CodeLanguage.typescript,
  'mts': CodeLanguage.typescript,
  'cts': CodeLanguage.typescript,
  'go': CodeLanguage.go,
  'c': CodeLanguage.c,
  'h': CodeLanguage.c,
  'cc': CodeLanguage.c,
  'cpp': CodeLanguage.c,
  'cxx': CodeLanguage.c,
  'hpp': CodeLanguage.c,
  'hh': CodeLanguage.c,
  'm': CodeLanguage.c,
  'cs': CodeLanguage.csharp,
  'rs': CodeLanguage.rust,
  'java': CodeLanguage.java,
  'kt': CodeLanguage.kotlin,
  'kts': CodeLanguage.kotlin,
  'py': CodeLanguage.python,
  'pyw': CodeLanguage.python,
  'pyi': CodeLanguage.python,
  'rb': CodeLanguage.ruby,
  'gemspec': CodeLanguage.ruby,
  'php': CodeLanguage.php,
  'lua': CodeLanguage.lua,
  'sh': CodeLanguage.shell,
  'bash': CodeLanguage.shell,
  'zsh': CodeLanguage.shell,
  'fish': CodeLanguage.shell,
  'ksh': CodeLanguage.shell,
  'swift': CodeLanguage.swift,
  'dart': CodeLanguage.dart,
  'sql': CodeLanguage.sql,
  'dockerfile': CodeLanguage.dockerfile,
  'mk': CodeLanguage.makefile,
  'mak': CodeLanguage.makefile,
  'log': CodeLanguage.log,
};

const Map<String, CodeLanguage> _byBaseName = {
  'dockerfile': CodeLanguage.dockerfile,
  'containerfile': CodeLanguage.dockerfile,
  'makefile': CodeLanguage.makefile,
  'gnumakefile': CodeLanguage.makefile,
  'gemfile': CodeLanguage.ruby,
  'rakefile': CodeLanguage.ruby,
  '.bashrc': CodeLanguage.shell,
  '.zshrc': CodeLanguage.shell,
  '.profile': CodeLanguage.shell,
  '.bash_profile': CodeLanguage.shell,
};

const Map<String, CodeLanguage> _byInterpreter = {
  'sh': CodeLanguage.shell,
  'bash': CodeLanguage.shell,
  'zsh': CodeLanguage.shell,
  'dash': CodeLanguage.shell,
  'ksh': CodeLanguage.shell,
  'fish': CodeLanguage.shell,
  'python': CodeLanguage.python,
  'node': CodeLanguage.javascript,
  'nodejs': CodeLanguage.javascript,
  'deno': CodeLanguage.javascript,
  'bun': CodeLanguage.javascript,
  'ruby': CodeLanguage.ruby,
  'php': CodeLanguage.php,
  'lua': CodeLanguage.lua,
  'swift': CodeLanguage.swift,
  'dart': CodeLanguage.dart,
};

/// The interpreter path on a `#!` line, and the first argument that is not a
/// flag, which is the interpreter when the path is `env`.
final RegExp _shebang = RegExp(r'^#!\s*(\S+)((?:\s+-\S+)*)(?:\s+(\S+))?');

/// The language to highlight the file called [fileName] in.
///
/// Decided by extension (case-insensitive), then by a well-known base name
/// (`Dockerfile`, `Dockerfile.dev`, `Makefile`), then by a `#!` shebang on
/// [firstLine] (`#!/usr/bin/env python3`, `#!/bin/bash`). Anything else is
/// [CodeLanguage.plain]. [fileName] may be a whole path.
CodeLanguage codeLanguageFor(String fileName, {String firstLine = ''}) {
  final name = fileName.split('/').last.toLowerCase();
  final dot = name.lastIndexOf('.');
  final byName =
      (dot >= 0 ? _byExtension[name.substring(dot + 1)] : null) ??
      _byBaseName[name] ??
      (name.startsWith('dockerfile.') ? CodeLanguage.dockerfile : null);
  if (byName != null) return byName;

  final match = _shebang.firstMatch(firstLine);
  if (match == null) return CodeLanguage.plain;
  final program = match.group(1)!.split('/').last;
  final interpreter = program == 'env' ? (match.group(3) ?? '') : program;
  // `python3.12` and `lua5.4` are still Python and Lua.
  final bare = interpreter.replaceFirst(RegExp(r'[\d.]+$'), '');
  return _byInterpreter[bare] ?? CodeLanguage.plain;
}
