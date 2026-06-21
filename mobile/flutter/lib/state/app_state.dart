import 'dart:typed_data';

import 'package:cryptography/cryptography.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';

import '../services/noteika_api.dart';
import '../services/vault_crypto.dart';

class AppState extends ChangeNotifier {
  AppState(this.api);

  final NoteikaApi api;
  final _storage = const FlutterSecureStorage();

  static const _vaultSetupKey = 'noteika_vault_setup_done';

  bool authLoading = true;
  bool vaultLoading = false;
  Map<String, dynamic>? user;
  SecretKey? vaultKey;
  Uint8List? salt;
  String? error;
  List<Map<String, dynamic>> captures = [];
  List<ProjectRef> projects = [NoteikaApi.inbox];
  String selectedProjectId = 'inbox';

  String get selectedProjectName {
    for (final p in projects) {
      if (p.id == selectedProjectId) return p.name;
    }
    return 'Inbox';
  }

  bool get isLoggedIn => user != null;
  bool get isUnlocked => vaultKey != null;

  Future<void> bootstrap() async {
    authLoading = true;
    notifyListeners();
    user = await api.currentUser();
    authLoading = false;
    notifyListeners();
  }

  Future<bool> needsSetup() async {
    final v = await _storage.read(key: _vaultSetupKey);
    return v != '1';
  }

  Future<void> markVaultSetup() async {
    await _storage.write(key: _vaultSetupKey, value: '1');
  }

  Future<bool> unlockVault(String passcode) async {
    vaultLoading = true;
    error = null;
    notifyListeners();
    try {
      salt ??= await api.fetchVaultSalt();
      vaultKey = await VaultCrypto.deriveVaultKey(passcode, salt!);
      await loadData();
      vaultLoading = false;
      notifyListeners();
      return true;
    } catch (e) {
      error = e.toString();
      vaultKey = null;
      vaultLoading = false;
      notifyListeners();
      return false;
    }
  }

  Future<bool> setupVault(String passcode, String confirm) async {
    if (passcode.length < 8) {
      error = 'Passcode must be at least 8 characters';
      notifyListeners();
      return false;
    }
    if (passcode != confirm) {
      error = 'Passcodes do not match';
      notifyListeners();
      return false;
    }
    final ok = await unlockVault(passcode);
    if (ok) await markVaultSetup();
    return ok;
  }

  void lockVault() {
    vaultKey = null;
    captures = [];
    notifyListeners();
  }

  Future<void> logout() async {
    await api.logout();
    user = null;
    lockVault();
    notifyListeners();
  }

  Future<void> loadData() async {
    if (vaultKey == null) return;
    projects = await api.listProjects(vaultKey!);
    captures = await api.listCaptures(
      projectId: selectedProjectId,
      vaultKey: vaultKey,
    );
    if (selectedProjectId != 'trash') {
      captures = captures.where((c) => (c['deleted_at'] ?? 0) == 0).toList();
    }
    captures.sort((a, b) => (b['created_at'] as int).compareTo(a['created_at'] as int));
    notifyListeners();
  }

  Future<void> setProject(String projectId) async {
    selectedProjectId = projectId;
    await loadData();
  }

  Future<void> saveCapture(String body) async {
    if (vaultKey == null) throw StateError('Vault locked');
    await api.createCapture(
      vaultKey: vaultKey!,
      body: body,
      projectName: selectedProjectName,
    );
    await loadData();
  }

  Future<void> deleteCapture(String id) async {
    await api.deleteCapture(id);
    await loadData();
  }
}
