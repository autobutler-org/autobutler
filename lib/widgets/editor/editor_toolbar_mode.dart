import 'package:shared_preferences/shared_preferences.dart';

/// Display mode for editor toolbars.
///
/// [top] — classic horizontal toolbar above the editor content (default).
/// [sidebar] — vertical panel docked to the left edge of the editor.
enum EditorToolbarMode { top, sidebar }

/// Persistence key for [DocumentEditorPage] toolbar mode preference.
const kPrefKeyDocumentToolbarMode = 'document_editor_toolbar_mode';

/// Persistence key for [SpreadsheetEditorPage] toolbar mode preference.
const kPrefKeySpreadsheetToolbarMode = 'spreadsheet_editor_toolbar_mode';

/// Loads the saved toolbar mode for the given [prefKey], defaulting to [EditorToolbarMode.top].
Future<EditorToolbarMode> loadToolbarMode(String prefKey) async {
  final prefs = await SharedPreferences.getInstance();
  final raw = prefs.getString(prefKey);
  return raw == 'sidebar' ? EditorToolbarMode.sidebar : EditorToolbarMode.top;
}

/// Persists [mode] under [prefKey].
Future<void> saveToolbarMode(String prefKey, EditorToolbarMode mode) async {
  final prefs = await SharedPreferences.getInstance();
  await prefs.setString(
    prefKey,
    mode == EditorToolbarMode.sidebar ? 'sidebar' : 'top',
  );
}
