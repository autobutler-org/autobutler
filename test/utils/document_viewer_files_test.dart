import 'package:flutter_test/flutter_test.dart';
import 'package:quark/utils/document_viewer_files.dart';
import 'package:quark/utils/file_kind.dart';

/// Which files the document viewer opens, which links it follows, and where
/// "Open as Quark doc" writes (#2987).
void main() {
  test('opens .docx, .docm, .odt and .rtf, labeled Document', () {
    for (final name in ['q3.docx', 'macros.DOCM', 'letter.odt', 'notes.rtf']) {
      expect(opensInDocumentViewer(name), isTrue, reason: name);
      expect(fileKindLabel(fileKindForName(name)), 'Document', reason: name);
    }
    for (final name in ['deck.pptx', 'old.doc', 'notes.qdoc', 'readme.md']) {
      expect(opensInDocumentViewer(name), isFalse, reason: name);
    }
  });

  test('follows only web and mail links', () {
    expect(
      documentLinkUri(' https://example.com/a?b=1 '),
      Uri.parse('https://example.com/a?b=1'),
    );
    expect(documentLinkUri('http://example.com'), isNotNull);
    expect(
      documentLinkUri('mailto:someone@example.com'),
      Uri.parse('mailto:someone@example.com'),
    );
    for (final href in [
      'javascript:alert(1)',
      'file:///etc/passwd',
      'data:text/html,hi',
      '../other.docx',
      '#bookmark',
      'https://',
      'mailto:',
      'http://[',
    ]) {
      expect(documentLinkUri(href), isNull, reason: href);
    }
  });

  test('the .qdoc lands beside the document under its name', () {
    expect(qdocPathFor('work/q3.docx'), 'work/q3.qdoc');
    expect(qdocPathFor('letter.v2.odt'), 'letter.v2.qdoc');
    expect(qdocPathFor('a.b/notes'), 'a.b/notes.qdoc');
    expect(qdocPathFor('dir/.rtf'), 'dir/.rtf.qdoc');
  });
}
