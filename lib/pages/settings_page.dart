import 'package:autobutler/utils/autobutler_widget.dart';
import 'package:flutter/material.dart';
import 'package:autobutler/services/app_settings.dart';

class SettingsPage extends StatefulWidget {
  const SettingsPage({super.key});

  @override
  State<SettingsPage> createState() => _SettingsPageState();
}

class _SettingsPageState extends State<SettingsPage> {
  List<HostEntry> _hosts = [];
  int _active = -1;
  ThemeMode _theme = ThemeMode.system;

  @override
  void initState() {
    super.initState();
    _load();
  }

  void _load() {
    _hosts = AppSettings.instance.hosts;
    _active = AppSettings.instance.activeIndex;
    _theme = AppSettings.instance.themeMode.value;
    setState(() {});
  }

  Future<void> _addOrEditHost({int? index}) async {
    final isEdit = index != null;
    final idx = index ?? 0;
    final nameController = TextEditingController(
      text: isEdit ? _hosts[idx].name : '',
    );
    final hostController = TextEditingController(
      text: isEdit ? _hosts[idx].hostAddress : '',
    );

    final result = await AutobutlerWidget.showDialog<bool>(
      context,
      builder: (context) => AutobutlerWidget.alertDialog(
        title: Text(isEdit ? 'Edit host' : 'Add host'),
        content: Column(
          mainAxisSize: MainAxisSize.min,
          children: [
            AutobutlerWidget.textField(
              context,
              controller: nameController,
              autofocus: true,
              hintText: 'Name',
            ),
            const SizedBox(height: 8),
            AutobutlerWidget.textField(
              context,
              controller: hostController,
              hintText: 'http://<hostname>:<port>',
            ),
          ],
        ),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(context).pop(false),
            child: const Text('Cancel'),
          ),
          TextButton(
            onPressed: () async {
              final name = nameController.text.trim();
              final host = hostController.text.trim();
              if (name.isEmpty || host.isEmpty) return;
              final entry = HostEntry(name: name, hostAddress: host);
              final navigator = Navigator.of(context);
              if (isEdit) {
                await AppSettings.instance.updateHost(idx, entry);
              } else {
                await AppSettings.instance.addHost(entry);
              }
              navigator.pop(true);
            },
            child: const Text('Save'),
          ),
        ],
      ),
    );

    if (result == true) {
      _load();
    }
  }

  Future<void> _removeHost(int index) async {
    final confirm = await AutobutlerWidget.showDialog(
      context,
      builder: (context) => AutobutlerWidget.alertDialog(
        title: const Text('Remove host'),
        content: const Text('Are you sure you want to remove this host?'),
        actions: [
          TextButton(
            onPressed: () => Navigator.of(context).pop(false),
            child: const Text('Cancel'),
          ),
          TextButton(
            onPressed: () => Navigator.of(context).pop(true),
            child: const Text('Remove'),
          ),
        ],
      ),
    );

    if (confirm == true) {
      await AppSettings.instance.removeHost(index);
      _load();
    }
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      appBar: AppBar(title: const Text('Settings')),
      body: ListView(
        padding: const EdgeInsets.all(16),
        children: [
          const Text(
            'Theme',
            style: TextStyle(fontSize: 16, fontWeight: FontWeight.bold),
          ),
          RadioGroup<ThemeMode>(
            groupValue: _theme,
            onChanged: (v) async {
              if (v == null) return;
              await AppSettings.instance.setThemeMode(v);
              setState(() {
                _theme = v;
              });
            },
            child: const Column(
              children: [
                RadioListTile<ThemeMode>(
                  title: Text('System'),
                  value: ThemeMode.system,
                ),
                RadioListTile<ThemeMode>(
                  title: Text('Light'),
                  value: ThemeMode.light,
                ),
                RadioListTile<ThemeMode>(
                  title: Text('Dark'),
                  value: ThemeMode.dark,
                ),
              ],
            ),
          ),
          const SizedBox(height: 24),
          const Text(
            'Backend hosts',
            style: TextStyle(fontSize: 16, fontWeight: FontWeight.bold),
          ),
          const SizedBox(height: 8),
          RadioGroup<int>(
            groupValue: _active,
            onChanged: (v) async {
              if (v == null) return;
              await AppSettings.instance.setActiveIndex(v);
              _load();
            },
            child: Column(
              children: _hosts.asMap().entries.map((e) {
                final idx = e.key;
                final host = e.value;
                return Card(
                  child: ListTile(
                    leading: Radio<int>(value: idx),
                    title: Text(host.name),
                    subtitle: Text(host.hostAddress),
                    trailing: PopupMenuButton<String>(
                      onSelected: (action) {
                        if (action == 'edit') {
                          _addOrEditHost(index: idx);
                        } else if (action == 'remove') {
                          _removeHost(idx);
                        }
                      },
                      itemBuilder: (_) => [
                        const PopupMenuItem(value: 'edit', child: Text('Edit')),
                        const PopupMenuItem(
                          value: 'remove',
                          child: Text('Remove'),
                        ),
                      ],
                    ),
                    onTap: () async {
                      await AppSettings.instance.setActiveIndex(idx);
                      _load();
                    },
                  ),
                );
              }).toList(),
            ),
          ),
          const SizedBox(height: 8),
          ElevatedButton.icon(
            onPressed: () => _addOrEditHost(),
            icon: const Icon(Icons.add),
            label: const Text('Add host'),
          ),
        ],
      ),
    );
  }
}
