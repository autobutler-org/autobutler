import 'package:flutter/material.dart';
import 'package:quark_widgets/quark_widgets.dart';

import 'docs.g.dart';
import 'registry.dart';
import 'token_fields.dart';

/// How many callback events the bottom panel keeps.
const int _eventLimit = 20;

/// Runs the gallery.
void main() {
  runApp(const WidgetGalleryApp());
}

/// The gallery: every widget in `quark_widgets` rendered with fake data, next
/// to its documentation, over a theme you can edit while it runs.
class WidgetGalleryApp extends StatefulWidget {
  /// Creates the gallery app.
  const WidgetGalleryApp({super.key});

  @override
  State<WidgetGalleryApp> createState() => _WidgetGalleryAppState();
}

class _WidgetGalleryAppState extends State<WidgetGalleryApp> {
  Brightness _brightness = Brightness.dark;
  QuarkTokens _tokens = QuarkTokens.dark;
  GalleryEntry _selected = registry.first;
  String _filter = '';
  final List<String> _events = [];

  void _log(String event) {
    setState(() {
      _events.insert(0, event);
      if (_events.length > _eventLimit) _events.removeLast();
    });
  }

  void _toggleBrightness() {
    setState(() {
      _brightness = _brightness == Brightness.dark
          ? Brightness.light
          : Brightness.dark;
      // Start each side from its shipped token set rather than carrying dark
      // colors into the light theme.
      _tokens = _brightness == Brightness.dark
          ? QuarkTokens.dark
          : QuarkTokens.light;
    });
  }

  @override
  Widget build(BuildContext context) {
    return MaterialApp(
      title: 'QuarkWidgets Gallery',
      debugShowCheckedModeBanner: false,
      theme: QuarkTheme.from(_tokens, _brightness),
      home: Builder(builder: _buildHome),
    );
  }

  Widget _buildHome(BuildContext context) {
    final tokens = QuarkTokens.of(context);

    return Scaffold(
      appBar: AppBar(
        title: const Text('QuarkWidgets Gallery'),
        actions: [
          Tooltip(
            message: 'Toggle light and dark',
            child: IconButton(
              icon: Icon(
                _brightness == Brightness.dark
                    ? Icons.light_mode
                    : Icons.dark_mode,
              ),
              onPressed: _toggleBrightness,
            ),
          ),
          SizedBox(width: tokens.spacingSm),
        ],
      ),
      body: Column(
        children: [
          Expanded(
            child: LayoutBuilder(
              builder: (context, constraints) {
                // Panels take a share of the width rather than a fixed size, so
                // the example area can never be squeezed to a negative width.
                final panel = (constraints.maxWidth * 0.28).clamp(0.0, 300.0);
                return Row(
                  children: [
                    SizedBox(width: panel, child: _buildIndex(context)),
                    const VerticalDivider(width: 1),
                    Expanded(child: _buildExample(context)),
                    const VerticalDivider(width: 1),
                    SizedBox(width: panel, child: _buildThemePanel(context)),
                  ],
                );
              },
            ),
          ),
          const Divider(height: 1),
          _buildEvents(context),
        ],
      ),
    );
  }

  Widget _buildIndex(BuildContext context) {
    final tokens = QuarkTokens.of(context);
    final matches = registry
        .where(
          (e) =>
              _filter.isEmpty ||
              e.name.toLowerCase().contains(_filter) ||
              e.group.toLowerCase().contains(_filter),
        )
        .toList();

    final groups = <String, List<GalleryEntry>>{};
    for (final entry in matches) {
      groups.putIfAbsent(entry.group, () => []).add(entry);
    }
    final groupNames = groups.keys.toList()..sort();

    return Column(
      crossAxisAlignment: CrossAxisAlignment.stretch,
      children: [
        Padding(
          padding: EdgeInsets.all(tokens.spacingSm),
          child: TextField(
            decoration: const InputDecoration(
              prefixIcon: Icon(Icons.search),
              hintText: 'Filter widgets',
              isDense: true,
            ),
            onChanged: (value) =>
                setState(() => _filter = value.trim().toLowerCase()),
          ),
        ),
        Expanded(
          child: matches.isEmpty
              ? Center(
                  child: Text(
                    'Nothing matches "$_filter"',
                    style: TextStyle(color: tokens.mutedForeground),
                  ),
                )
              : ListView(
                  children: [
                    for (final group in groupNames) ...[
                      Padding(
                        padding: EdgeInsets.fromLTRB(
                          tokens.spacingMd,
                          tokens.spacingMd,
                          tokens.spacingMd,
                          tokens.spacingXs,
                        ),
                        child: Text(
                          group.toUpperCase(),
                          style: Theme.of(context).textTheme.labelSmall
                              ?.copyWith(
                                color: tokens.primary,
                                letterSpacing: 0.8,
                              ),
                        ),
                      ),
                      for (final entry in groups[group]!)
                        ListTile(
                          key: ValueKey('gallery_entry_${entry.name}'),
                          dense: true,
                          selected: entry.name == _selected.name,
                          selectedTileColor: tokens.primary.withValues(
                            alpha: 0.12,
                          ),
                          title: Text(entry.name),
                          onTap: () => setState(() => _selected = entry),
                        ),
                    ],
                  ],
                ),
        ),
      ],
    );
  }

  Widget _buildExample(BuildContext context) {
    final tokens = QuarkTokens.of(context);
    final docs = widgetDocs[_selected.name];

    return SingleChildScrollView(
      padding: EdgeInsets.all(tokens.spacingLg),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          Text(_selected.name, style: Theme.of(context).textTheme.titleLarge),
          SizedBox(height: tokens.spacingLg),
          _selected.build(context, _log),
          SizedBox(height: tokens.spacingLg),
          const Divider(),
          SizedBox(height: tokens.spacingMd),
          Text('Documentation', style: Theme.of(context).textTheme.titleSmall),
          SizedBox(height: tokens.spacingSm),
          SelectableText(
            docs ??
                'No class documentation found. Add a /// block to the class and '
                    'run `make -C packages/quark_widgets generate/docs`.',
            style: TextStyle(
              fontFamily: 'monospace',
              fontSize: 12,
              height: 1.5,
              color: docs == null
                  ? tokens.mutedForeground
                  : tokens.secondaryForeground,
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildThemePanel(BuildContext context) {
    final tokens = QuarkTokens.of(context);

    return ListView(
      padding: EdgeInsets.all(tokens.spacingMd),
      children: [
        Text('Theme', style: Theme.of(context).textTheme.titleSmall),
        SizedBox(height: tokens.spacingSm),
        SwitchListTile(
          contentPadding: EdgeInsets.zero,
          dense: true,
          title: const Text('Dark'),
          value: _brightness == Brightness.dark,
          onChanged: (_) => _toggleBrightness(),
        ),
        SizedBox(height: tokens.spacingSm),
        for (final field in colorFields)
          _HexField(
            // Rebuild the controllers when the token set is swapped wholesale.
            key: ValueKey('${field.name}-$_brightness'),
            name: field.name,
            value: field.read(_tokens),
            onSubmitted: (color) =>
                setState(() => _tokens = field.write(_tokens, color)),
          ),
        SizedBox(height: tokens.spacingMd),
        for (final field in numberFields)
          _NumberField(
            name: field.name,
            value: field.read(_tokens),
            max: field.max,
            onChanged: (value) =>
                setState(() => _tokens = field.write(_tokens, value)),
          ),
      ],
    );
  }

  Widget _buildEvents(BuildContext context) {
    final tokens = QuarkTokens.of(context);

    return SizedBox(
      height: 148,
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Padding(
            padding: EdgeInsets.fromLTRB(
              tokens.spacingMd,
              tokens.spacingSm,
              tokens.spacingSm,
              0,
            ),
            child: Row(
              children: [
                Text(
                  'Events (${_events.length})',
                  style: Theme.of(context).textTheme.titleSmall,
                ),
                const Spacer(),
                TextButton(
                  onPressed: _events.isEmpty
                      ? null
                      : () => setState(_events.clear),
                  child: const Text('Clear'),
                ),
              ],
            ),
          ),
          Expanded(
            child: _events.isEmpty
                ? Padding(
                    padding: EdgeInsets.symmetric(horizontal: tokens.spacingMd),
                    child: Text(
                      'Callbacks from the example land here, newest first.',
                      style: TextStyle(
                        color: tokens.mutedForeground,
                        fontSize: 12,
                      ),
                    ),
                  )
                : ListView.builder(
                    padding: EdgeInsets.symmetric(horizontal: tokens.spacingMd),
                    itemCount: _events.length,
                    itemBuilder: (context, index) => Text(
                      _events[index],
                      style: TextStyle(
                        fontFamily: 'monospace',
                        fontSize: 12,
                        color: tokens.secondaryForeground,
                      ),
                    ),
                  ),
          ),
        ],
      ),
    );
  }
}

/// A hex entry for one color token, applied when the field is submitted.
class _HexField extends StatefulWidget {
  const _HexField({
    super.key,
    required this.name,
    required this.value,
    required this.onSubmitted,
  });

  final String name;
  final Color value;
  final ValueChanged<Color> onSubmitted;

  @override
  State<_HexField> createState() => _HexFieldState();
}

class _HexFieldState extends State<_HexField> {
  late final TextEditingController _controller = TextEditingController(
    text: toHex(widget.value),
  );

  @override
  void dispose() {
    _controller.dispose();
    super.dispose();
  }

  void _submit(String text) {
    final color = parseHex(text);
    if (color == null) {
      _controller.text = toHex(widget.value);
      return;
    }
    widget.onSubmitted(color);
  }

  @override
  Widget build(BuildContext context) {
    final tokens = QuarkTokens.of(context);

    return Padding(
      padding: EdgeInsets.only(bottom: tokens.spacingSm),
      child: TextField(
        controller: _controller,
        onSubmitted: _submit,
        style: const TextStyle(fontFamily: 'monospace', fontSize: 12),
        decoration: InputDecoration(
          isDense: true,
          labelText: widget.name,
          prefixIcon: Padding(
            padding: EdgeInsets.all(tokens.spacingSm),
            // The prefix slot hands down loose constraints, so the swatch needs
            // an explicit height or it collapses to its 2px border and the
            // color it is meant to show is invisible.
            child: Container(
              width: 16,
              height: 16,
              decoration: BoxDecoration(
                color: widget.value,
                border: Border.all(color: tokens.border),
                borderRadius: BorderRadius.circular(tokens.radiusSm),
              ),
            ),
          ),
          prefixIconConstraints: const BoxConstraints(
            minWidth: 36,
            minHeight: 16,
          ),
        ),
      ),
    );
  }
}

/// A slider for one radius or spacing token, applied as it moves.
class _NumberField extends StatelessWidget {
  const _NumberField({
    required this.name,
    required this.value,
    required this.max,
    required this.onChanged,
  });

  final String name;
  final double value;
  final double max;
  final ValueChanged<double> onChanged;

  @override
  Widget build(BuildContext context) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(
          '$name  ${value.toStringAsFixed(0)}',
          style: const TextStyle(fontFamily: 'monospace', fontSize: 12),
        ),
        Slider(
          value: value.clamp(0, max),
          max: max,
          divisions: max.round(),
          onChanged: onChanged,
        ),
      ],
    );
  }
}
