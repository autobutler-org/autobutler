import 'dart:math' as math;

import 'package:flutter/widgets.dart';

/// The slots of a `QuarkAppBar` row, as `LayoutId` ids for
/// [QuarkAppBarLayout].
enum QuarkAppBarSlot {
  /// The brand button and the refresh beside it. Every bar has one.
  brand,

  /// A page's own content between the brand and the actions.
  middle,

  /// The page's own actions, beside the app-wide controls.
  pageActions,

  /// The app-wide controls, at the end edge.
  appActions,
}

/// Lays a `QuarkAppBar` row out in order of who gives way: the brand keeps the
/// width its page name needs, up to [brandMaxWidth]; the app-wide controls get
/// what is left after it; the page's actions get what is left after those;
/// and the middle, when there is one, fills whatever
/// remains between them, [gap] from each, never less than [middleMinWidth].
/// Each action slot keeps [actionMinWidth] and scrolls sideways when it is
/// handed less than it needs.
///
/// A `Row` cannot say this. A flexible child is handed a fixed share of the
/// free space rather than what it needs, and an inflexible one is laid out
/// unbounded, which is how a phone's actions ended up painted over the brand.
///
/// [textDirection] is the bar's own direction, already mirrored for a
/// left-handed bar, so the brand sits at its start edge.
class QuarkAppBarLayout extends MultiChildLayoutDelegate {
  /// Creates the layout for a bar whose brand slot is at most [brandMaxWidth].
  QuarkAppBarLayout({
    required this.brandMaxWidth,
    required this.gap,
    required this.textDirection,
    this.middleMinWidth = 0,
    this.actionMinWidth = 0,
  });

  /// The most the brand and refresh may take.
  final double brandMaxWidth;

  /// The space on each side of the middle.
  final double gap;

  /// Which edge the brand starts from.
  final TextDirection textDirection;

  /// The width the middle keeps however many actions there are. Ignored
  /// without a middle.
  final double middleMinWidth;

  /// The width each action slot keeps, so a scrolling row always shows at
  /// least one whole action. The brand gives way to it.
  final double actionMinWidth;

  @override
  void performLayout(Size size) {
    double startOf(double offset, double width) =>
        textDirection == TextDirection.ltr
        ? offset
        : size.width - offset - width;
    void place(QuarkAppBarSlot slot, double offset, Size child) =>
        positionChild(
          slot,
          Offset(
            startOf(offset, child.width),
            (size.height - child.height) / 2,
          ),
        );
    Size loose(QuarkAppBarSlot slot, double maxWidth) => layoutChild(
      slot,
      BoxConstraints(maxWidth: math.max(0, maxWidth), maxHeight: size.height),
    );

    final hasMiddle = hasChild(QuarkAppBarSlot.middle);
    final reserved = hasMiddle ? gap * 2 + middleMinWidth : 0;
    final slots = [
      QuarkAppBarSlot.appActions,
      QuarkAppBarSlot.pageActions,
    ].where(hasChild).toList();
    final brand = loose(
      QuarkAppBarSlot.brand,
      math.min(
        brandMaxWidth,
        size.width - reserved - slots.length * actionMinWidth,
      ),
    );
    place(QuarkAppBarSlot.brand, 0, brand);
    var actionsWidth = 0.0;
    for (final (index, slot) in slots.indexed) {
      final kept = (slots.length - index - 1) * actionMinWidth;
      final actions = loose(
        slot,
        size.width - brand.width - actionsWidth - reserved - kept,
      );
      actionsWidth += actions.width;
      place(slot, size.width - actionsWidth, actions);
    }
    if (hasMiddle) {
      final width = math.max(
        0.0,
        size.width - brand.width - actionsWidth - gap * 2,
      );
      final middle = layoutChild(
        QuarkAppBarSlot.middle,
        BoxConstraints(
          minWidth: width,
          maxWidth: width,
          maxHeight: size.height,
        ),
      );
      place(QuarkAppBarSlot.middle, brand.width + gap, middle);
    }
  }

  @override
  bool shouldRelayout(QuarkAppBarLayout oldDelegate) =>
      brandMaxWidth != oldDelegate.brandMaxWidth ||
      gap != oldDelegate.gap ||
      middleMinWidth != oldDelegate.middleMinWidth ||
      actionMinWidth != oldDelegate.actionMinWidth ||
      textDirection != oldDelegate.textDirection;
}
