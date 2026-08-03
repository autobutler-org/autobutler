import 'package:flutter/material.dart';
import 'package:flutter_quill/flutter_quill.dart';

/// Toolbar placement: top bar or collapsible sidebar.
enum EditorToolbarMode { top, sidebar }

/// A collapsible sidebar that wraps [QuillSimpleToolbar] and renders
/// tool buttons vertically with optional icon + label layout.
///
/// On small screens (< 600 px wide) the sidebar collapses to icon-only
/// regardless of [expanded] state.
class EditorSidebarToolbar extends StatefulWidget {
  const EditorSidebarToolbar({
    super.key,
    required this.controller,
    required this.toolbarTheme,
    required this.config,
    this.initiallyExpanded = true,
  });

  final QuillController controller;
  final ThemeData toolbarTheme;
  final QuillSimpleToolbarConfig config;
  final bool initiallyExpanded;

  @override
  State<EditorSidebarToolbar> createState() => _EditorSidebarToolbarState();
}

class _EditorSidebarToolbarState extends State<EditorSidebarToolbar>
    with SingleTickerProviderStateMixin {
  late bool _expanded;
  late AnimationController _animCtrl;
  late Animation<double> _widthAnim;

  static const _expandedWidth = 200.0;
  static const _collapsedWidth = 52.0;
  static const _iconOnlyWidth = 52.0;

  @override
  void initState() {
    super.initState();
    _expanded = widget.initiallyExpanded;
    _animCtrl = AnimationController(
      vsync: this,
      duration: const Duration(milliseconds: 200),
      value: _expanded ? 1.0 : 0.0,
    );
    _widthAnim = Tween<double>(
      begin: _collapsedWidth,
      end: _expandedWidth,
    ).animate(CurvedAnimation(parent: _animCtrl, curve: Curves.easeInOut));
  }

  @override
  void dispose() {
    _animCtrl.dispose();
    super.dispose();
  }

  void _toggle() {
    setState(() => _expanded = !_expanded);
    if (_expanded) {
      _animCtrl.forward();
    } else {
      _animCtrl.reverse();
    }
  }

  bool _isNarrow(BuildContext context) =>
      MediaQuery.sizeOf(context).width < 600;

  @override
  Widget build(BuildContext context) {
    final cs = widget.toolbarTheme.colorScheme;
    final narrow = _isNarrow(context);
    final showLabels = _expanded && !narrow;

    return AnimatedBuilder(
      animation: _widthAnim,
      builder: (context, child) {
        final w = narrow ? _iconOnlyWidth : _widthAnim.value;
        return Container(
          width: w,
          decoration: BoxDecoration(
            color: cs.surfaceContainer,
            border: Border(right: BorderSide(color: cs.outline)),
          ),
          child: child,
        );
      },
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          // Expand / collapse toggle
          if (!narrow)
            _ToggleButton(
              expanded: _expanded,
              color: cs.onSurface,
              onTap: _toggle,
            ),
          Expanded(
            child: Theme(
              data: widget.toolbarTheme,
              child: _SidebarToolbarContent(
                controller: widget.controller,
                config: widget.config,
                showLabels: showLabels,
                cs: cs,
              ),
            ),
          ),
        ],
      ),
    );
  }
}

class _ToggleButton extends StatelessWidget {
  const _ToggleButton({
    required this.expanded,
    required this.color,
    required this.onTap,
  });

  final bool expanded;
  final Color color;
  final VoidCallback onTap;

  @override
  Widget build(BuildContext context) {
    return InkWell(
      onTap: onTap,
      child: Padding(
        padding: const EdgeInsets.symmetric(vertical: 6),
        child: Icon(
          expanded ? Icons.chevron_left : Icons.chevron_right,
          color: color,
          size: 20,
        ),
      ),
    );
  }
}

/// Renders a [QuillSimpleToolbar] rotated and re-flowed vertically inside
/// the sidebar. Because QuillSimpleToolbar is a horizontal Wrap widget, we
/// wrap it in a `RotatedBox` + constrain its width so it flows into a
/// single column of buttons.
class _SidebarToolbarContent extends StatelessWidget {
  const _SidebarToolbarContent({
    required this.controller,
    required this.config,
    required this.showLabels,
    required this.cs,
  });

  final QuillController controller;
  final QuillSimpleToolbarConfig config;
  final bool showLabels;
  final ColorScheme cs;

  @override
  Widget build(BuildContext context) {
    // Rebuild the toolbar config with vertical alignment.
    final verticalConfig = config.copyWith(
      toolbarIconAlignment: WrapAlignment.start,
      toolbarSectionSpacing: 4,
    );

    return SingleChildScrollView(
      child: QuillSimpleToolbar(controller: controller, config: verticalConfig),
    );
  }
}

/// A widget that renders either a top-bar toolbar or a sidebar toolbar
/// depending on [mode], and wraps the [child] (editor body) accordingly.
class EditorToolbarLayout extends StatelessWidget {
  const EditorToolbarLayout({
    super.key,
    required this.mode,
    required this.toolbar,
    required this.sidebarToolbar,
    required this.child,
  });

  /// The top-bar toolbar widget (used in [EditorToolbarMode.top] mode).
  final Widget toolbar;

  /// The sidebar toolbar widget (used in [EditorToolbarMode.sidebar] mode).
  final Widget sidebarToolbar;

  /// The editor body.
  final Widget child;

  /// Current toolbar mode.
  final EditorToolbarMode mode;

  @override
  Widget build(BuildContext context) {
    if (mode == EditorToolbarMode.sidebar) {
      return Row(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          sidebarToolbar,
          Expanded(child: child),
        ],
      );
    }
    return Column(
      children: [
        toolbar,
        Expanded(child: child),
      ],
    );
  }
}
