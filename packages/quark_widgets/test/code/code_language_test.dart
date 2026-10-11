import 'package:flutter_test/flutter_test.dart';
import 'package:quark_widgets/quark_widgets.dart';

/// `codeLanguageFor` picks the language a file is highlighted in from its
/// name, and from a shebang when the name says nothing.
void main() {
  test('goes by extension, in any case', () {
    expect(codeLanguageFor('main.dart'), CodeLanguage.dart);
    expect(codeLanguageFor('config.JSON'), CodeLanguage.json);
    expect(codeLanguageFor('deploy.yml'), CodeLanguage.yaml);
    expect(codeLanguageFor('server.go'), CodeLanguage.go);
    expect(codeLanguageFor('app.tsx'), CodeLanguage.typescript);
    expect(codeLanguageFor('lib.rs'), CodeLanguage.rust);
    expect(codeLanguageFor('quark.log'), CodeLanguage.log);
    expect(codeLanguageFor('/srv/files/notes/setup.sh'), CodeLanguage.shell);
  });

  test('goes by a well-known base name when there is no extension', () {
    expect(codeLanguageFor('Dockerfile'), CodeLanguage.dockerfile);
    expect(codeLanguageFor('Dockerfile.dev'), CodeLanguage.dockerfile);
    expect(codeLanguageFor('Makefile'), CodeLanguage.makefile);
    expect(codeLanguageFor('makefile'), CodeLanguage.makefile);
    expect(codeLanguageFor('.bashrc'), CodeLanguage.shell);
    expect(codeLanguageFor('.env'), CodeLanguage.ini);
  });

  test('falls back to the shebang', () {
    expect(
      codeLanguageFor('run', firstLine: '#!/usr/bin/env python3'),
      CodeLanguage.python,
    );
    expect(
      codeLanguageFor('run', firstLine: '#!/bin/bash -e'),
      CodeLanguage.shell,
    );
    expect(
      codeLanguageFor('run', firstLine: '#!/usr/bin/env -S node --harmony'),
      CodeLanguage.javascript,
    );
    expect(
      codeLanguageFor('run', firstLine: '#! /usr/bin/ruby'),
      CodeLanguage.ruby,
    );
    expect(
      codeLanguageFor('run', firstLine: '#!/usr/bin/perl'),
      CodeLanguage.plain,
    );
  });

  test('the extension wins over the shebang', () {
    expect(
      codeLanguageFor('tool.rb', firstLine: '#!/bin/sh'),
      CodeLanguage.ruby,
    );
  });

  test('anything unknown is plain', () {
    expect(codeLanguageFor('README'), CodeLanguage.plain);
    expect(codeLanguageFor('notes.txt'), CodeLanguage.plain);
    expect(codeLanguageFor(''), CodeLanguage.plain);
    expect(codeLanguageFor('run', firstLine: '#!'), CodeLanguage.plain);
    expect(codeLanguageFor('run', firstLine: 'python'), CodeLanguage.plain);
  });
}
