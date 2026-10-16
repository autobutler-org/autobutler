import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:quark/widgets/photos/photos_search_bar.dart';

void main() {
  final field = find.byKey(const ValueKey('photos_search_field'));

  Future<void> pumpBar(
    WidgetTester tester, {
    required bool open,
    FocusNode? holder,
    List<String>? typed,
  }) => tester.pumpWidget(
    MaterialApp(
      home: Scaffold(
        body: Focus(
          focusNode: holder,
          autofocus: true,
          child: Column(
            children: [
              if (open)
                PhotosSearchBar(
                  onChanged: (t) => typed?.add(t),
                  onClose: () {},
                ),
            ],
          ),
        ),
      ),
    ),
  );

  testWidgets('the field asks for the focus as it appears', (tester) async {
    await pumpBar(tester, open: true);
    expect(tester.widget<TextField>(field).autofocus, isTrue);
  });

  // #3363: autofocus does nothing while something else in the scope has the
  // focus, which the Photos page always does.
  testWidgets('the field takes the focus from whatever held it', (
    tester,
  ) async {
    final holder = FocusNode();
    addTearDown(holder.dispose);
    final typed = <String>[];
    await pumpBar(tester, open: false, holder: holder, typed: typed);
    await tester.pump();
    expect(holder.hasPrimaryFocus, isTrue);

    await pumpBar(tester, open: true, holder: holder, typed: typed);
    await tester.pump();

    final editable = tester.widget<EditableText>(
      find.descendant(of: field, matching: find.byType(EditableText)),
    );
    expect(editable.focusNode.hasFocus, isTrue);

    tester.testTextInput.updateEditingValue(
      const TextEditingValue(
        text: 'mp4',
        selection: TextSelection.collapsed(offset: 3),
      ),
    );
    await tester.pump();
    expect(typed, ['mp4']);
  });
}
