import 'dart:convert';
import 'dart:io';

import 'package:flutter/material.dart';
import 'package:path_provider/path_provider.dart';
import 'package:provider/provider.dart';
import 'package:url_launcher/url_launcher.dart';

import '../state/app_state.dart';

class AccountScreen extends StatefulWidget {
  const AccountScreen({super.key});

  @override
  State<AccountScreen> createState() => _AccountScreenState();
}

class _AccountScreenState extends State<AccountScreen> {
  int _searchMinPct = 70;
  bool _settingsSaving = false;
  bool _settingsSaved = false;

  bool _billingLoading = false;
  String? _billingError;

  bool _exporting = false;

  bool _confirmDelete = false;
  final _deleteController = TextEditingController();
  bool _deleting = false;

  @override
  void initState() {
    super.initState();
    final user = context.read<AppState>().user;
    if (user != null) {
      final sim = user['search_min_similarity'] ??
          user['effective_search_min_similarity'] ??
          0.70;
      _searchMinPct = (sim * 100).round();
    }
  }

  @override
  void dispose() {
    _deleteController.dispose();
    super.dispose();
  }

  String _searchSensitivityHint(int pct) {
    if (pct <= 55) return 'Broad — may show loosely related notes. Good if search feels too strict.';
    if (pct <= 69) return 'Balanced — wider net than default; some maybe-results may appear.';
    if (pct <= 74) return 'Focused — recommended default. Strong matches without much noise.';
    if (pct <= 79) return 'Strict — only closely related notes. May miss reworded captures.';
    return 'Near-duplicate — almost identical text only. Best for deduping, not discovery.';
  }

  Future<void> _saveSettings() async {
    setState(() {
      _settingsSaving = true;
      _settingsSaved = false;
    });
    try {
      final state = context.read<AppState>();
      await state.changeSearchMinSimilarity(_searchMinPct / 100.0);
      setState(() => _settingsSaved = true);
      Future.delayed(const Duration(seconds: 3), () {
        if (mounted) setState(() => _settingsSaved = false);
      });
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            backgroundColor: const Color(0xFF13151A),
            content: Text('Failed to save settings: $e', style: const TextStyle(color: Color(0xFFFE4A49))),
          ),
        );
      }
    } finally {
      if (mounted) setState(() => _settingsSaving = false);
    }
  }

  Future<void> _manageBilling() async {
    setState(() {
      _billingLoading = true;
      _billingError = null;
    });
    try {
      final state = context.read<AppState>();
      final isPro = state.user?['pro_access'] == true;
      final res = isPro
          ? await state.api.openBillingPortal()
          : await state.api.checkoutStripe();

      final urlStr = res['url'] as String?;
      if (urlStr != null) {
        final url = Uri.parse(urlStr);
        if (await canLaunchUrl(url)) {
          await launchUrl(url, mode: LaunchMode.externalApplication);
        } else {
          throw Exception('Could not launch URL.');
        }
      } else {
        throw Exception('Server did not return portal/checkout URL.');
      }
    } catch (e) {
      setState(() => _billingError = e.toString());
    } finally {
      setState(() => _billingLoading = false);
    }
  }

  Future<void> _exportData() async {
    setState(() => _exporting = true);
    try {
      final state = context.read<AppState>();
      final data = await state.api.exportData();
      final jsonText = const JsonEncoder.withIndent('  ').convert(data);
      
      final dir = await getApplicationDocumentsDirectory();
      final file = File('${dir.path}/noteika-export.json');
      await file.writeAsString(jsonText);

      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            backgroundColor: const Color(0xFF13151A),
            content: Text(
              'Export saved to documents: noteika-export.json (${file.lengthSync()} bytes)',
              style: const TextStyle(color: Color(0xFFA78BFA)),
            ),
          ),
        );
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            backgroundColor: const Color(0xFF13151A),
            content: Text('Export failed: $e', style: const TextStyle(color: Color(0xFFFE4A49))),
          ),
        );
      }
    } finally {
      if (mounted) setState(() => _exporting = false);
    }
  }

  Future<void> _deleteAccount() async {
    if (_deleteController.text != 'DELETE') return;
    setState(() => _deleting = true);
    try {
      final state = context.read<AppState>();
      await state.api.deleteAccount();
      await state.logout();
      if (mounted) {
        Navigator.of(context).pushReplacementNamed('/login');
      }
    } catch (e) {
      if (mounted) {
        ScaffoldMessenger.of(context).showSnackBar(
          SnackBar(
            backgroundColor: const Color(0xFF13151A),
            content: Text('Delete failed: $e', style: const TextStyle(color: Color(0xFFFE4A49))),
          ),
        );
      }
    } finally {
      if (mounted) setState(() => _deleting = false);
    }
  }

  @override
  Widget build(BuildContext context) {
    final state = context.watch<AppState>();
    final u = state.user;
    final isPro = u?['pro_access'] == true;
    final noteCount = u?['capture_count'] ?? 0;
    final noteLimit = u?['capture_limit'] ?? 10;
    final createdAtEpoch = u?['created_at'] as int?;
    final memberSince = createdAtEpoch != null
        ? DateTime.fromMillisecondsSinceEpoch(createdAtEpoch * 1000)
        : null;

    return Scaffold(
      appBar: AppBar(
        title: const Text('Your Account'),
        leading: IconButton(
          icon: const Icon(Icons.arrow_back_ios_new, size: 18),
          onPressed: () => Navigator.pop(context),
        ),
      ),
      body: SingleChildScrollView(
        padding: const EdgeInsets.symmetric(horizontal: 20, vertical: 24),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            const Text(
              'ACCOUNT',
              style: TextStyle(
                color: Color(0xFFA78BFA),
                fontWeight: FontWeight.w600,
                letterSpacing: 1.5,
                fontSize: 12,
              ),
            ),
            const SizedBox(height: 8),
            const Text(
              'Settings & Profile',
              style: TextStyle(
                color: Colors.white,
                fontWeight: FontWeight.bold,
                fontSize: 28,
                letterSpacing: -0.5,
              ),
            ),
            const SizedBox(height: 24),

            // Profile Card
            _buildCard(
              title: 'Profile Details',
              child: Column(
                children: [
                  _buildProfileRow('Name', u?['full_name'] ?? '—'),
                  const Divider(color: Color(0xFF1F2228), height: 1),
                  _buildProfileRow('Email', u?['email'] ?? '—'),
                  const Divider(color: Color(0xFF1F2228), height: 1),
                  _buildProfileRow('Plan', isPro ? 'Pro Access' : 'Free tier'),
                  const Divider(color: Color(0xFF1F2228), height: 1),
                  _buildProfileRow(
                    'Notes saved',
                    isPro ? '$noteCount (unlimited)' : '$noteCount / $noteLimit',
                  ),
                  if (memberSince != null) ...[
                    const Divider(color: Color(0xFF1F2228), height: 1),
                    _buildProfileRow(
                      'Member since',
                      '${memberSince.month}/${memberSince.day}/${memberSince.year}',
                    ),
                  ],
                ],
              ),
            ),
            const SizedBox(height: 20),

            // Billing Card
            _buildCard(
              title: 'Billing & Plan Upgrade',
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  Text(
                    isPro
                        ? 'Thank you for supporting Noteika! You have active Pro access with unlimited encrypted captures and semantic search.'
                        : 'Free plan includes a limit of $noteLimit encrypted captures. Upgrade to Pro to support development and save unlimited notes.',
                    style: const TextStyle(color: Color(0xFF9CA3AF), fontSize: 14, height: 1.45),
                  ),
                  const SizedBox(height: 20),
                  if (_billingError != null) ...[
                    Container(
                      padding: const EdgeInsets.all(12),
                      margin: const EdgeInsets.only(bottom: 12),
                      decoration: BoxDecoration(
                        color: const Color(0x1FFE4A49),
                        borderRadius: BorderRadius.circular(10),
                        border: Border.all(color: const Color(0x66FE4A49)),
                      ),
                      child: Text(
                        _billingError!,
                        style: const TextStyle(color: Color(0xFFFE4A49), fontSize: 13),
                      ),
                    ),
                  ],
                  if (u?['stripe_enabled'] == true)
                    ElevatedButton(
                      onPressed: _billingLoading ? null : _manageBilling,
                      style: ElevatedButton.styleFrom(
                        backgroundColor: isPro ? const Color(0xFF1F2228) : const Color(0xFF8B5CF6),
                        foregroundColor: Colors.white,
                        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(10)),
                        padding: const EdgeInsets.symmetric(vertical: 14),
                      ),
                      child: _billingLoading
                          ? const SizedBox(
                              height: 18,
                              width: 18,
                              child: CircularProgressIndicator(strokeWidth: 2, color: Colors.white),
                            )
                          : Text(isPro ? 'Manage subscription' : 'Upgrade to Pro — \$8/mo'),
                    )
                  else
                    const Text(
                      'Billing details are not configured on this server.',
                      style: TextStyle(color: Color(0xFF6B7280), fontSize: 13, fontStyle: FontStyle.italic),
                    ),
                ],
              ),
            ),
            const SizedBox(height: 20),

            // Search Sensitivity Card
            _buildCard(
              title: 'Search Match Sensitivity',
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  const Text(
                    'Adjust how closely note contents must dynamically match semantic queries to display in FTS search.',
                    style: TextStyle(color: Color(0xFF9CA3AF), fontSize: 14, height: 1.45),
                  ),
                  const SizedBox(height: 20),
                  Row(
                    mainAxisAlignment: MainAxisAlignment.spaceBetween,
                    children: [
                      const Text(
                        'Minimum match accuracy:',
                        style: TextStyle(color: Colors.white, fontSize: 14, fontWeight: FontWeight.w500),
                      ),
                      Text(
                        '$_searchMinPct%',
                        style: const TextStyle(
                          color: Color(0xFFA78BFA),
                          fontWeight: FontWeight.bold,
                          fontSize: 16,
                        ),
                      ),
                    ],
                  ),
                  const SizedBox(height: 8),
                  SliderTheme(
                    data: SliderTheme.of(context).copyWith(
                      activeTrackColor: const Color(0xFF8B5CF6),
                      inactiveTrackColor: const Color(0xFF1F2228),
                      thumbColor: const Color(0xFF8B5CF6),
                      overlayColor: const Color(0xFF8B5CF6).withOpacity(0.15),
                      activeTickMarkColor: Colors.transparent,
                      inactiveTickMarkColor: Colors.transparent,
                      trackHeight: 4,
                    ),
                    child: Slider(
                      value: _searchMinPct.toDouble(),
                      min: 50,
                      max: 85,
                      divisions: 7,
                      onChanged: (val) {
                        setState(() => _searchMinPct = val.round());
                      },
                    ),
                  ),
                  Padding(
                    padding: const EdgeInsets.symmetric(horizontal: 8.0),
                    child: Row(
                      mainAxisAlignment: MainAxisAlignment.spaceBetween,
                      children: const [
                        Text('50% broad', style: TextStyle(color: Color(0xFF6B7280), fontSize: 11)),
                        Text('70% default', style: TextStyle(color: Color(0xFF6B7280), fontSize: 11)),
                        Text('85% strict', style: TextStyle(color: Color(0xFF6B7280), fontSize: 11)),
                      ],
                    ),
                  ),
                  const SizedBox(height: 16),
                  Container(
                    padding: const EdgeInsets.all(12),
                    decoration: BoxDecoration(
                      color: const Color(0xFF0A0B0D),
                      borderRadius: BorderRadius.circular(8),
                      border: Border.all(color: const Color(0xFF1F2228)),
                    ),
                    child: Text(
                      _searchSensitivityHint(_searchMinPct),
                      style: const TextStyle(
                        color: Color(0xFF9CA3AF),
                        fontSize: 13,
                        height: 1.4,
                      ),
                    ),
                  ),
                  const SizedBox(height: 20),
                  Row(
                    mainAxisAlignment: MainAxisAlignment.end,
                    children: [
                      TextButton(
                        onPressed: () => setState(() => _searchMinPct = 70),
                        child: const Text('Reset to 70%'),
                      ),
                      const SizedBox(width: 12),
                      ElevatedButton(
                        onPressed: _settingsSaving ? null : _saveSettings,
                        style: ElevatedButton.styleFrom(
                          backgroundColor: const Color(0xFF8B5CF6),
                          padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
                          shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(10)),
                        ),
                        child: _settingsSaving
                            ? const SizedBox(
                                height: 16,
                                width: 16,
                                child: CircularProgressIndicator(strokeWidth: 2, color: Colors.white),
                              )
                            : Text(_settingsSaved ? 'Saved!' : 'Save settings'),
                      ),
                    ],
                  ),
                ],
              ),
            ),
            const SizedBox(height: 20),

            // Data Card
            _buildCard(
              title: 'Data Export & Backup',
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  const Text(
                    'Export all encrypted captures locally as formatted JSON at any time.',
                    style: TextStyle(color: Color(0xFF9CA3AF), fontSize: 14, height: 1.45),
                  ),
                  const SizedBox(height: 16),
                  OutlinedButton.icon(
                    icon: const Icon(Icons.download_rounded, size: 18),
                    label: Text(_exporting ? 'Preparing export...' : 'Export local JSON backup'),
                    onPressed: _exporting ? null : _exportData,
                    style: OutlinedButton.styleFrom(
                      side: const BorderSide(color: Color(0xFF1F2228)),
                      shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(10)),
                      padding: const EdgeInsets.symmetric(vertical: 14),
                    ),
                  ),
                ],
              ),
            ),
            const SizedBox(height: 20),

            // Danger Zone Card
            _buildCard(
              title: 'Danger Zone',
              isDanger: true,
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  const Text(
                    'Permanently delete your account and remove all captures. This action is immediate, final, and cannot be undone.',
                    style: TextStyle(color: Color(0xFF9CA3AF), fontSize: 14, height: 1.45),
                  ),
                  const SizedBox(height: 16),
                  if (!_confirmDelete)
                    ElevatedButton(
                      onPressed: () => setState(() => _confirmDelete = true),
                      style: ElevatedButton.styleFrom(
                        backgroundColor: const Color(0xFFEF4444),
                        foregroundColor: Colors.white,
                        shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(10)),
                        padding: const EdgeInsets.symmetric(vertical: 14),
                      ),
                      child: const Text('Delete my account'),
                    )
                  else ...[
                    const Text(
                      'Type DELETE to confirm account deletion:',
                      style: TextStyle(color: Colors.white, fontSize: 13, fontWeight: FontWeight.bold),
                    ),
                    const SizedBox(height: 8),
                    TextField(
                      controller: _deleteController,
                      decoration: const InputDecoration(
                        hintText: 'DELETE',
                        fillColor: Color(0xFF0A0B0D),
                      ),
                      onChanged: (_) => setState(() {}),
                    ),
                    const SizedBox(height: 12),
                    Row(
                      mainAxisAlignment: MainAxisAlignment.end,
                      children: [
                        TextButton(
                          onPressed: () {
                            setState(() {
                              _confirmDelete = false;
                              _deleteController.clear();
                            });
                          },
                          child: const Text('Cancel', style: TextStyle(color: Color(0xFF9CA3AF))),
                        ),
                        const SizedBox(width: 12),
                        ElevatedButton(
                          onPressed: _deleteController.text != 'DELETE' || _deleting
                              ? null
                              : _deleteAccount,
                          style: ElevatedButton.styleFrom(
                            backgroundColor: const Color(0xFFEF4444),
                            foregroundColor: Colors.white,
                            disabledBackgroundColor: const Color(0xFF1F2228),
                            disabledForegroundColor: const Color(0xFF6B7280),
                            shape: RoundedRectangleBorder(borderRadius: BorderRadius.circular(10)),
                            padding: const EdgeInsets.symmetric(horizontal: 16, vertical: 12),
                          ),
                          child: _deleting
                              ? const SizedBox(
                                  height: 16,
                                  width: 16,
                                  child: CircularProgressIndicator(strokeWidth: 2, color: Colors.white),
                                )
                              : const Text('Permanently delete'),
                        ),
                      ],
                    ),
                  ],
                ],
              ),
            ),
            const SizedBox(height: 32),
          ],
        ),
      ),
    );
  }

  Widget _buildCard({
    required String title,
    required Widget child,
    bool isDanger = false,
  }) {
    return Container(
      padding: const EdgeInsets.all(20),
      decoration: BoxDecoration(
        color: const Color(0xFF13151A),
        borderRadius: BorderRadius.circular(12),
        border: Border.all(
          color: isDanger ? const Color(0xFFEF4444).withOpacity(0.3) : const Color(0xFF1F2228),
        ),
      ),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          Text(
            title,
            style: TextStyle(
              color: isDanger ? const Color(0xFFEF4444) : Colors.white,
              fontWeight: FontWeight.bold,
              fontSize: 16,
              letterSpacing: -0.3,
            ),
          ),
          const SizedBox(height: 16),
          child,
        ],
      ),
    );
  }

  Widget _buildProfileRow(String label, String value) {
    return Padding(
      padding: const EdgeInsets.symmetric(vertical: 12),
      child: Row(
        crossAxisAlignment: CrossAxisAlignment.start,
        children: [
          SizedBox(
            width: 110,
            child: Text(
              label,
              style: const TextStyle(color: Color(0xFF9CA3AF), fontSize: 14, fontWeight: FontWeight.w500),
            ),
          ),
          Expanded(
            child: Text(
              value,
              style: const TextStyle(color: Colors.white, fontSize: 14, fontWeight: FontWeight.w500),
            ),
          ),
        ],
      ),
    );
  }
}
