import 'dart:typed_data';

import 'package:cookie_jar/cookie_jar.dart';
import 'package:cryptography/cryptography.dart';
import 'package:dio/dio.dart';
import 'package:dio_cookie_manager/dio_cookie_manager.dart';
import 'package:path_provider/path_provider.dart';
import 'package:uuid/uuid.dart';

import '../config/api_config.dart';
import '../utils/capture_content.dart';
import 'vault_crypto.dart';

class ProjectRef {
  const ProjectRef({required this.id, required this.name});
  final String id;
  final String name;
}

class NoteikaApi {
  NoteikaApi._(this._dio, this._cookieJar);

  final Dio _dio;
  final CookieJar _cookieJar;
  static const inbox = ProjectRef(id: 'inbox', name: 'Inbox');

  static Future<NoteikaApi> create() async {
    final dir = await getApplicationDocumentsDirectory();
    final jar = PersistCookieJar(
      storage: FileStorage('${dir.path}/.cookies/'),
    );
    final dio = Dio(BaseOptions(
      baseUrl: ApiConfig.baseUrl,
      connectTimeout: const Duration(seconds: 15),
      receiveTimeout: const Duration(seconds: 30),
      headers: {'Accept': 'application/json'},
    ));
    dio.interceptors.add(CookieManager(jar));
    return NoteikaApi._(dio, jar);
  }

  String get baseUrl => ApiConfig.baseUrl;

  Future<Map<String, dynamic>?> currentUser() async {
    try {
      final res = await _dio.get('/api/auth/me');
      if (res.statusCode == 200) {
        return Map<String, dynamic>.from(res.data as Map);
      }
    } catch (_) {}
    return null;
  }

  Future<void> logout() async {
    await _dio.get('/auth/logout');
    await _cookieJar.deleteAll();
  }

  /// Development-only sign-in (ENVIRONMENT=development). Google OAuth is blocked in WebViews.
  Future<Map<String, dynamic>> devLogin({String? email, String? name}) async {
    final res = await _dio.post('/api/auth/dev-login', data: {
      if (email != null && email.isNotEmpty) 'email': email,
      if (name != null && name.isNotEmpty) 'name': name,
    });
    if (res.statusCode != 200) {
      throw Exception('Dev login failed (${res.statusCode})');
    }
    return Map<String, dynamic>.from(res.data as Map);
  }

  Future<Uint8List> fetchVaultSalt() async {
    final res = await _dio.get('/api/vault/salt');
    final saltB64 = (res.data as Map)['salt'] as String;
    return VaultCrypto.saltFromBase64(saltB64);
  }

  Future<List<ProjectRef>> listProjects(SecretKey vaultKey) async {
    final res = await _dio.get('/api/projects');
    final raw = (res.data as List?) ?? [];
    final projects = <ProjectRef>[inbox];
    for (final item in raw) {
      final map = Map<String, dynamic>.from(item as Map);
      final plain = await VaultCrypto.decryptCapturePayload(
        vaultKey,
        map['ciphertext'] as String? ?? '',
      );
      final name = plain?['name'] as String? ?? 'Project';
      projects.add(ProjectRef(id: map['id'] as String, name: name));
    }
    projects.sort((a, b) {
      if (a.id == 'inbox') return -1;
      if (b.id == 'inbox') return 1;
      return a.name.compareTo(b.name);
    });
    return projects;
  }

  Future<String> resolveProjectId(SecretKey vaultKey, String name) async {
    final trimmed = name.trim();
    if (trimmed.isEmpty || trimmed.toLowerCase() == 'inbox') return 'inbox';

    final projects = await listProjects(vaultKey);
    for (final p in projects) {
      if (p.name.toLowerCase() == trimmed.toLowerCase()) return p.id;
    }

    final id = const Uuid().v4();
    final ciphertext = await VaultCrypto.encryptCapturePayload(vaultKey, {'name': trimmed});
    await _dio.post('/api/projects', data: {'id': id, 'ciphertext': ciphertext});
    return id;
  }

  Future<List<Map<String, dynamic>>> listCaptures({
    String? projectId,
    SecretKey? vaultKey,
  }) async {
    final res = await _dio.get(
      '/api/captures',
      queryParameters: projectId != null && projectId.isNotEmpty
          ? {'project_id': projectId}
          : null,
    );
    final raw = (res.data as List).cast<dynamic>();
    if (vaultKey == null) {
      return raw.map((e) => Map<String, dynamic>.from(e as Map)).toList();
    }
    return VaultCrypto.decryptCaptureList(vaultKey, raw);
  }

  /// Save encrypted capture. Fingerprint + encrypted_vector require on-device
  /// BGE embeddings (not yet in Flutter) — omitted for v0.1; search is web-only.
  Future<Map<String, dynamic>> createCapture({
    required SecretKey vaultKey,
    required String body,
    String projectName = 'Inbox',
    List<String> tags = const [],
    String sourceUrl = '',
  }) async {
    final trimmed = body.trim();
    if (trimmed.isEmpty) throw Exception('Empty note body');

    final mergedTags = mergeTags([tags, parseHashtags(trimmed)]);
    final title = generateAutoTitle(trimmed);
    final cType = classifyContentType(trimmed);
    final projectId = await resolveProjectId(vaultKey, projectName);

    final ciphertext = await VaultCrypto.encryptCapturePayload(vaultKey, {
      'title': title,
      'body': trimmed,
      'source_url': sourceUrl,
      'tags': mergedTags,
    });

    final res = await _dio.post('/api/captures', data: {
      'ciphertext': ciphertext,
      'project_id': projectId,
      'type': cType,
    });

    if (res.statusCode != 201) {
      final err = res.data is Map ? (res.data as Map)['error'] : null;
      throw Exception(err?.toString() ?? 'Save failed (${res.statusCode})');
    }
    return Map<String, dynamic>.from(res.data as Map);
  }

  Future<void> deleteCapture(String id) async {
    await _dio.delete('/api/captures/$id');
  }
}
