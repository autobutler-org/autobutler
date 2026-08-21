import 'dart:convert';

import 'package:flutter/material.dart';
import 'package:go_router/go_router.dart';
import 'package:http/http.dart' as http;
import 'package:quark/router.dart';
import 'package:quark/services/cirrus_service.dart';
import 'package:quark/utils/file_browser_path_utils.dart';
import 'package:quark/widgets/layout/theme_toggle_button.dart';

/// A simple plaintext editor for text-like files (txt, md, json, yaml, etc.)
class PlaintextEditorPage extends StatefulWidget {
  final String filePath;
  final String deviceSerial;

  const PlaintextEditorPage({
    required this.filePath,
    this.deviceSerial = '',
    super.key,
  });

  @override
  State<PlaintextEditorPage> createState() => _PlaintextEditorPageState();
}

class _PlaintextEditorPageState extends State<PlaintextEditorPage> {
  final TextEditingController _textController = TextEditingController();

  bool _loading = true;
  bool _saving = false;
  bool _dirty = false;
  String? _error;

  late String _displayName;

  @override
  void initState() {
    super.initState();
    _displayName = widget.filePath.split('/').last;
    _textController.addListener(_onTextChanged);
    _loadFile();
  }

  @override
  void dispose() {
    _textController.dispose();
    super.dispose();
  }

  void _onTextChanged() {
    if (!_dirty && mounted) {
      setState(() => _dirty = true);
    }
  }

  Future<void> _loadFile() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final serial = serialOrNull(widget.deviceSerial);
      final bytes = await CirrusService.downloadFileBytes(
        widget.filePath,
        serial: serial,
      );
      if (!mounted) return;
      final text = bytes != null && bytes.isNotEmpty
          ? utf8.decode(bytes, allowMalformed: true)
          : '';
      _textController.removeListener(_onTextChanged);
      _textController.text = text;
      _textController.addListener(_onTextChanged);
      setState(() {
        _loading = false;
        _dirty = false;
      });
    } catch (e) {
      if (!mounted) return;
      setState(() {
        _loading = false;
        _error = 'Failed to load file: $e';
      });
    }
  }

  Future<void> _saveFile() async {
    if (_saving) return;
    setState(() => _saving = true);
    try {
      final bytes = utf8.encode(_textController.text);
      final fileName = _displayName;
      final parentDir = parentPath(widget.filePath);
      final serial = serialOrNull(widget.deviceSerial);
      final file = http.MultipartFile.fromBytes(
        'files',
        bytes,
        filename: fileName,
      );
      await CirrusService.uploadFilesFromFormData(
        parentDir,
        [file],
        serial: serial,
        overwrite: true,
      );
      if (!mounted) return;
      setState(() {
        _saving = false;
        _dirty = false;
      });
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(const SnackBar(content: Text('Saved')));
    } catch (e) {
      if (!mounted) return;
      setState(() => _saving = false);
      ScaffoldMessenger.of(
        context,
      ).showSnackBar(SnackBar(content: Text('Save failed: $e')));
    }
  }

  @override
  Widget build(BuildContext context) {
    final title = _dirty ? '$_displayName •' : _displayName;

    return Scaffold(
      appBar: AppBar(
        title: Text(title),
        leading: IconButton(
          icon: const Icon(Icons.arrow_back),
          tooltip: 'Back',
          onPressed: () {
            if (context.canPop()) {
              context.pop();
            } else {
              context.go(AppRoutes.cirrus);
            }
          },
        ),
        actions: [
          if (_saving)
            const Padding(
              padding: EdgeInsets.symmetric(horizontal: 12),
              child: SizedBox(
                width: 20,
                height: 20,
                child: CircularProgressIndicator(strokeWidth: 2),
              ),
            )
          else
            IconButton(
              icon: const Icon(Icons.save_outlined),
              tooltip: 'Save',
              onPressed: _dirty ? _saveFile : null,
            ),
          const ThemeToggleButton(),
        ],
      ),
      body: _buildBody(context),
    );
  }

  Widget _buildBody(BuildContext context) {
    if (_loading) {
      return const Center(child: CircularProgressIndicator());
    }

    if (_error != null) {
      return Center(
        child: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            Icon(
              Icons.error_outline,
              size: 48,
              color: Theme.of(context).colorScheme.error,
            ),
            const SizedBox(height: 12),
            Text(_error!, textAlign: TextAlign.center),
            const SizedBox(height: 12),
            FilledButton(onPressed: _loadFile, child: const Text('Retry')),
          ],
        ),
      );
    }

    return SingleChildScrollView(
      padding: const EdgeInsets.all(16),
      child: TextField(
        controller: _textController,
        maxLines: null,
        keyboardType: TextInputType.multiline,
        style: const TextStyle(fontFamily: 'monospace', fontSize: 13),
        decoration: const InputDecoration(
          border: InputBorder.none,
          hintText: 'Empty file',
          isCollapsed: true,
        ),
      ),
    );
  }
}
