/// Gives an SVG root element the size `flutter_svg` needs before it will lay
/// out.
///
/// A root with no `viewBox` is sized by its `width` and `height`, and the
/// decoder only reads absolute lengths there. With neither attribute it throws;
/// with a missing one, a percentage or a unit it does not know, it reports an
/// infinite size, which fails layout rather than the decode (#2989). A browser
/// falls back to 300 by 150 in those cases, so this writes that fallback into
/// the root tag for each dimension the decoder could not use. Markup with a
/// `viewBox`, or with nothing that looks like an `<svg>` tag, comes back
/// unchanged.
String svgWithViewport(String markup) {
  // ponytail: the root tag is found by regex, so a `>` inside one of its
  // attribute values cuts it short. Parse with package:xml if that turns up.
  final tag = RegExp(r'<svg\b[^>]*>').firstMatch(markup)?.group(0);
  if (tag == null || _attribute('viewBox').hasMatch(tag)) return markup;
  final sized = _withLength(_withLength(tag, 'width', 300), 'height', 150);
  return markup.replaceFirst(tag, sized);
}

RegExp _attribute(String name) =>
    RegExp('\\s$name\\s*=\\s*("[^"]*"|\'[^\']*\')');

/// [tag] with [name] set to [fallback], unless it already holds a length the
/// decoder reads: a bare number, or one in `px`, `em` or `ex`.
String _withLength(String tag, String name, int fallback) {
  final attribute = _attribute(name).firstMatch(tag);
  final declared = ' $name="$fallback"';
  if (attribute == null) return tag.replaceFirst('<svg', '<svg$declared');
  final value = attribute.group(1)!;
  final absolute = RegExp(
    r'^\d*\.?\d+(px|em|ex)?$',
  ).hasMatch(value.substring(1, value.length - 1).trim());
  return absolute ? tag : tag.replaceFirst(attribute.group(0)!, declared);
}
