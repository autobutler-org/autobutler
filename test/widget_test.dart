import 'package:flutter_test/flutter_test.dart';

import 'package:autobutler/main.dart';

void main() {
  testWidgets('renders file browser screen', (WidgetTester tester) async {
    await tester.pumpWidget(const AutobutlerApp());
    await tester.pumpAndSettle();

    expect(find.text('Cirrus'), findsOneWidget);
    expect(find.text('Name'), findsOneWidget);
    expect(find.text('Device'), findsOneWidget);
    expect(find.text('Size'), findsOneWidget);
  });
}
