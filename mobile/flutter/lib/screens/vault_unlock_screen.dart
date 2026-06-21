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
      backgroundColor: const Color(0xFF090A0C),
      appBar: AppBar(
        title: Text(_setup ? 'Create Vault' : 'Unlock Vault'),
        backgroundColor: const Color(0xFF090A0C),
        actions: [
          IconButton(
            icon: const Icon(Icons.logout_rounded),
            tooltip: 'Sign Out',
            onPressed: () async {
              await state.logout();
              if (context.mounted) {
                Navigator.of(context).pushReplacementNamed('/login');
              }
            },
          ),
        ],
      ),
      body: SingleChildScrollView(
        padding: const EdgeInsets.symmetric(horizontal: 24, vertical: 16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            const SizedBox(height: 32),
            // Lock Icon Header
            Center(
              child: Container(
                height: 80,
                width: 80,
                decoration: BoxDecoration(
                  color: const Color(0xFF8B5CF6).withOpacity(0.06),
                  shape: BoxShape.circle,
                  border: Border.all(
                    color: const Color(0xFF8B5CF6).withOpacity(0.2),
                    width: 2,
                  ),
                ),
                child: const Icon(
                  Icons.lock_outline_rounded,
                  size: 38,
                  color: Color(0xFF8B5CF6),
                ),
              ),
            ),
            const SizedBox(height: 32),
            Text(
              _setup ? 'Secure Your Notes' : 'Enter Vault Passcode',
              style: const TextStyle(color: Colors.white, fontSize: 20, fontWeight: FontWeight.bold),
              textAlign: TextAlign.center,
            ),
            const SizedBox(height: 12),
            Padding(
              padding: const EdgeInsets.symmetric(horizontal: 16),
              child: Text(
                _setup
                    ? 'Choose a passcode to encrypt your notes on this device. We cannot recover it if you forget.'
                    : 'Enter your vault passcode to decrypt your notes.',
                style: const TextStyle(color: Color(0xFF9CA3AF), height: 1.45, fontSize: 13.5),
                textAlign: TextAlign.center,
              ),
            ),
            const SizedBox(height: 32),
            TextField(
              controller: _pass,
              obscureText: _obscure,
              decoration: InputDecoration(
                labelText: 'Passcode',
                suffixIcon: IconButton(
                  icon: Icon(_obscure ? Icons.visibility : Icons.visibility_off, color: const Color(0xFF9CA3AF)),
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
              const SizedBox(height: 16),
              Text(
                state.error!,
                style: const TextStyle(color: Colors.redAccent, fontSize: 13.5),
                textAlign: TextAlign.center,
              ),
            ],
            const SizedBox(height: 32),
            SizedBox(
              height: 50,
              child: DecoratedBox(
                decoration: BoxDecoration(
                  borderRadius: BorderRadius.circular(10),
                  gradient: const LinearGradient(
                    colors: [Color(0xFF7C3AED), Color(0xFF8B5CF6)],
                  ),
                  boxShadow: [
                    BoxShadow(
                      color: const Color(0xFF7C3AED).withOpacity(0.35),
                      blurRadius: 14,
                      offset: const Offset(0, 4),
                    ),
                  ],
                ),
                child: ElevatedButton(
                  onPressed: state.vaultLoading ? null : _submit,
                  style: ElevatedButton.styleFrom(
                    backgroundColor: Colors.transparent,
                    shadowColor: Colors.transparent,
                    shape: RoundedRectangleBorder(
                      borderRadius: BorderRadius.circular(10),
                    ),
                  ),
                  child: state.vaultLoading
                      ? const SizedBox(
                          height: 18,
                          width: 18,
                          child: CircularProgressIndicator(strokeWidth: 2, color: Colors.white),
                        )
                      : Text(_setup ? 'Create Vault' : 'Unlock'),
                ),
              ),
            ),
          ],
        ),
      ),
    );
  }
}
