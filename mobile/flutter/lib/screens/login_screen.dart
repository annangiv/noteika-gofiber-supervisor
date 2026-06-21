import 'package:cookie_jar/cookie_jar.dart';
import 'package:flutter/material.dart';
import 'package:path_provider/path_provider.dart';
import 'package:provider/provider.dart';

import '../state/app_state.dart';
import 'webview_screen.dart';

class LoginScreen extends StatefulWidget {
  const LoginScreen({super.key});

  @override
  State<LoginScreen> createState() => _LoginScreenState();
}

class _LoginScreenState extends State<LoginScreen> {
  final _email = TextEditingController(text: 'dev-user@example.com');
  final _name = TextEditingController(text: 'Developer User');
  bool _loading = false;
  String? _error;

  Future<void> _devSignIn() async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final state = context.read<AppState>();
      await state.api.devLogin(
        email: _email.text.trim(),
        name: _name.text.trim(),
      );
      await state.bootstrap();
      if (mounted && state.isLoggedIn) {
        Navigator.of(context).pushReplacementNamed('/vault');
      }
    } catch (e) {
      if (mounted) {
        setState(() => _error = e.toString());
      }
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  Future<void> _startOAuth(String provider) async {
    setState(() {
      _loading = true;
      _error = null;
    });
    try {
      final state = context.read<AppState>();
      final url = '${state.api.baseUrl}/auth/login/$provider';
      final token = await Navigator.of(context).push<String>(
        MaterialPageRoute(
          builder: (_) => WebViewScreen(
            url: url,
            title: 'Sign in with ${provider == "github" ? "GitHub" : "Google"}',
          ),
        ),
      );

      if (token != null && token.isNotEmpty) {
        final dir = await getApplicationDocumentsDirectory();
        final jar = PersistCookieJar(storage: FileStorage('${dir.path}/.cookies/'));
        final uri = Uri.parse(state.api.baseUrl);
        final cookie = Cookie('keller_session', token)
          ..domain = uri.host
          ..path = '/';
        await jar.saveFromResponse(uri, [cookie]);

        await state.bootstrap();
        if (mounted && state.isLoggedIn) {
          Navigator.of(context).pushReplacementNamed('/vault');
        }
      }
    } catch (e) {
      setState(() => _error = 'OAuth login failed: $e');
    } finally {
      if (mounted) setState(() => _loading = false);
    }
  }

  @override
  void dispose() {
    _email.dispose();
    _name.dispose();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    return Scaffold(
      backgroundColor: const Color(0xFF0D1117),
      appBar: AppBar(
        title: const Text('Sign in to Noteika'),
        backgroundColor: const Color(0xFF161B22),
      ),
      body: SingleChildScrollView(
        padding: const EdgeInsets.all(24),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.stretch,
          children: [
            const Text(
              'Sign in with your provider:',
              style: TextStyle(color: Colors.white, fontSize: 16, fontWeight: FontWeight.bold),
            ),
            const SizedBox(height: 16),
            OutlinedButton.icon(
              icon: const Icon(Icons.code),
              label: const Text('Sign in with GitHub'),
              onPressed: _loading ? null : () => _startOAuth('github'),
              style: OutlinedButton.styleFrom(
                padding: const EdgeInsets.symmetric(vertical: 14),
                side: const BorderSide(color: Color(0xFF30363D)),
              ),
            ),
            const SizedBox(height: 12),
            OutlinedButton.icon(
              icon: const Icon(Icons.g_mobiledata_outlined, size: 28),
              label: const Text('Sign in with Google'),
              onPressed: _loading ? null : () => _startOAuth('google'),
              style: OutlinedButton.styleFrom(
                padding: const EdgeInsets.symmetric(vertical: 14),
                side: const BorderSide(color: Color(0xFF30363D)),
              ),
            ),
            const SizedBox(height: 32),
            const Row(
              children: [
                Expanded(child: Divider(color: Color(0xFF30363D))),
                Padding(
                  padding: EdgeInsets.symmetric(horizontal: 16),
                  child: Text('OR DEV SIGN IN', style: TextStyle(color: Color(0xFF8B949E), fontSize: 12)),
                ),
                Expanded(child: Divider(color: Color(0xFF30363D))),
              ],
            ),
            const SizedBox(height: 24),
            TextField(
              controller: _email,
              keyboardType: TextInputType.emailAddress,
              decoration: const InputDecoration(labelText: 'Email'),
            ),
            const SizedBox(height: 12),
            TextField(
              controller: _name,
              decoration: const InputDecoration(labelText: 'Name'),
            ),
            if (_error != null) ...[
              const SizedBox(height: 12),
              Text(_error!, style: const TextStyle(color: Colors.redAccent)),
            ],
            const SizedBox(height: 24),
            FilledButton(
              onPressed: _loading ? null : _devSignIn,
              child: _loading
                  ? const SizedBox(
                      height: 20,
                      width: 20,
                      child: CircularProgressIndicator(strokeWidth: 2),
                    )
                  : const Text('Dev sign in'),
            ),
          ],
        ),
      ),
    );
  }
}
