import 'dart:convert';
import 'dart:typed_data';

import 'package:cryptography/cryptography.dart';

const _pbkdf2Iterations = 310000;

/// Client-side vault crypto — mirrors frontend/src/lib/crypto.js
class VaultCrypto {
  static final _pbkdf2 = Pbkdf2(
    macAlgorithm: Hmac.sha256(),
    iterations: _pbkdf2Iterations,
    bits: 256,
  );
  static final _aesGcm = AesGcm.with256bits();

  static Uint8List saltFromBase64(String b64) => base64Decode(b64);

  static Future<SecretKey> deriveVaultKey(String passcode, Uint8List salt) {
    return _pbkdf2.deriveKeyFromPassword(
      password: passcode,
      nonce: salt,
    );
  }

  static Future<String> encryptCapturePayload(
    SecretKey key,
    Map<String, dynamic> payload,
  ) async {
    final iv = _aesGcm.newNonce();
    final secretBox = await _aesGcm.encrypt(
      utf8.encode(jsonEncode(payload)),
      secretKey: key,
      nonce: iv,
    );
    final combined = Uint8List(iv.length + secretBox.cipherText.length + secretBox.mac.bytes.length);
    combined.setRange(0, iv.length, iv);
    combined.setRange(iv.length, iv.length + secretBox.cipherText.length, secretBox.cipherText);
    combined.setRange(
      iv.length + secretBox.cipherText.length,
      combined.length,
      secretBox.mac.bytes,
    );
    return base64Encode(combined);
  }

  static Future<Map<String, dynamic>?> decryptCapturePayload(
    SecretKey key,
    String ciphertextB64,
  ) async {
    if (ciphertextB64.isEmpty) return null;
    final combined = base64Decode(ciphertextB64);
    if (combined.length < 12 + 16) return null;

    const ivLen = 12;
    const macLen = 16;
    final iv = combined.sublist(0, ivLen);
    final cipherText = combined.sublist(ivLen, combined.length - macLen);
    final mac = Mac(combined.sublist(combined.length - macLen));

    final plain = await _aesGcm.decrypt(
      SecretBox(cipherText, nonce: iv, mac: mac),
      secretKey: key,
    );
    return jsonDecode(utf8.decode(plain)) as Map<String, dynamic>;
  }

  static Future<Map<String, dynamic>> decryptCaptureRecord(
    SecretKey key,
    Map<String, dynamic> record,
  ) async {
    final ct = record['ciphertext'] as String?;
    if (ct == null || ct.isEmpty) return record;

    final plain = await decryptCapturePayload(key, ct);
    if (plain == null) return record;

    return {
      ...record,
      'title': plain['title'] ?? '',
      'body': plain['body'] ?? '',
      'source_url': plain['source_url'] ?? '',
      'tags': (plain['tags'] as List?)?.cast<String>() ?? <String>[],
      '_encrypted': true,
    };
  }

  static Future<List<Map<String, dynamic>>> decryptCaptureList(
    SecretKey key,
    List<dynamic> records,
  ) async {
    final out = <Map<String, dynamic>>[];
    for (final r in records) {
      out.add(await decryptCaptureRecord(key, Map<String, dynamic>.from(r as Map)));
    }
    return out;
  }
}
