import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../state/app_state.dart';
import '../utils/capture_content.dart';

class InboxScreen extends StatefulWidget {
  const InboxScreen({super.key});

  @override
  State<InboxScreen> createState() => _InboxScreenState();
}

class _InboxScreenState extends State<InboxScreen> {
  final _body = TextEditingController();
  bool _saving = false;

  Future<void> _save() async {
    final text = _body.text.trim();
    if (text.isEmpty) return;
    setState(() => _saving = true);
    try {
      await context.read<AppState>().saveCapture(text);
      _body.clear();
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          const SnackBar(content: Text('Saved')),
        );
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(content: Text(e.toString())),
        );
      }
    } finally {
      if (mounted) setState(() => _saving = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final state = context.watch<AppState>();

    return Scaffold(
      backgroundColor: const Color(0xFF0D1117),
      appBar: AppBar(
        title: Text(state.selectedProjectName),
        backgroundColor: const Color(0xFF161B22),
        actions: [
          PopupMenuButton<String>(
            icon: const Icon(Icons.folder_outlined),
            onSelected: state.setProject,
            itemBuilder: (_) => state.projects
                .map((p) => PopupMenuItem(value: p.id, child: Text(p.name)))
                .toList(),
          ),
          IconButton(
            icon: const Icon(Icons.lock_outline),
            onPressed: () {
              state.lockVault();
              Navigator.of(context).pushReplacementNamed('/vault');
            },
          ),
        ],
      ),
      body: Column(
        children: [
          Padding(
            padding: const EdgeInsets.all(12),
            child: TextField(
              controller: _body,
              minLines: 3,
              maxLines: 6,
              decoration: InputDecoration(
                hintText: 'Paste anything here…',
                filled: true,
                fillColor: const Color(0xFF161B22),
                border: OutlineInputBorder(borderRadius: BorderRadius.circular(12)),
              ),
            ),
          ),
          Padding(
            padding: const EdgeInsets.symmetric(horizontal: 12),
            child: SizedBox(
              width: double.infinity,
              child: FilledButton(
                onPressed: _saving ? null : _save,
                child: _saving
                    ? const SizedBox(
                        height: 18,
                        width: 18,
                        child: CircularProgressIndicator(strokeWidth: 2),
                      )
                    : const Text('Save'),
              ),
            ),
          ),
          const SizedBox(height: 8),
          Expanded(
            child: RefreshIndicator(
              onRefresh: state.loadData,
              child: state.captures.isEmpty
                  ? ListView(
                      children: const [
                        SizedBox(height: 80),
                        Center(
                          child: Text(
                            'Nothing yet — paste something above',
                            style: TextStyle(color: Color(0xFF8B949E)),
                          ),
                        ),
                      ],
                    )
                  : ListView.builder(
                      itemCount: state.captures.length,
                      itemBuilder: (_, i) {
                        final cap = state.captures[i];
                        final title = (cap['title'] as String?)?.trim().isNotEmpty == true
                            ? cap['title'] as String
                            : generateAutoTitle(cap['body'] as String? ?? '');
                        final body = cap['body'] as String? ?? '';
                        final created = cap['created_at'] as int? ?? 0;

                        return ListTile(
                          title: Text(title, maxLines: 1, overflow: TextOverflow.ellipsis),
                          subtitle: Text(
                            body,
                            maxLines: 2,
                            overflow: TextOverflow.ellipsis,
                          ),
                          trailing: Text(
                            formatRelativeTime(created),
                            style: const TextStyle(fontSize: 12, color: Color(0xFF8B949E)),
                          ),
                          onTap: () => _showDetail(context, cap),
                        );
                      },
                    ),
            ),
          ),
        ],
      ),
    );
  }

  void _showDetail(BuildContext context, Map<String, dynamic> cap) {
    showModalBottomSheet(
      context: context,
      isScrollControlled: true,
      backgroundColor: const Color(0xFF161B22),
      builder: (ctx) => Padding(
        padding: EdgeInsets.only(
          left: 16,
          right: 16,
          top: 16,
          bottom: MediaQuery.of(ctx).viewInsets.bottom + 24,
        ),
        child: Column(
          mainAxisSize: MainAxisSize.min,
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Text(
              cap['title'] as String? ?? 'Capture',
              style: Theme.of(ctx).textTheme.titleLarge,
            ),
            const SizedBox(height: 12),
            SelectableText(cap['body'] as String? ?? ''),
            const SizedBox(height: 16),
            OutlinedButton.icon(
              icon: const Icon(Icons.delete_outline),
              label: const Text('Move to trash'),
              onPressed: () async {
                Navigator.pop(ctx);
                await context.read<AppState>().deleteCapture(cap['id'] as String);
              },
            ),
          ],
        ),
      ),
    );
  }
}
