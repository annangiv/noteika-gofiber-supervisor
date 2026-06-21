import 'package:flutter/material.dart';
import 'package:provider/provider.dart';

import 'screens/inbox_screen.dart';
import 'screens/login_screen.dart';
import 'screens/vault_unlock_screen.dart';
import 'services/noteika_api.dart';
import 'state/app_state.dart';

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();
  final api = await NoteikaApi.create();
  runApp(NoteikaApp(api: api));
}

class NoteikaApp extends StatelessWidget {
  const NoteikaApp({super.key, required this.api});

  final NoteikaApi api;

  @override
  Widget build(BuildContext context) {
    return ChangeNotifierProvider(
      create: (_) => AppState(api)..bootstrap(),
      child: MaterialApp(
        title: 'Noteika',
        debugShowCheckedModeBanner: false,
        theme: ThemeData(
          brightness: Brightness.dark,
          colorScheme: ColorScheme.fromSeed(
            seedColor: const Color(0xFF58A6FF),
            brightness: Brightness.dark,
            surface: const Color(0xFF161B22),
          ),
          scaffoldBackgroundColor: const Color(0xFF0D1117),
          inputDecorationTheme: InputDecorationTheme(
            border: OutlineInputBorder(borderRadius: BorderRadius.circular(8)),
            filled: true,
            fillColor: const Color(0xFF0D1117),
          ),
        ),
        home: const _RootGate(),
        routes: {
          '/login': (_) => const LoginScreen(),
          '/vault': (_) => const VaultUnlockScreen(),
          '/inbox': (_) => const InboxScreen(),
        },
      ),
    );
  }
}

class _RootGate extends StatelessWidget {
  const _RootGate();

  @override
  Widget build(BuildContext context) {
    final state = context.watch<AppState>();

    if (state.authLoading) {
      return const Scaffold(
        body: Center(child: CircularProgressIndicator()),
      );
    }
    if (!state.isLoggedIn) return const LoginScreen();
    if (!state.isUnlocked) return const VaultUnlockScreen();
    return const InboxScreen();
  }
}
