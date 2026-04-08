# flutter_tanstack_table

A minimal Flutter web wrapper that exposes a simple vanilla JS table renderer (TanStack-style API surface) to Flutter.

Usage (in your Flutter web app):

1. In web/index.html add:

   <script src="/packages/flutter_tanstack_table/tanstack_table_glue.js"></script>

2. Use the widget in Flutter:

```dart
import 'package:flutter_tanstack_table/tanstack_table.dart';

final columns = [
  {'accessor': 'id', 'header': 'ID'},
  {'accessor': 'name', 'header': 'Name'},
];
final data = [
  {'id': 1, 'name': 'Alice'},
  {'id': 2, 'name': 'Bob'},
];

TanstackTable(
  columns: columns,
  data: data,
  height: 400,
);
```

Notes:

- This is a lightweight starting point. The JS glue currently renders a basic HTML table and exposes createTable/updateTable. Later integration with @tanstack/table-core can replace the renderer while keeping the Dart API stable.
