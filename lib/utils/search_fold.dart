/// Folds text for case- and accent-insensitive matching, as the on-device
/// chat search does (#3678).
library;

// cspell:disable

const _folds = <String, String>{
  'àáâãäåāăą': 'a',
  'çćĉċč': 'c',
  'ďđ': 'd',
  'èéêëēĕėęě': 'e',
  'ĝğġģ': 'g',
  'ĥħ': 'h',
  'ìíîïĩīĭįı': 'i',
  'ĵ': 'j',
  'ķ': 'k',
  'ĺļľŀł': 'l',
  'ñńņňŉ': 'n',
  'òóôõöøōŏő': 'o',
  'ŕŗř': 'r',
  'śŝşš': 's',
  'ţťŧ': 't',
  'ùúûüũūŭůűų': 'u',
  'ŵ': 'w',
  'ýÿŷ': 'y',
  'źżž': 'z',
  'ß': 'ss',
  'æ': 'ae',
  'œ': 'oe',
};

final _foldMap = {
  for (final e in _folds.entries)
    for (final rune in e.key.runes) String.fromCharCode(rune): e.value,
};

/// [text] lowercased with Latin accents removed and `ß`, `æ` and `œ` spelled
/// out, so `Café`, `cafe` and `CAFÉ` fold alike, as do `Straße` and `STRASSE`.
String foldForSearch(String text) {
  final lower = text.toLowerCase();
  final out = StringBuffer();
  for (final rune in lower.runes) {
    final char = String.fromCharCode(rune);
    out.write(_foldMap[char] ?? char);
  }
  return out.toString();
}
