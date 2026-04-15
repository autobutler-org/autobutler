import 'package:flutter/material.dart';

import 'data_sheet_controller.dart';

/// A ready-made control bar for [DataSheetController].
///
/// Provides "Add row" and "Add column" buttons wired to the supplied
/// [controller]. Place it wherever suits your layout — typically above or
/// below a [DataSheet]:
///
/// ```dart
/// Column(
///   children: [
///     DataSheetControlBar(controller: myController),
///     Expanded(child: DataSheet(controller: myController, table: myTable)),
///   ],
/// )
/// ```
///
/// Users who want a custom control bar should build their own widget and
/// call methods on [DataSheetController] directly (e.g. [DataSheetController.addRow],
/// [DataSheetController.addColumn]).
class DataSheetControlBar extends StatelessWidget {
  final DataSheetController controller;

  const DataSheetControlBar({super.key, required this.controller});

  @override
  Widget build(BuildContext context) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 8.0),
      child: Row(
        mainAxisAlignment: MainAxisAlignment.end,
        children: [
          ElevatedButton.icon(
            onPressed: controller.addRow,
            icon: const Icon(Icons.add),
            label: const Text('Add row'),
          ),
          const SizedBox(width: 8),
          ElevatedButton.icon(
            onPressed: controller.addColumn,
            icon: const Icon(Icons.view_column),
            label: const Text('Add column'),
          ),
          const SizedBox(width: 8),
        ],
      ),
    );
  }
}
