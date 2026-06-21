import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import '../state/app_state.dart';

class VaultUnlockScreen extends StatefulWidget {
  const VaultUnlockScreen({super.key});

  @override
  State<VaultUnlockScreen> createState() => _VaultUnlockScreenState();
}

class _VaultUnlockScreenState extends State<VaultUnlockScreen> {
  final _pass = TextEditingController();
  final _confirm = TextEditingController();
  bool _setup = false;
  bool _obscure = true;

  @override
  void initState() {
    super.initState();
    _loadSetup();
  }

  Future<void> _loadSetup() async {
    final needs = await context.read<AppState>().needsSetup();
    if (mounted) setState(() => _setup = needs);
  }

  Future<void> _submit() async {
    final state = context.read<AppState>();
    final ok = _setup
        ? await state.setupVault(_pass.text, _confirm.text)
        : await state.unlockVault(_pass.text);
    if (ok && mounted) {
      Navigator.of(context).pushReplacementNamed('/inbox');
    }
  }

  @override
  Widget build(BuildContext context) {
    final state = context.watch<AppState>();

    return Scaffold(
      backgroundColor: const Color(0xFF0D1117),
      appBar: AppBar(
        title: Text(_setup ? 'Create vault passcode' : 'Unlock vault'),
        backgroundColor: const Color(0xFF161B22),
        actions: [
          IconButton(
            icon: const Icon(Icons.logout),
            onPressed: () async {
              await state.logout();
              if (context.mounted) {
                Navigator.of(context).pushReplacementNamed('/login');
              }
            },
          ),
        ],
      ),
      body: Padding(
        padding: const EdgeInsets.all(24),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            Text(
              _setup
                  ? 'Choose a passcode to encrypt your notes on this device. We cannot recover it if you forget.'
                  : 'Enter your vault passcode to decrypt your notes.',
              style: const TextStyle(color: Color(0xFF8B949E), height: 1.4),
            ),
            const SizedBox(height: 24),
            TextField(
              controller: _pass,
              obscureText: _obscure,
              decoration: InputDecoration(
                labelText: 'Passcode',
                suffixIcon: IconButton(
                  icon: Icon(_obscure ? Icons.visibility : Icons.visibility_off),
                  onPressed: () => setState(() => _obscure = !_obscure),
                ),
              ),
            ),
            if (_setup) ...[
              const SizedBox(height: 12),
              TextField(
                controller: _confirm,
                obscureText: _obscure,
                decoration: const InputDecoration(labelText: 'Confirm passcode'),
              ),
            ],
            if (state.error != null) ...[
              const SizedBox(height: 12),
              Text(state.error!, style: const TextStyle(color: Colors.redAccent)),
            ],
            const SizedBox(height: 24),
            FilledButton(
              onPressed: state.vaultLoading ? null : _submit,
              child: state.vaultLoading
                  ? const SizedBox(
                      height: 20,
                      width: 20,
                      child: CircularProgressIndicator(strokeWidth: 2),
                    )
                  : Text(_setup ? 'Create vault' : 'Unlock'),
            ),
          ],
        ),
      ),
    );
  }
}
