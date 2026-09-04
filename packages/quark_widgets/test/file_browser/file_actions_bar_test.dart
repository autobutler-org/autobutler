import 'package:flutter/material.dart';
import 'package:flutter_test/flutter_test.dart';
import 'package:quark_widgets/quark_widgets.dart';

import '../support/pump.dart';

void main() {
  testBothViewports('offers upload and new folder when idle', (
    tester,
    size,
  ) async {
    final events = <String>[];
    await pumpAt(
      tester,
      FileActionsBar(
        isUploading: false,
        isCreatingFolder: false,
        isSearchMode: false,
        onUploadPressed: () => events.add('upload'),
        onCreateFolderPressed: () => events.add('newFolder'),
      ),
      size: size,
    );

    expect(find.text('Upload'), findsOneWidget);
    expect(find.text('New Folder'), findsOneWidget);

    await tester.tap(find.byKey(const ValueKey('file_actions_upload')));
    await tester.tap(find.byKey(const ValueKey('file_actions_new_folder')));
    await tester.pump();

    expect(events, ['upload', 'newFolder']);
  });

  testBothViewports('counts a multi-file upload while it runs', (
    tester,
    size,
  ) async {
    await pumpAt(
      tester,
      FileActionsBar(
        isUploading: true,
        isCreatingFolder: false,
        isSearchMode: false,
        uploadTotal: 5,
        uploadCompleted: 2,
        onUploadPressed: () {},
        onCreateFolderPressed: () {},
      ),
      size: size,
    );

    expect(find.text('Uploading 2 of 5...'), findsOneWidget);
    final progress = tester.widget<CircularProgressIndicator>(
      find.byType(CircularProgressIndicator),
    );
    expect(progress.value, 2 / 5);
  });

  testWidgets('shows an indeterminate spinner for a single upload', (
    tester,
  ) async {
    await pumpAt(
      tester,
      FileActionsBar(
        isUploading: true,
        isCreatingFolder: false,
        isSearchMode: false,
        uploadTotal: 1,
        onUploadPressed: () {},
        onCreateFolderPressed: () {},
      ),
      size: narrowViewport,
    );

    expect(find.text('Uploading...'), findsOneWidget);
    final progress = tester.widget<CircularProgressIndicator>(
      find.byType(CircularProgressIndicator),
    );
    expect(progress.value, isNull);
  });

  testWidgets('says so and disables itself while creating a folder', (
    tester,
  ) async {
    var taps = 0;
    await pumpAt(
      tester,
      FileActionsBar(
        isUploading: false,
        isCreatingFolder: true,
        isSearchMode: false,
        onUploadPressed: () {},
        onCreateFolderPressed: () => taps++,
      ),
      size: wideViewport,
    );

    expect(find.text('Creating...'), findsOneWidget);
    final button = tester.widget<OutlinedButton>(
      find.byKey(const ValueKey('file_actions_new_folder')),
    );
    expect(button.onPressed, isNull);
    expect(taps, 0);
  });

  testBothViewports('disappears in search mode', (tester, size) async {
    await pumpAt(
      tester,
      FileActionsBar(
        isUploading: false,
        isCreatingFolder: false,
        isSearchMode: true,
        onUploadPressed: () {},
        onCreateFolderPressed: () {},
      ),
      size: size,
    );

    expect(find.text('Upload'), findsNothing);
    expect(find.text('New Folder'), findsNothing);
  });
}
