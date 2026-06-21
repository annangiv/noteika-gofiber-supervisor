import 'dart:async';
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
  final _searchController = TextEditingController();
  bool _saving = false;
  bool _searching = false;

  // Real-time duplicate check states
  List<Map<String, dynamic>> _duplicates = [];
  Timer? _debounce;

  @override
  void initState() {
    super.initState();
    _body.addListener(_onBodyChanged);
  }

  @override
  void dispose() {
    _body.removeListener(_onBodyChanged);
    _body.dispose();
    _searchController.dispose();
    _debounce?.cancel();
    super.dispose();
  }

  void _onBodyChanged() {
    if (_debounce?.isActive ?? false) _debounce!.cancel();
    _debounce = Timer(const Duration(milliseconds: 350), () async {
      final text = _body.text.trim();
      if (text.isEmpty) {
        if (mounted) setState(() => _duplicates = []);
        return;
      }
      final state = context.read<AppState>();
      final hits = await state.findDuplicates(text);
      if (mounted) {
        setState(() {
          _duplicates = hits;
        });
      }
    });
  }

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
        title: _searching
            ? TextField(
                controller: _searchController,
                style: const TextStyle(color: Colors.white),
                decoration: const InputDecoration(
                  hintText: 'Search notes by meaning...',
                  border: InputBorder.none,
                  filled: false,
                  hintStyle: TextStyle(color: Color(0xFF8B949E)),
                ),
                onChanged: (val) {
                  state.search(val);
                },
                autofocus: true,
              )
            : Text(state.selectedProjectId == 'trash' ? 'Trash' : state.selectedProjectName),
        backgroundColor: const Color(0xFF161B22),
        actions: [
          if (_searching)
            IconButton(
              icon: const Icon(Icons.close),
              onPressed: () {
                setState(() {
                  _searching = false;
                  _searchController.clear();
                });
                state.search('');
              },
            )
          else
            IconButton(
              icon: const Icon(Icons.search),
              onPressed: () => setState(() => _searching = true),
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
      drawer: Drawer(
        backgroundColor: const Color(0xFF161B22),
        child: Column(
          children: [
            UserAccountsDrawerHeader(
              decoration: const BoxDecoration(color: Color(0xFF0D1117)),
              currentAccountPicture: CircleAvatar(
                backgroundColor: const Color(0xFF58A6FF),
                child: Text(
                  (state.user?['full_name'] as String? ?? 'N').substring(0, 1).toUpperCase(),
                  style: const TextStyle(color: Colors.white, fontSize: 24, fontWeight: FontWeight.bold),
                ),
              ),
              accountName: Text(
                state.user?['full_name'] ?? 'Developer User',
                style: const TextStyle(color: Colors.white, fontWeight: FontWeight.bold),
              ),
              accountEmail: Text(
                state.user?['email'] ?? 'dev-user@example.com',
                style: const TextStyle(color: Color(0xFF8B949E)),
              ),
            ),
            Expanded(
              child: ListView(
                padding: EdgeInsets.zero,
                children: [
                  ListTile(
                    leading: const Icon(Icons.note_alt_outlined, color: Colors.white),
                    title: const Text('All Notes', style: TextStyle(color: Colors.white)),
                    selected: state.selectedProjectId == 'inbox',
                    onTap: () {
                      state.setProject('inbox');
                      Navigator.pop(context);
                    },
                  ),
                  const Divider(color: Color(0xFF30363D)),
                  const Padding(
                    padding: EdgeInsets.symmetric(horizontal: 16, vertical: 8),
                    child: Text(
                      'Projects',
                      style: TextStyle(color: Color(0xFF8B949E), fontWeight: FontWeight.bold, fontSize: 12),
                    ),
                  ),
                  ...state.projects
                      .where((p) => p.id != 'inbox' && p.id != 'trash')
                      .map((p) => ListTile(
                            leading: const Icon(Icons.folder_open_outlined, color: Color(0xFF8B949E)),
                            title: Text(p.name, style: const TextStyle(color: Colors.white)),
                            selected: state.selectedProjectId == p.id,
                            onTap: () {
                              state.setProject(p.id);
                              Navigator.pop(context);
                            },
                          )),
                  const Divider(color: Color(0xFF30363D)),
                  ListTile(
                    leading: const Icon(Icons.delete_outline, color: Color(0xFFDA3637)),
                    title: const Text('Trash', style: TextStyle(color: Color(0xFFDA3637))),
                    selected: state.selectedProjectId == 'trash',
                    onTap: () {
                      state.setProject('trash');
                      Navigator.pop(context);
                    },
                  ),
                  ListTile(
                    leading: const Icon(Icons.account_circle_outlined, color: Colors.white),
                    title: const Text('Account', style: TextStyle(color: Colors.white)),
                    onTap: () {
                      Navigator.pop(context);
                      Navigator.pushNamed(context, '/account');
                    },
                  ),
                  ListTile(
                    leading: const Icon(Icons.credit_card, color: Colors.white),
                    title: const Text('Pricing', style: TextStyle(color: Colors.white)),
                    onTap: () {
                      Navigator.pop(context);
                      Navigator.pushNamed(context, '/pricing');
                    },
                  ),
                  ListTile(
                    leading: const Icon(Icons.info_outline, color: Colors.white),
                    title: const Text('About', style: TextStyle(color: Colors.white)),
                    onTap: () {
                      Navigator.pop(context);
                      Navigator.pushNamed(context, '/about');
                    },
                  ),
                ],
              ),
            ),
            const Divider(color: Color(0xFF30363D)),
            ListTile(
              leading: const Icon(Icons.logout, color: Color(0xFF8B949E)),
              title: const Text('Logout', style: TextStyle(color: Color(0xFF8B949E))),
              onTap: () async {
                Navigator.pop(context);
                await state.logout();
                if (context.mounted) {
                  Navigator.of(context).pushReplacementNamed('/login');
                }
              },
            ),
            const SizedBox(height: 16),
          ],
        ),
      ),
      body: Column(
        children: [
          // Model Download progress banner
          if (state.modelDownloading)
            Container(
              padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
              color: const Color(0xFF161B22),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  Row(
                    mainAxisAlignment: MainAxisAlignment.spaceBetween,
                    children: [
                      const Text(
                        'Downloading semantic search model...',
                        style: TextStyle(color: Colors.white, fontSize: 13, fontWeight: FontWeight.bold),
                      ),
                      Text(
                        '${(state.downloadProgress * 100).toStringAsFixed(0)}%',
                        style: const TextStyle(color: Color(0xFF58A6FF), fontSize: 13, fontWeight: FontWeight.bold),
                      ),
                    ],
                  ),
                  const SizedBox(height: 8),
                  LinearProgressIndicator(
                    value: state.downloadProgress,
                    backgroundColor: const Color(0xFF30363D),
                    color: const Color(0xFF58A6FF),
                  ),
                ],
              ),
            ),
          
          if (state.modelError != null)
            Container(
              padding: const EdgeInsets.all(12),
              color: const Color(0x33DA3637),
              child: Row(
                children: [
                  const Icon(Icons.warning_amber_rounded, color: Color(0xFFDA3637)),
                  const SizedBox(width: 12),
                  Expanded(
                    child: Text(
                      'Semantic load failed. keyword search active.',
                      style: const TextStyle(color: Color(0xFFDA3637), fontSize: 13),
                    ),
                  ),
                  IconButton(
                    icon: const Icon(Icons.refresh, color: Colors.white),
                    onPressed: () => state.initEmbeddingModel(),
                  ),
                ],
              ),
            ),

          if (!_searching && state.selectedProjectId != 'trash') ...[
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
                  style: FilledButton.styleFrom(
                    backgroundColor: const Color(0xFF1F6FEB),
                    shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(8)),
                  ),
                  child: _saving
                      ? const SizedBox(
                          height: 18,
                          width: 18,
                          child: CircularProgressIndicator(strokeWidth: 2, color: Colors.white),
                        )
                      : const Text('Capture'),
                ),
              ),
            ),

            // Duplicate warnings list
            if (_duplicates.isNotEmpty)
              Padding(
                padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 8),
                child: Container(
                  padding: const EdgeInsets.all(16),
                  decoration: BoxDecoration(
                    color: const Color(0xFF262216),
                    borderRadius: BorderRadius.circular(8),
                    border: Border.all(color: const Color(0xFFD4A32D)),
                  ),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.stretch,
                    children: [
                      const Row(
                        children: [
                          Icon(Icons.warning_amber_rounded, color: Color(0xFFD4A32D), size: 20),
                          SizedBox(width: 8),
                          Text(
                            'Already in your docket',
                            style: TextStyle(color: Color(0xFFD4A32D), fontWeight: FontWeight.bold, fontSize: 14),
                          ),
                        ],
                      ),
                      const SizedBox(height: 8),
                      ..._duplicates.map((dup) {
                        final sim = ((dup['similarity'] as double) * 100).round();
                        final title = dup['capture']['title'] as String? ?? 'Untitled Note';
                        final created = dup['capture']['created_at'] as int? ?? 0;
                        return Padding(
                          padding: const EdgeInsets.only(bottom: 6),
                          child: Text(
                            '• $title ($sim% match) created ${formatRelativeTime(created)}',
                            style: const TextStyle(color: Color(0xFFE2B04E), fontSize: 13),
                          ),
                        );
                      }),
                    ],
                  ),
                ),
              ),
            const SizedBox(height: 8),
          ],

          Expanded(
            child: RefreshIndicator(
              onRefresh: state.loadData,
              child: _buildMainList(state),
            ),
          ),
        ],
      ),
    );
  }

  Widget _buildMainList(AppState state) {
    if (state.isSearching && state.searchQuery.trim().isNotEmpty) {
      if (state.searchResults.isEmpty) {
        return ListView(
          children: const [
            SizedBox(height: 80),
            Center(
              child: Text(
                'No matching notes found',
                style: TextStyle(color: Color(0xFF8B949E)),
              ),
            ),
          ],
        );
      }
      return ListView.builder(
        itemCount: state.searchResults.length,
        itemBuilder: (_, i) {
          final item = state.searchResults[i];
          final cap = item['capture'];
          final sim = ((item['similarity'] as double) * 100).round();
          final title = (cap['title'] as String?)?.trim().isNotEmpty == true
              ? cap['title'] as String
              : generateAutoTitle(cap['body'] as String? ?? '');
          final body = cap['body'] as String? ?? '';
          final created = cap['created_at'] as int? ?? 0;

          return ListTile(
            title: Row(
              children: [
                Expanded(child: Text(title, maxLines: 1, overflow: TextOverflow.ellipsis)),
                const SizedBox(width: 8),
                Container(
                  padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
                  decoration: BoxDecoration(
                    color: const Color(0x2258A6FF),
                    border: Border.all(color: const Color(0x5558A6FF)),
                    borderRadius: BorderRadius.circular(4),
                  ),
                  child: Text(
                    '$sim% match',
                    style: const TextStyle(color: Color(0xFF58A6FF), fontSize: 10, fontWeight: FontWeight.bold),
                  ),
                ),
              ],
            ),
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
      );
    }

    if (state.captures.isEmpty) {
      return ListView(
        children: const [
          SizedBox(height: 80),
          Center(
            child: Text(
              'Nothing yet — paste something above',
              style: TextStyle(color: Color(0xFF8B949E)),
            ),
          ),
        ],
      );
    }

    return ListView.builder(
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
            Row(
              mainAxisAlignment: MainAxisAlignment.spaceBetween,
              children: [
                Expanded(
                  child: Text(
                    cap['title'] as String? ?? 'Capture',
                    style: const TextStyle(color: Colors.white, fontSize: 20, fontWeight: FontWeight.bold),
                  ),
                ),
                IconButton(
                  icon: const Icon(Icons.edit_outlined, color: Color(0xFF58A6FF)),
                  onPressed: () {
                    Navigator.pop(ctx);
                    _showEditForm(context, cap);
                  },
                ),
              ],
            ),
            const SizedBox(height: 12),
            SelectableText(
              cap['body'] as String? ?? '',
              style: const TextStyle(color: Color(0xFFC9D1D9), fontSize: 15, height: 1.4),
            ),
            const SizedBox(height: 24),
            OutlinedButton.icon(
              icon: const Icon(Icons.delete_outline),
              label: Text(cap['deleted_at'] != null && cap['deleted_at'] > 0
                  ? 'Delete permanently'
                  : 'Move to trash'),
              style: OutlinedButton.styleFrom(
                foregroundColor: const Color(0xFFDA3637),
                side: const BorderSide(color: Color(0xFFDA3637)),
              ),
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

  void _showEditForm(BuildContext context, Map<String, dynamic> cap) {
    final bodyController = TextEditingController(text: cap['body'] as String? ?? '');
    final tagsController = TextEditingController(
      text: (cap['tags'] as List?)?.join(', ') ?? '',
    );
    bool isSavingEdit = false;

    showModalBottomSheet(
      context: context,
      isScrollControlled: true,
      backgroundColor: const Color(0xFF161B22),
      builder: (ctx) => StatefulBuilder(
        builder: (ctx, setModalState) => Padding(
          padding: EdgeInsets.only(
            left: 16,
            right: 16,
            top: 20,
            bottom: MediaQuery.of(ctx).viewInsets.bottom + 24,
          ),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              const Text(
                'Edit note',
                style: TextStyle(color: Colors.white, fontSize: 18, fontWeight: FontWeight.bold),
              ),
              const SizedBox(height: 16),
              TextField(
                controller: bodyController,
                minLines: 3,
                maxLines: 8,
                decoration: const InputDecoration(
                  labelText: 'Body',
                  fillColor: Color(0xFF0D1117),
                ),
              ),
              const SizedBox(height: 12),
              TextField(
                controller: tagsController,
                decoration: const InputDecoration(
                  labelText: 'Tags (comma separated)',
                  hintText: 'groceries, breakfast',
                  fillColor: Color(0xFF0D1117),
                ),
              ),
              const SizedBox(height: 24),
              Row(
                mainAxisAlignment: MainAxisAlignment.end,
                children: [
                  TextButton(
                    onPressed: () => Navigator.pop(ctx),
                    child: const Text('Cancel', style: TextStyle(color: Color(0xFF8B949E))),
                  ),
                  const SizedBox(width: 8),
                  FilledButton(
                    onPressed: isSavingEdit
                        ? null
                        : () async {
                            final text = bodyController.text.trim();
                            if (text.isEmpty) return;
                            setModalState(() => isSavingEdit = true);
                            try {
                              final tagList = tagsController.text
                                  .split(',')
                                  .map((e) => e.trim())
                                  .where((e) => e.isNotEmpty)
                                  .toList();
                              await context
                                  .read<AppState>()
                                  .editCapture(cap['id'] as String, text, tags: tagList);
                              if (ctx.mounted) Navigator.pop(ctx);
                            } catch (e) {
                              if (ctx.mounted) {
                                ScaffoldMessenger.of(ctx).showSnackBar(
                                  SnackBar(content: Text('Failed to update: $e')),
                                );
                              }
                            } finally {
                              setModalState(() => isSavingEdit = false);
                            }
                          },
                    style: FilledButton.styleFrom(backgroundColor: const Color(0xFF1F6FEB)),
                    child: isSavingEdit
                        ? const SizedBox(
                            height: 18,
                            width: 18,
                            child: CircularProgressIndicator(strokeWidth: 2, color: Colors.white),
                          )
                        : const Text('Save'),
                  ),
                ],
              ),
            ],
          ),
        ),
      ),
    );
  }
}
