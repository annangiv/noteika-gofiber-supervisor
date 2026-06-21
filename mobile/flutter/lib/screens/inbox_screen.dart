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
      backgroundColor: const Color(0xFF090A0C),
      appBar: AppBar(
        title: _searching
            ? Container(
                height: 40,
                decoration: BoxDecoration(
                  color: const Color(0xFF13151A),
                  borderRadius: BorderRadius.circular(10),
                  border: Border.all(color: const Color(0xFF1F2228)),
                ),
                padding: const EdgeInsets.symmetric(horizontal: 12),
                child: TextField(
                  controller: _searchController,
                  style: const TextStyle(color: Colors.white, fontSize: 14),
                  decoration: const InputDecoration(
                    hintText: 'Search notes by meaning...',
                    border: InputBorder.none,
                    enabledBorder: InputBorder.none,
                    focusedBorder: InputBorder.none,
                    filled: false,
                    contentPadding: EdgeInsets.symmetric(vertical: 10),
                    hintStyle: TextStyle(color: Color(0xFF6B7280)),
                  ),
                  onChanged: (val) {
                    state.search(val);
                  },
                  autofocus: true,
                ),
              )
            : Text(
                state.selectedProjectId == 'trash' ? 'Trash' : state.selectedProjectName,
                style: const TextStyle(fontWeight: FontWeight.bold, fontSize: 20),
              ),
        backgroundColor: const Color(0xFF090A0C),
        bottom: PreferredSize(
          preferredSize: const Size.fromHeight(1),
          child: Container(
            color: const Color(0xFF1F2228),
            height: 1,
          ),
        ),
        actions: [
          if (_searching)
            IconButton(
              icon: const Icon(Icons.close_rounded),
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
              icon: const Icon(Icons.search_rounded),
              onPressed: () => setState(() => _searching = true),
            ),
          IconButton(
            icon: const Icon(Icons.lock_outline_rounded),
            onPressed: () {
              state.lockVault();
              Navigator.of(context).pushReplacementNamed('/vault');
            },
          ),
        ],
      ),
      drawer: Drawer(
        backgroundColor: const Color(0xFF13151A),
        child: Column(
          children: [
            // Custom premium drawer header (no material accounts header)
            Container(
              padding: const EdgeInsets.only(left: 20, right: 20, top: 60, bottom: 24),
              decoration: const BoxDecoration(
                color: Color(0xFF090A0C),
                border: Border(bottom: BorderSide(color: Color(0xFF1F2228))),
              ),
              child: Row(
                children: [
                  CircleAvatar(
                    radius: 26,
                    backgroundColor: const Color(0xFF8B5CF6),
                    child: Text(
                      (state.user?['full_name'] as String? ?? 'N').substring(0, 1).toUpperCase(),
                      style: const TextStyle(color: Colors.white, fontSize: 20, fontWeight: FontWeight.bold),
                    ),
                  ),
                  const SizedBox(width: 14),
                  Expanded(
                    child: Column(
                      crossAxisAlignment: CrossAxisAlignment.start,
                      children: [
                        Text(
                          state.user?['full_name'] ?? 'Developer User',
                          style: const TextStyle(color: Colors.white, fontWeight: FontWeight.bold, fontSize: 16),
                          overflow: TextOverflow.ellipsis,
                        ),
                        const SizedBox(height: 4),
                        Text(
                          state.user?['email'] ?? 'dev-user@example.com',
                          style: const TextStyle(color: Color(0xFF9CA3AF), fontSize: 12),
                          overflow: TextOverflow.ellipsis,
                        ),
                      ],
                    ),
                  ),
                ],
              ),
            ),
            const SizedBox(height: 12),
            Expanded(
              child: ListView(
                padding: EdgeInsets.zero,
                children: [
                  Padding(
                    padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 2),
                    child: ListTile(
                      leading: Icon(
                        Icons.note_alt_outlined, 
                        color: state.selectedProjectId == 'inbox' ? const Color(0xFF8B5CF6) : const Color(0xFF9CA3AF),
                      ),
                      title: Text(
                        'All Notes', 
                        style: TextStyle(
                          color: state.selectedProjectId == 'inbox' ? Colors.white : const Color(0xFF9CA3AF),
                          fontWeight: state.selectedProjectId == 'inbox' ? FontWeight.bold : FontWeight.normal,
                        ),
                      ),
                      selected: state.selectedProjectId == 'inbox',
                      selectedTileColor: const Color(0xFF8B5CF6).withOpacity(0.08),
                      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(10)),
                      onTap: () {
                        state.setProject('inbox');
                        Navigator.pop(context);
                      },
                    ),
                  ),
                  const Padding(
                    padding: EdgeInsets.symmetric(horizontal: 20, vertical: 8),
                    child: Divider(color: Color(0xFF1F2228)),
                  ),
                  const Padding(
                    padding: EdgeInsets.symmetric(horizontal: 20, vertical: 4),
                    child: Text(
                      'PROJECTS',
                      style: TextStyle(color: Color(0xFF6B7280), fontWeight: FontWeight.bold, fontSize: 11, letterSpacing: 0.8),
                    ),
                  ),
                  ...state.projects
                      .where((p) => p.id != 'inbox' && p.id != 'trash')
                      .map((p) => Padding(
                            padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 2),
                            child: ListTile(
                              leading: Icon(
                                Icons.folder_open_rounded, 
                                color: state.selectedProjectId == p.id ? const Color(0xFF8B5CF6) : const Color(0xFF9CA3AF),
                              ),
                              title: Text(
                                p.name, 
                                style: TextStyle(
                                  color: state.selectedProjectId == p.id ? Colors.white : const Color(0xFF9CA3AF),
                                  fontWeight: state.selectedProjectId == p.id ? FontWeight.bold : FontWeight.normal,
                                ),
                              ),
                              selected: state.selectedProjectId == p.id,
                              selectedTileColor: const Color(0xFF8B5CF6).withOpacity(0.08),
                              shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(10)),
                              onTap: () {
                                state.setProject(p.id);
                                Navigator.pop(context);
                              },
                            ),
                          )),
                  const Padding(
                    padding: EdgeInsets.symmetric(horizontal: 20, vertical: 8),
                    child: Divider(color: Color(0xFF1F2228)),
                  ),
                  Padding(
                    padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 2),
                    child: ListTile(
                      leading: Icon(
                        Icons.delete_outline_rounded, 
                        color: state.selectedProjectId == 'trash' ? const Color(0xFFEF4444) : const Color(0xFF9CA3AF),
                      ),
                      title: Text(
                        'Trash', 
                        style: TextStyle(
                          color: state.selectedProjectId == 'trash' ? const Color(0xFFEF4444) : const Color(0xFF9CA3AF),
                          fontWeight: state.selectedProjectId == 'trash' ? FontWeight.bold : FontWeight.normal,
                        ),
                      ),
                      selected: state.selectedProjectId == 'trash',
                      selectedTileColor: const Color(0xFFEF4444).withOpacity(0.08),
                      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(10)),
                      onTap: () {
                        state.setProject('trash');
                        Navigator.pop(context);
                      },
                    ),
                  ),
                  Padding(
                    padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 2),
                    child: ListTile(
                      leading: const Icon(Icons.account_circle_outlined, color: Color(0xFF9CA3AF)),
                      title: const Text('Account', style: TextStyle(color: Color(0xFF9CA3AF))),
                      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(10)),
                      onTap: () {
                        Navigator.pop(context);
                        Navigator.pushNamed(context, '/account');
                      },
                    ),
                  ),
                  Padding(
                    padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 2),
                    child: ListTile(
                      leading: const Icon(Icons.credit_card_rounded, color: Color(0xFF9CA3AF)),
                      title: const Text('Pricing', style: TextStyle(color: Color(0xFF9CA3AF))),
                      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(10)),
                      onTap: () {
                        Navigator.pop(context);
                        Navigator.pushNamed(context, '/pricing');
                      },
                    ),
                  ),
                  Padding(
                    padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 2),
                    child: ListTile(
                      leading: const Icon(Icons.info_outline_rounded, color: Color(0xFF9CA3AF)),
                      title: const Text('About Noteika', style: TextStyle(color: Color(0xFF9CA3AF))),
                      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(10)),
                      onTap: () {
                        Navigator.pop(context);
                        Navigator.pushNamed(context, '/about');
                      },
                    ),
                  ),
                ],
              ),
            ),
            const Divider(color: Color(0xFF1F2228)),
            Padding(
              padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
              child: ListTile(
                leading: const Icon(Icons.logout_rounded, color: Color(0xFF6B7280)),
                title: const Text('Logout', style: TextStyle(color: Color(0xFF6B7280))),
                shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(10)),
                onTap: () async {
                  Navigator.pop(context);
                  await state.logout();
                  if (context.mounted) {
                    Navigator.of(context).pushReplacementNamed('/login');
                  }
                },
              ),
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
              decoration: const BoxDecoration(
                color: Color(0xFF13151A),
                border: Border(bottom: BorderSide(color: Color(0xFF1F2228))),
              ),
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
                        style: const TextStyle(color: Color(0xFF8B5CF6), fontSize: 13, fontWeight: FontWeight.bold),
                      ),
                    ],
                  ),
                  const SizedBox(height: 8),
                  LinearProgressIndicator(
                    value: state.downloadProgress,
                    backgroundColor: const Color(0xFF1F2228),
                    color: const Color(0xFF8B5CF6),
                  ),
                ],
              ),
            ),
          
          if (state.modelError != null)
            Container(
              padding: const EdgeInsets.all(12),
              decoration: const BoxDecoration(
                color: Color(0x1AEF4444),
                border: Border(bottom: BorderSide(color: Color(0x33EF4444))),
              ),
              child: Row(
                children: [
                  const Icon(Icons.warning_amber_rounded, color: Color(0xFFEF4444)),
                  const SizedBox(width: 12),
                  const Expanded(
                    child: Text(
                      'Semantic load failed. keyword search active.',
                      style: TextStyle(color: Color(0xFFEF4444), fontSize: 13),
                    ),
                  ),
                  IconButton(
                    icon: const Icon(Icons.refresh_rounded, color: Colors.white),
                    onPressed: () => state.initEmbeddingModel(),
                  ),
                ],
              ),
            ),

          if (!_searching && state.selectedProjectId != 'trash') ...[
            Padding(
              padding: const EdgeInsets.all(12),
              child: Container(
                decoration: BoxDecoration(
                  color: const Color(0xFF13151A),
                  borderRadius: BorderRadius.circular(16),
                  border: Border.all(color: const Color(0xFF1F2228)),
                ),
                padding: const EdgeInsets.all(12),
                child: Column(
                  crossAxisAlignment: CrossAxisAlignment.stretch,
                  children: [
                    TextField(
                      controller: _body,
                      minLines: 2,
                      maxLines: 6,
                      style: const TextStyle(fontSize: 14.5),
                      decoration: const InputDecoration(
                        hintText: 'Paste anything here…',
                        filled: false,
                        border: InputBorder.none,
                        enabledBorder: InputBorder.none,
                        focusedBorder: InputBorder.none,
                        contentPadding: EdgeInsets.zero,
                      ),
                    ),
                    const SizedBox(height: 12),
                    Row(
                      mainAxisAlignment: MainAxisAlignment.spaceBetween,
                      children: [
                        // Tag/format indicators
                        Row(
                          children: [
                            Icon(Icons.tag_rounded, size: 18, color: const Color(0xFF6B7280)),
                            const SizedBox(width: 4),
                            Text(
                              'Auto-tagging active',
                              style: TextStyle(color: const Color(0xFF6B7280), fontSize: 12),
                            ),
                          ],
                        ),
                        SizedBox(
                          height: 36,
                          child: DecoratedBox(
                            decoration: BoxDecoration(
                              borderRadius: BorderRadius.circular(8),
                              gradient: const LinearGradient(
                                colors: [Color(0xFF7C3AED), Color(0xFF8B5CF6)],
                              ),
                            ),
                            child: ElevatedButton(
                              onPressed: _saving ? null : _save,
                              style: ElevatedButton.styleFrom(
                                backgroundColor: Colors.transparent,
                                shadowColor: Colors.transparent,
                                shape: RoundedRectangleBorder(
                                  borderRadius: BorderRadius.circular(8),
                                ),
                                padding: const EdgeInsets.symmetric(horizontal: 16),
                                textStyle: const TextStyle(fontSize: 13, fontWeight: FontWeight.bold),
                              ),
                              child: _saving
                                  ? const SizedBox(
                                      height: 14,
                                      width: 14,
                                      child: CircularProgressIndicator(strokeWidth: 2, color: Colors.white),
                                    )
                                  : const Text('Capture'),
                            ),
                          ),
                        ),
                      ],
                    ),
                  ],
                ),
              ),
            ),

            // Duplicate warnings list
            if (_duplicates.isNotEmpty)
              Padding(
                padding: const EdgeInsets.symmetric(horizontal: 12, vertical: 4),
                child: Container(
                  padding: const EdgeInsets.all(14),
                  decoration: BoxDecoration(
                    color: const Color(0xFF231809),
                    borderRadius: BorderRadius.circular(12),
                    border: Border.all(color: const Color(0xFF8A590F)),
                  ),
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.stretch,
                    children: [
                      const Row(
                        children: [
                          Icon(Icons.warning_amber_rounded, color: Color(0xFFFBBF24), size: 18),
                          SizedBox(width: 8),
                          Text(
                            'Already in your docket',
                            style: TextStyle(color: Color(0xFFFBBF24), fontWeight: FontWeight.bold, fontSize: 13.5),
                          ),
                        ],
                      ),
                      const SizedBox(height: 8),
                      ..._duplicates.map((dup) {
                        final sim = ((dup['similarity'] as double) * 100).round();
                        final title = dup['capture']['title'] as String? ?? 'Untitled Note';
                        final created = dup['capture']['created_at'] as int? ?? 0;
                        return Padding(
                          padding: const EdgeInsets.only(bottom: 4),
                          child: Text(
                            '• $title ($sim% match) created ${formatRelativeTime(created)}',
                            style: const TextStyle(color: Color(0xFFFDE68A), fontSize: 12.5),
                          ),
                        );
                      }),
                    ],
                  ),
                ),
              ),
            const SizedBox(height: 4),
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
                style: TextStyle(color: Color(0xFF6B7280)),
              ),
            ),
          ],
        );
      }
      return ListView.builder(
        padding: const EdgeInsets.only(bottom: 24),
        itemCount: state.searchResults.length,
        itemBuilder: (_, i) {
          final item = state.searchResults[i];
          final cap = item['capture'];
          final sim = ((item['similarity'] as double) * 100).round();
          return _buildNoteCard(context, cap, similarity: sim);
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
              style: TextStyle(color: Color(0xFF6B7280)),
            ),
          ),
        ],
      );
    }

    return ListView.builder(
      padding: const EdgeInsets.only(bottom: 24),
      itemCount: state.captures.length,
      itemBuilder: (_, i) {
        final cap = state.captures[i];
        return _buildNoteCard(context, cap);
      },
    );
  }

  Widget _buildNoteCard(BuildContext context, Map<String, dynamic> cap, {int? similarity}) {
    final title = (cap['title'] as String?)?.trim().isNotEmpty == true
        ? cap['title'] as String
        : generateAutoTitle(cap['body'] as String? ?? '');
    final body = cap['body'] as String? ?? '';
    final created = cap['created_at'] as int? ?? 0;
    final tags = (cap['tags'] as List?)?.cast<String>() ?? [];
    
    return Container(
      margin: const EdgeInsets.symmetric(horizontal: 12, vertical: 6),
      decoration: BoxDecoration(
        color: const Color(0xFF13151A),
        borderRadius: BorderRadius.circular(12),
        border: Border.all(color: const Color(0xFF1F2228), width: 1),
      ),
      child: InkWell(
        borderRadius: BorderRadius.circular(12),
        onTap: () => _showDetail(context, cap),
        child: Padding(
          padding: const EdgeInsets.all(16),
          child: Column(
            crossAxisAlignment: CrossAxisAlignment.start,
            children: [
              Row(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Expanded(
                    child: Text(
                      title,
                      style: const TextStyle(
                        color: Colors.white,
                        fontSize: 15,
                        fontWeight: FontWeight.bold,
                      ),
                      maxLines: 1,
                      overflow: TextOverflow.ellipsis,
                    ),
                  ),
                  const SizedBox(width: 8),
                  if (similarity != null)
                    Container(
                      padding: const EdgeInsets.symmetric(horizontal: 6, vertical: 2),
                      decoration: BoxDecoration(
                        color: const Color(0x1F8B5CF6),
                        border: Border.all(color: const Color(0x668B5CF6)),
                        borderRadius: BorderRadius.circular(6),
                      ),
                      child: Text(
                        '$similarity% match',
                        style: const TextStyle(
                          color: Color(0xFFA78BFA),
                          fontSize: 9,
                          fontWeight: FontWeight.bold,
                        ),
                      ),
                    ),
                ],
              ),
              const SizedBox(height: 8),
              Text(
                body,
                style: const TextStyle(
                  color: Color(0xFF9CA3AF),
                  fontSize: 13,
                  height: 1.45,
                ),
                maxLines: 2,
                overflow: TextOverflow.ellipsis,
              ),
              if (tags.isNotEmpty) ...[
                const SizedBox(height: 12),
                Wrap(
                  spacing: 6,
                  runSpacing: 6,
                  children: tags.map((t) => Container(
                    padding: const EdgeInsets.symmetric(horizontal: 8, vertical: 3),
                    decoration: BoxDecoration(
                      color: const Color(0xFF1F2228),
                      borderRadius: BorderRadius.circular(6),
                    ),
                    child: Text(
                      '#$t',
                      style: const TextStyle(
                        color: Color(0xFF9CA3AF),
                        fontSize: 10.5,
                        fontWeight: FontWeight.w500,
                      ),
                    ),
                  )).toList(),
                ),
              ],
              const SizedBox(height: 12),
              Row(
                mainAxisAlignment: MainAxisAlignment.spaceBetween,
                children: [
                  Row(
                    children: [
                      Icon(
                        _getTypeIcon(cap['type'] as String? ?? 'note'),
                        size: 13,
                        color: const Color(0xFF6B7280),
                      ),
                      const SizedBox(width: 4),
                      Text(
                        (cap['type'] as String? ?? 'note').toUpperCase(),
                        style: const TextStyle(
                          color: Color(0xFF6B7280),
                          fontSize: 9,
                          fontWeight: FontWeight.bold,
                          letterSpacing: 0.5,
                        ),
                      ),
                    ],
                  ),
                  Text(
                    formatRelativeTime(created),
                    style: const TextStyle(
                      color: Color(0xFF6B7280),
                      fontSize: 11,
                    ),
                  ),
                ],
              ),
            ],
          ),
        ),
      ),
    );
  }

  IconData _getTypeIcon(String type) {
    switch (type.toLowerCase()) {
      case 'link':
        return Icons.link_rounded;
      case 'qa':
        return Icons.question_answer_rounded;
      case 'code':
        return Icons.code_rounded;
      default:
        return Icons.description_rounded;
    }
  }

  void _showDetail(BuildContext context, Map<String, dynamic> cap) {
    showModalBottomSheet(
      context: context,
      isScrollControlled: true,
      backgroundColor: const Color(0xFF13151A),
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
      ),
      builder: (ctx) => Padding(
        padding: EdgeInsets.only(
          left: 20,
          right: 20,
          top: 24,
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
                    cap['title'] as String? ?? 'Note',
                    style: const TextStyle(color: Colors.white, fontSize: 18, fontWeight: FontWeight.bold),
                  ),
                ),
                IconButton(
                  icon: const Icon(Icons.edit_outlined, color: Color(0xFF8B5CF6)),
                  onPressed: () {
                    Navigator.pop(ctx);
                    _showEditForm(context, cap);
                  },
                ),
              ],
            ),
            const SizedBox(height: 16),
            const Divider(color: Color(0xFF1F2228), height: 1),
            const SizedBox(height: 16),
            SelectableText(
              cap['body'] as String? ?? '',
              style: const TextStyle(color: Color(0xFFE5E7EB), fontSize: 14, height: 1.5),
            ),
            const SizedBox(height: 28),
            OutlinedButton.icon(
              icon: const Icon(Icons.delete_outline_rounded),
              label: Text(cap['deleted_at'] != null && cap['deleted_at'] > 0
                  ? 'Delete permanently'
                  : 'Move to trash'),
              style: OutlinedButton.styleFrom(
                foregroundColor: const Color(0xFFEF4444),
                side: const BorderSide(color: Color(0x4DEF4444)),
                backgroundColor: const Color(0x0AEF4444),
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
      backgroundColor: const Color(0xFF13151A),
      shape: const RoundedRectangleBorder(
        borderRadius: BorderRadius.vertical(top: Radius.circular(16)),
      ),
      builder: (ctx) => StatefulBuilder(
        builder: (ctx, setModalState) => Padding(
          padding: EdgeInsets.only(
            left: 20,
            right: 20,
            top: 24,
            bottom: MediaQuery.of(ctx).viewInsets.bottom + 24,
          ),
          child: Column(
            mainAxisSize: MainAxisSize.min,
            crossAxisAlignment: CrossAxisAlignment.stretch,
            children: [
              const Text(
                'Edit note',
                style: TextStyle(color: Colors.white, fontSize: 16, fontWeight: FontWeight.bold),
              ),
              const SizedBox(height: 20),
              TextField(
                controller: bodyController,
                minLines: 3,
                maxLines: 8,
                decoration: const InputDecoration(
                  labelText: 'Body',
                  fillColor: Color(0xFF090A0C),
                ),
              ),
              const SizedBox(height: 12),
              TextField(
                controller: tagsController,
                decoration: const InputDecoration(
                  labelText: 'Tags (comma separated)',
                  hintText: 'ideas, work, notes',
                  fillColor: Color(0xFF090A0C),
                ),
              ),
              const SizedBox(height: 24),
              Row(
                mainAxisAlignment: MainAxisAlignment.end,
                children: [
                  TextButton(
                    onPressed: () => Navigator.pop(ctx),
                    child: const Text('Cancel', style: TextStyle(color: Color(0xFF6B7280))),
                  ),
                  const SizedBox(width: 12),
                  SizedBox(
                    height: 40,
                    child: DecoratedBox(
                      decoration: BoxDecoration(
                        borderRadius: BorderRadius.circular(8),
                        gradient: const LinearGradient(
                          colors: [Color(0xFF7C3AED), Color(0xFF8B5CF6)],
                        ),
                      ),
                      child: ElevatedButton(
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
                        style: ElevatedButton.styleFrom(
                          backgroundColor: Colors.transparent,
                          shadowColor: Colors.transparent,
                          shape: RoundedRectangleBorder(
                            borderRadius: BorderRadius.circular(8),
                          ),
                          padding: const EdgeInsets.symmetric(horizontal: 20),
                        ),
                        child: isSavingEdit
                            ? const SizedBox(
                                height: 16,
                                width: 16,
                                child: CircularProgressIndicator(strokeWidth: 2, color: Colors.white),
                              )
                            : const Text('Save'),
                      ),
                    ),
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
