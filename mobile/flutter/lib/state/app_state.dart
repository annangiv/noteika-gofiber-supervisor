import 'dart:math' as math;

import 'package:cryptography/cryptography.dart';
import 'package:flutter/foundation.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';

import '../services/noteika_api.dart';
import '../services/vault_crypto.dart';
import '../services/fingerprint_service.dart';
import '../services/embedding_service.dart';
import '../services/iap_service.dart';
import '../utils/capture_content.dart';

class AppState extends ChangeNotifier {
  AppState(this.api) {
    iapService = IapService(this);
  }

  final NoteikaApi api;
  late final IapService iapService;
  final _storage = const FlutterSecureStorage();

  static const _vaultSetupKey = 'noteika_vault_setup_done';
  static const _hasSeenLandingKey = 'noteika_has_seen_landing';

  final embeddingService = EmbeddingService();
  final fingerprintService = FingerprintService();

  bool authLoading = true;
  bool vaultLoading = false;
  Map<String, dynamic>? user;
  SecretKey? vaultKey;
  Uint8List? salt;
  String? error;
  List<Map<String, dynamic>> captures = [];
  List<ProjectRef> projects = [NoteikaApi.inbox];
  String selectedProjectId = 'inbox';
  bool hasSeenLanding = false;

  // Embedding model states
  bool modelDownloading = false;
  double downloadProgress = 0.0;
  bool modelLoaded = false;
  String? modelError;

  // Search states
  String searchQuery = '';
  List<Map<String, dynamic>> searchResults = [];
  bool isSearching = false;

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
    final v = await _storage.read(key: _hasSeenLandingKey);
    hasSeenLanding = v == '1';

    if (!kIsWeb) {
      iapService.initialize();
    }

    authLoading = false;
    notifyListeners();
  }

  Future<void> markLandingSeen() async {
    await _storage.write(key: _hasSeenLandingKey, value: '1');
    hasSeenLanding = true;
    notifyListeners();
  }

  Future<void> resetLandingState() async {
    await _storage.delete(key: _hasSeenLandingKey);
    hasSeenLanding = false;
    notifyListeners();
  }

  void notify() {
    notifyListeners();
  }

  Future<bool> needsSetup() async {
    final v = await _storage.read(key: _vaultSetupKey);
    return v != '1';
  }

  Future<void> markVaultSetup() async {
    await _storage.write(key: _vaultSetupKey, value: '1');
  }

  Future<void> initEmbeddingModel() async {
    final exists = await embeddingService.checkModelExists();
    if (!exists) {
      modelDownloading = true;
      downloadProgress = 0.0;
      notifyListeners();
    }
    try {
      await embeddingService.downloadAndLoadModel((p) {
        downloadProgress = p;
        notifyListeners();
      });
      modelDownloading = false;
      modelLoaded = true;
      modelError = null;
      notifyListeners();
    } catch (e) {
      modelDownloading = false;
      modelError = e.toString();
      notifyListeners();
    }
  }

  Future<bool> unlockVault(String passcode) async {
    vaultLoading = true;
    error = null;
    notifyListeners();
    try {
      salt ??= await api.fetchVaultSalt();
      vaultKey = await VaultCrypto.deriveVaultKey(passcode, salt!);
      
      // Derive fingerprint matrix in FingerprintService
      await fingerprintService.deriveAndSetMatrix(vaultKey!);

      // Load or download embedding model asynchronously in background
      initEmbeddingModel();

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
    searchResults = [];
    searchQuery = '';
    isSearching = false;
    fingerprintService.clear();
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

  String buildCaptureEmbeddingText(String body, List<String> tags) {
    final title = generateAutoTitle(body);
    final merged = mergeTags([tags, parseHashtags(body)]);
    final parts = <String>[];
    if (merged.isNotEmpty) parts.add(merged.join(' '));
    if (title.isNotEmpty) parts.add(title);
    if (body.isNotEmpty) parts.add(body);
    return parts.join('\n');
  }

  Future<void> saveCapture(String body, {List<String> tags = const []}) async {
    if (vaultKey == null) throw StateError('Vault locked');
    
    String? fingerprint;
    String? encryptedVector;

    if (modelLoaded) {
      try {
        final embedText = buildCaptureEmbeddingText(body, tags);
        final embedding = await embeddingService.embedText(embedText, isQuery: false);
        fingerprint = fingerprintService.fingerprintEmbeddingB64(embedding);
        encryptedVector = await VaultCrypto.encryptEmbedding(vaultKey!, embedding);
      } catch (e) {
        debugPrint('Save embedding error: $e');
      }
    }

    await api.createCapture(
      vaultKey: vaultKey!,
      body: body,
      projectName: selectedProjectName,
      tags: tags,
      fingerprint: fingerprint,
      encryptedVector: encryptedVector,
    );
    await loadData();
  }

  Future<void> editCapture(String id, String body, {List<String> tags = const []}) async {
    if (vaultKey == null) throw StateError('Vault locked');

    String? fingerprint;
    String? encryptedVector;

    if (modelLoaded) {
      try {
        final embedText = buildCaptureEmbeddingText(body, tags);
        final embedding = await embeddingService.embedText(embedText, isQuery: false);
        fingerprint = fingerprintService.fingerprintEmbeddingB64(embedding);
        encryptedVector = await VaultCrypto.encryptEmbedding(vaultKey!, embedding);
      } catch (e) {
        debugPrint('Save embedding error: $e');
      }
    }

    await api.updateCapture(
      id: id,
      vaultKey: vaultKey!,
      body: body,
      projectName: selectedProjectName,
      tags: tags,
      fingerprint: fingerprint,
      encryptedVector: encryptedVector,
    );
    await loadData();
  }

  Future<void> deleteCapture(String id) async {
    await api.deleteCapture(id);
    await loadData();
  }

  Future<void> changeSearchMinSimilarity(double value) async {
    await api.saveSettings(searchMinSimilarity: value);
    if (user != null) {
      user!['search_min_similarity'] = value;
      notifyListeners();
    }
  }

  Future<List<Map<String, dynamic>>> findDuplicates(String text) async {
    if (text.trim().isEmpty || vaultKey == null) return [];
    final embedText = buildCaptureEmbeddingText(text, []);

    // Jaccard similarity local fallback
    final textHits = <Map<String, dynamic>>[];
    for (final cap in captures) {
      final compText = FingerprintService.captureComparableText(cap);
      final sim = FingerprintService.textSimilarity(embedText, compText);
      if (sim >= 0.65) {
        textHits.add({'capture': cap, 'similarity': sim});
      }
    }

    if (!modelLoaded) {
      textHits.sort((a, b) => (b['similarity'] as double).compareTo(a['similarity'] as double));
      return textHits.take(5).toList();
    }

    try {
      final queryEmb = await embeddingService.embedText(embedText, isQuery: false);
      final queryFp = fingerprintService.fingerprintEmbeddingB64(queryEmb);
      final candidates = await api.searchCaptures(
        queryFingerprint: queryFp,
        projectId: selectedProjectId == 'trash' ? '' : selectedProjectId,
        limit: 30,
      );

      final rescored = <Map<String, dynamic>>[];
      for (final item in candidates) {
        final capture = await VaultCrypto.decryptCaptureRecord(
          vaultKey!,
          Map<String, dynamic>.from(item['capture'] as Map),
        );
        final encVector = item['capture']['encrypted_vector'] as String?;
        List<double>? candidateEmb;
        if (encVector != null && encVector.isNotEmpty) {
          candidateEmb = await VaultCrypto.decryptEmbedding(vaultKey!, encVector);
        }
        if (candidateEmb == null || candidateEmb.isEmpty) {
          final noteText = buildCaptureEmbeddingText(
            capture['body'] as String? ?? '',
            (capture['tags'] as List?)?.cast<String>() ?? [],
          );
          candidateEmb = await embeddingService.embedText(noteText, isQuery: false);
        }
        final similarity = FingerprintService.cosineSimilarity(queryEmb, candidateEmb);
        if (similarity >= 0.65) {
          rescored.add({'capture': capture, 'similarity': similarity});
        }
      }

      final merged = <String, Map<String, dynamic>>{};
      for (final hit in [...rescored, ...textHits]) {
        final id = hit['capture']['id'] as String;
        final existing = merged[id];
        if (existing == null || (hit['similarity'] as double) > (existing['similarity'] as double)) {
          merged[id] = hit;
        }
      }

      final sorted = merged.values.toList();
      sorted.sort((a, b) => (b['similarity'] as double).compareTo(a['similarity'] as double));
      return sorted.take(5).toList();
    } catch (_) {
      textHits.sort((a, b) => (b['similarity'] as double).compareTo(a['similarity'] as double));
      return textHits.take(5).toList();
    }
  }

  Future<void> search(String query) async {
    searchQuery = query;
    if (query.trim().isEmpty) {
      searchResults = [];
      isSearching = false;
      notifyListeners();
      return;
    }

    isSearching = true;
    notifyListeners();

    final searchMinSimilarity =
        (user?['search_min_similarity'] as num?)?.toDouble() ??
        (user?['effective_search_min_similarity'] as num?)?.toDouble() ??
        0.70;

    if (!modelLoaded) {
      // Fallback: Local Full Text Search
      final allRes = await api.listCaptures(vaultKey: vaultKey);
      searchResults = _ftsSearch(query, allRes);
      searchResults = searchResults
          .where((x) => (x['similarity'] as double) >= searchMinSimilarity)
          .toList();
      notifyListeners();
      return;
    }

    try {
      final queryEmb = await embeddingService.embedText(query, isQuery: true);
      final queryFp = fingerprintService.fingerprintEmbeddingB64(queryEmb);
      final candidates = await api.searchCaptures(
        queryFingerprint: queryFp,
        projectId: selectedProjectId == 'trash' ? '' : selectedProjectId,
        limit: 30,
      );

      final rescored = <Map<String, dynamic>>[];
      for (final item in candidates) {
        final capture = await VaultCrypto.decryptCaptureRecord(
          vaultKey!,
          Map<String, dynamic>.from(item['capture'] as Map),
        );
        final encVector = item['capture']['encrypted_vector'] as String?;
        List<double>? candidateEmb;
        if (encVector != null && encVector.isNotEmpty) {
          candidateEmb = await VaultCrypto.decryptEmbedding(vaultKey!, encVector);
        }
        if (candidateEmb == null || candidateEmb.isEmpty) {
          final noteText = buildCaptureEmbeddingText(
            capture['body'] as String? ?? '',
            (capture['tags'] as List?)?.cast<String>() ?? [],
          );
          candidateEmb = await embeddingService.embedText(noteText, isQuery: false);
        }
        double similarity = FingerprintService.cosineSimilarity(queryEmb, candidateEmb);
        if (similarity == 0.0) {
          similarity = FingerprintService.textSimilarity(
            query,
            FingerprintService.captureComparableText(capture),
          );
        }
        rescored.add({'capture': capture, 'similarity': similarity});
      }

      // Also merge with client-side FTS results
      final allDecrypted = await api.listCaptures(vaultKey: vaultKey);
      final ftsHits = _ftsSearch(query, allDecrypted);

      final merged = <String, Map<String, dynamic>>{};
      for (final hit in [...rescored, ...ftsHits]) {
        final id = hit['capture']['id'] as String;
        final existing = merged[id];
        if (existing == null || (hit['similarity'] as double) > (existing['similarity'] as double)) {
          merged[id] = hit;
        }
      }

      final sorted = merged.values.toList();
      sorted.sort((a, b) => (b['similarity'] as double).compareTo(a['similarity'] as double));

      searchResults = sorted
          .where((x) => (x['similarity'] as double) >= searchMinSimilarity)
          .toList();
      notifyListeners();
    } catch (e) {
      // Fallback
      final allRes = await api.listCaptures(vaultKey: vaultKey);
      searchResults = _ftsSearch(query, allRes);
      searchResults = searchResults
          .where((x) => (x['similarity'] as double) >= searchMinSimilarity)
          .toList();
      notifyListeners();
    }
  }

  List<Map<String, dynamic>> _ftsSearch(String query, List<Map<String, dynamic>> allDecrypted) {
    final terms = query.trim().toLowerCase().split(RegExp(r'\s+')).where((x) => x.isNotEmpty).toList();
    if (terms.isEmpty) return [];
    
    final results = <Map<String, dynamic>>[];
    for (final cap in allDecrypted) {
      final title = (cap['title'] as String? ?? '').toLowerCase();
      final body = (cap['body'] as String? ?? '').toLowerCase();
      final tags = (cap['tags'] as List?)?.join(' ').toLowerCase() ?? '';
      
      double score = 0.0;
      bool matchesAll = true;
      for (final term in terms) {
        if (title.contains(term)) {
          score += 3.0;
        } else if (tags.contains(term)) {
          score += 2.0;
        } else if (body.contains(term)) {
          score += 1.0;
        } else {
          matchesAll = false;
          break;
        }
      }
      if (matchesAll) {
        final sim = math.min(0.99, 0.72 + score * 0.04);
        results.add({'capture': cap, 'similarity': sim});
      }
    }
    results.sort((a, b) => (b['similarity'] as double).compareTo(a['similarity'] as double));
    return results;
  }

  @override
  void dispose() {
    embeddingService.dispose();
    iapService.dispose();
    super.dispose();
  }
}
