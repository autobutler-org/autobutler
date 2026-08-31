import 'dart:convert';
import 'dart:typed_data';

import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:quark/pages/image_viewer_page.dart';
import 'package:shared_preferences/shared_preferences.dart';

/// #1707: swiping the photo left/right did nothing. [InteractiveViewer]'s own
/// scale recognizer wins the gesture arena against a drag detector wrapped
/// around it, so the swipe has to be read from raw pointer events.
void main() {
  // 64x64 solid PNG — the photo needs a real box for a gesture to land on.
  final bytes = Uint8List.fromList(
    base64Decode(
      'iVBORw0KGgoAAAANSUhEUgAAAEAAAABACAIAAAAlC+aJAAAAT0lEQVR42u3P'
      'QQkAAAgEsIttCIMZywi+hcEKLNXzWgQEBAQEBAQEBAQEBAQEBAQEBAQEBAQE'
      'BAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQELgvWNcGlSbHPawAAAABJRU5ErkJg'
      'gg==',
    ),
  );

  setUp(() => SharedPreferences.setMockInitialValues({}));

  /// Pumps the viewer on image 1 of 3 and returns the indexes it asks to load.
  Future<List<int>> pumpViewer(WidgetTester tester) async {
    final requested = <int>[];
    final page = MaterialApp(
      home: ImageViewerPage(
        bytes: bytes,
        name: 'first.jpg',
        initialIndex: 1,
        imageCount: 3,
        onLoadImage: (index) async {
          requested.add(index);
          return (bytes, 'next.jpg', null, null);
        },
      ),
    );
    // Decoding the image takes real async, and it has to be laid out before a
    // gesture can hit it.
    await tester.runAsync(() async {
      await tester.pumpWidget(page);
      await Future<void>.delayed(const Duration(milliseconds: 200));
      await tester.pump();
    });
    await tester.pumpAndSettle();
    return requested;
  }

  testWidgets('swiping left loads the next image', (tester) async {
    final requested = await pumpViewer(tester);
    await tester.fling(
      find.byType(InteractiveViewer),
      const Offset(-300, 0),
      1200,
    );
    await tester.pumpAndSettle();
    expect(requested, [2]);
  });

  testWidgets('swiping right loads the previous image', (tester) async {
    final requested = await pumpViewer(tester);
    await tester.fling(
      find.byType(InteractiveViewer),
      const Offset(300, 0),
      1200,
    );
    await tester.pumpAndSettle();
    expect(requested, [0]);
  });

  // The regression itself: a real finger drifts vertically before it travels
  // sideways, which is enough for InteractiveViewer to claim the gesture.
  testWidgets('a swipe that starts with vertical drift still navigates', (
    tester,
  ) async {
    final requested = await pumpViewer(tester);
    var elapsed = Duration.zero;
    Duration tick() => elapsed += const Duration(milliseconds: 16);

    final gesture = await tester.startGesture(
      tester.getCenter(find.byType(InteractiveViewer)),
    );
    await gesture.moveBy(const Offset(-8, -34), timeStamp: tick());
    await tester.pump(const Duration(milliseconds: 16));
    await gesture.moveBy(const Offset(-60, -6), timeStamp: tick());
    await tester.pump(const Duration(milliseconds: 16));
    await gesture.moveBy(const Offset(-60, 0), timeStamp: tick());
    await tester.pump(const Duration(milliseconds: 16));
    await gesture.up(timeStamp: tick());
    await tester.pumpAndSettle();

    expect(requested, [2]);
  });

  testWidgets('a tap does not navigate', (tester) async {
    final requested = await pumpViewer(tester);
    await tester.tap(find.byType(InteractiveViewer));
    await tester.pumpAndSettle();
    expect(requested, isEmpty);
  });
}
