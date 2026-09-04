import 'package:flutter/material.dart';

import '../theme/quark_tokens.dart';
import 'quark_toolbar.dart';

/// A titled block of a page: a heading, whatever actions belong to it, and the
/// content under both.
///
/// Every page grew its own version of this out of a bold `Text`, a `SizedBox`,
/// and a `Row` — settings alone had eight, in two different sizes. The heading
/// style, the gap under it, and what happens to the actions on a narrow phone
/// are decided here instead.
///
/// The actions go through a [QuarkToolbar], so a heading with three buttons
/// wraps them onto a second line rather than overflowing at 360 pixels.
///
/// Key prefixes: `section_<title>`, with the title lowercased and every run of
/// non-alphanumeric characters turned into a single underscore, so
/// `QuarkSection(title: 'Help & Support')` is `#section_help_support`. The
/// actions carry their own keys.
///
/// ```dart
/// QuarkSection(
///   title: 'Backend hosts',
///   actions: [IconButton(icon: const Icon(Icons.add), onPressed: addHost)],
///   child: HostList(hosts: hosts),
/// );
/// ```
class QuarkSection extends StatelessWidget {
  /// Creates a section headed [title] around [child].
  const QuarkSection({
    required this.title,
    required this.child,
    this.icon,
    this.actions = const [],
    super.key,
  });

  /// The heading. Also the source of the section's key.
  final String title;

  /// The section's content, laid out under the heading.
  final Widget child;

  /// A glyph before the heading, for a section that is reference material
  /// rather than something the reader configures. Null renders no glyph.
  final IconData? icon;

  /// Controls belonging to the section, rendered after the heading.
  final List<Widget> actions;

  /// The key suffix [title] produces, exposed so a test or a `.probe` script
  /// can build the same key without guessing at the rules.
  static String slug(String title) => title
      .toLowerCase()
      .replaceAll(RegExp(r'[^a-z0-9]+'), '_')
      .replaceAll(RegExp(r'^_+|_+$'), '');

  @override
  Widget build(BuildContext context) {
    final theme = Theme.of(context);
    final tokens = QuarkTokens.of(context);

    return Column(
      key: ValueKey('section_${slug(title)}'),
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Row(
          children: [
            if (icon != null) ...[
              Icon(icon, size: 16, color: tokens.secondaryForeground),
              SizedBox(width: tokens.spacingXs + tokens.spacingXs),
            ],
            Expanded(
              child: Text(
                title,
                overflow: TextOverflow.ellipsis,
                style: theme.textTheme.titleMedium?.copyWith(
                  fontWeight: FontWeight.w600,
                ),
              ),
            ),
            if (actions.isNotEmpty)
              Flexible(child: QuarkToolbar(actions: actions)),
          ],
        ),
        SizedBox(height: tokens.spacingSm),
        child,
      ],
    );
  }
}
