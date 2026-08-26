import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:quark/widgets/file_browser/file_selection_bar.dart';

// The selection bar is custom chrome, not a real AppBar, so nothing pads it
// away from the status bar for free. On a notched iPhone it drew underneath
// the clock and the wifi/battery indicators, burying "Deselect all" (#1597).
//
// These pump the bar under a MediaQuery carrying real device insets and check
// where the controls actually land, which is the part a screenshot caught and
// a widget test can keep caught.
void main() {
  // iPhone 15 Pro portrait: 59pt status bar / Dynamic Island band.
  const notchInsets = EdgeInsets.only(top: 59, bottom: 34);
  // Same device rotated: the inset moves to the sides.
  const landscapeInsets = EdgeInsets.only(left: 59, right: 59, bottom: 21);

  Future<void> pumpSelectionBar(
    WidgetTester tester, {
    EdgeInsets insets = EdgeInsets.zero,
    int selectedCount = 3,
    int totalCount = 3,
  }) {
    return tester.pumpWidget(
      MaterialApp(
        home: MediaQuery(
          data: MediaQueryData(padding: insets, viewPadding: insets),
          child: Scaffold(
            body: Column(
              children: [
                FileSelectionBar(
                  selectedCount: selectedCount,
                  totalCount: totalCount,
                  onSelectAll: () {},
                  onDeselectAll: () {},
                  onCancel: () {},
                  onDelete: () {},
                ),
              ],
            ),
          ),
        ),
      ),
    );
  }

  testWidgets('keeps every control below the top inset', (
    WidgetTester tester,
  ) async {
    await pumpSelectionBar(tester, insets: notchInsets);

    final controls = <String, Finder>{
      'the close button': find.byTooltip('Cancel selection'),
      'the count label': find.text('3 selected'),
      'Deselect all': find.text('Deselect all'),
      'the delete button': find.byTooltip('Delete selected'),
    };
    for (final entry in controls.entries) {
      expect(
        tester.getRect(entry.value).top,
        greaterThanOrEqualTo(notchInsets.top),
        reason: '${entry.key} must clear the status bar',
      );
    }
  });

  testWidgets('paints the bar through the inset region', (
    WidgetTester tester,
  ) async {
    await pumpSelectionBar(tester, insets: notchInsets);

    // The bar itself starts at y=0 — only its contents are pushed down — so
    // the status bar sits on the bar's own surface color, not on whatever is
    // scrolling underneath it.
    expect(tester.getRect(find.byType(FileSelectionBar)).top, 0);
    expect(
      tester.getRect(find.byType(FileSelectionBar)).height,
      56 + notchInsets.top,
    );
  });

  testWidgets('honors the side insets in landscape', (
    WidgetTester tester,
  ) async {
    await pumpSelectionBar(tester, insets: landscapeInsets);

    final screenWidth =
        tester.view.physicalSize.width / tester.view.devicePixelRatio;
    expect(
      tester.getRect(find.byTooltip('Cancel selection')).left,
      greaterThanOrEqualTo(landscapeInsets.left),
    );
    expect(
      tester.getRect(find.byTooltip('Delete selected')).right,
      lessThanOrEqualTo(screenWidth - landscapeInsets.right),
    );
  });

  testWidgets('adds no padding on a device without insets', (
    WidgetTester tester,
  ) async {
    await pumpSelectionBar(tester);

    expect(tester.getRect(find.byType(FileSelectionBar)).height, 56);
  });

  testWidgets('offers "Select all" until everything is selected', (
    WidgetTester tester,
  ) async {
    await pumpSelectionBar(tester, selectedCount: 1, totalCount: 3);

    expect(find.text('Select all'), findsOneWidget);
    expect(find.text('Deselect all'), findsNothing);
  });
}
