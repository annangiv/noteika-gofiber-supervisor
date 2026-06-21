import 'dart:convert';
import 'dart:math' as math;
import 'dart:typed_data';
import 'package:cryptography/cryptography.dart';

class FingerprintService {
  static const int rows = 256;
  static const int cols = 384;
  static const int chunkBytes = (rows * cols) ~/ 2 ~/ 8; // 6144 bytes

  static const List<String> matrixInfoLabels = [
    'noteika-fingerprint-matrix-0',
    'noteika-fingerprint-matrix-1'
  ];

  Int8List? _matrix;

  bool get isReady => _matrix != null;

  void clear() {
    _matrix = null;
  }

  Future<void> deriveAndSetMatrix(SecretKey rootKey) async {
    final hkdf = Hkdf(
      hmac: Hmac.sha256(),
      outputLength: chunkBytes,
    );

    final chunks = await Future.wait(
      matrixInfoLabels.map((label) async {
        final derivedKey = await hkdf.deriveKey(
          secretKey: rootKey,
          info: utf8.encode(label),
          nonce: Uint8List(0),
        );
        final bytes = await derivedKey.extractBytes();
        return Uint8List.fromList(bytes);
      }),
    );

    final randomBytes = Uint8List(chunks[0].length + chunks[1].length);
    randomBytes.setRange(0, chunks[0].length, chunks[0]);
    randomBytes.setRange(chunks[0].length, randomBytes.length, chunks[1]);

    final m = Int8List(rows * cols);
    for (var i = 0; i < m.length; i++) {
      final byte = randomBytes[i >> 3];
      final bit = (byte >> (i & 7)) & 1;
      m[i] = bit == 1 ? 1 : -1;
    }
    _matrix = m;
  }

  Uint8List fingerprintEmbedding(List<double> embedding) {
    final m = _matrix;
    if (m == null) {
      throw StateError('Vault not unlocked — no fingerprint matrix');
    }
    if (embedding.length != cols) {
      throw ArgumentError('Expected a $cols-dim embedding, got ${embedding.length}');
    }

    final packed = Uint8List(rows ~/ 8); // 32 bytes
    for (var row = 0; row < rows; row++) {
      final base = row * cols;
      double dot = 0.0;
      for (var col = 0; col < cols; col++) {
        dot += m[base + col] * embedding[col];
      }
      if (dot >= 0.0) {
        packed[row >> 3] |= (1 << (row & 7));
      }
    }
    return packed;
  }

  String fingerprintEmbeddingB64(List<double> embedding) {
    return base64Encode(fingerprintEmbedding(embedding));
  }

  static double cosineSimilarity(List<double> a, List<double> b) {
    if (a.isEmpty || b.isEmpty || a.length != b.length) return 0.0;
    double dot = 0.0;
    double normA = 0.0;
    double normB = 0.0;
    for (var i = 0; i < a.length; i++) {
      dot += a[i] * b[i];
      normA += a[i] * a[i];
      normB += b[i] * b[i];
    }
    if (normA == 0.0 || normB == 0.0) return 0.0;
    return dot / (math.sqrt(normA) * math.sqrt(normB));
  }

  static double textSimilarity(String a, String b) {
    final na = _normalizeText(a);
    final nb = _normalizeText(b);
    if (na.isEmpty || nb.isEmpty) return 0.0;
    if (na == nb) return 1.0;

    if (na.contains(nb) || nb.contains(na)) {
      final shorter = na.length <= nb.length ? na.length : nb.length;
      final longer = na.length > nb.length ? na.length : nb.length;
      final ratio = shorter / longer;
      if (ratio > 0.85) return ratio;
    }

    final wordsA = na.split(' ');
    final wordsB = nb.split(' ');
    if (wordsA.isEmpty || wordsB.isEmpty) return 0.0;

    final setA = wordsA.toSet();
    final setB = wordsB.toSet();
    int intersection = 0;
    for (final w in setB) {
      if (setA.contains(w)) intersection += 1;
    }
    final union = setA.length + setB.length - intersection;
    if (union == 0) return 0.0;
    return intersection / union;
  }

  static String _normalizeText(String s) {
    final trimmed = s.trim().toLowerCase();
    final parts = trimmed.split(RegExp(r'\s+')).where((x) => x.isNotEmpty);
    return parts.join(' ');
  }

  static String captureComparableText(Map<String, dynamic> capture) {
    final parts = <String>[];
    final tagsList = capture['tags'] as List?;
    final tags = tagsList != null ? tagsList.join(' ') : '';
    if (tags.isNotEmpty) parts.add(tags);
    final title = (capture['title'] as String? ?? '').trim();
    if (title.isNotEmpty) parts.add(title);
    final body = (capture['body'] as String? ?? '').trim();
    if (body.isNotEmpty) parts.add(body);
    return parts.join('\n');
  }
}
