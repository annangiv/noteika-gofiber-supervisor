import { bytesToB64 } from './crypto';

const ROWS = 256; // fingerprint bits
const COLS = 384; // BGE-small embedding dimension
const HKDF_HASH = 'SHA-256';
// One HKDF-Expand call maxes out at 255*hashLen bytes (8160 for SHA-256);
// ROWS*COLS bits = 12288 bytes needed, so split across two labeled calls.
const CHUNK_BITS = (ROWS * COLS) / 2;
const MATRIX_INFO_LABELS = ['noteika-fingerprint-matrix-0', 'noteika-fingerprint-matrix-1'];

let matrix = null; // Int8Array(ROWS*COLS) of +/-1, row-major

async function hkdfExpand(hkdfKey, infoLabel, bits) {
  const bytes = await crypto.subtle.deriveBits(
    {
      name: 'HKDF',
      hash: HKDF_HASH,
      salt: new Uint8Array(0),
      info: new TextEncoder().encode(infoLabel),
    },
    hkdfKey,
    bits,
  );
  return new Uint8Array(bytes);
}

/** Derive this user's projection matrix from their vault HKDF key and cache it. Call once at unlock. */
export async function deriveAndSetMatrix(hkdfKey) {
  const chunks = await Promise.all(
    MATRIX_INFO_LABELS.map((label) => hkdfExpand(hkdfKey, label, CHUNK_BITS)),
  );
  const randomBytes = new Uint8Array(chunks[0].length + chunks[1].length);
  randomBytes.set(chunks[0], 0);
  randomBytes.set(chunks[1], chunks[0].length);

  const m = new Int8Array(ROWS * COLS);
  for (let i = 0; i < m.length; i += 1) {
    const byte = randomBytes[i >> 3];
    const bit = (byte >> (i & 7)) & 1;
    m[i] = bit ? 1 : -1;
  }
  matrix = m;
}

/** Drop the cached matrix (call on vault lock — it's derived from secret material). */
export function clearMatrix() {
  matrix = null;
}

export function isMatrixReady() {
  return matrix !== null;
}

/** Project + binarize + pack a real embedding into a 32-byte fingerprint. */
export function fingerprintEmbedding(embedding) {
  if (!matrix) throw new Error('Vault not unlocked — no fingerprint matrix');
  if (embedding.length !== COLS) {
    throw new Error(`Expected a ${COLS}-dim embedding, got ${embedding.length}`);
  }

  const packed = new Uint8Array(ROWS / 8);
  for (let row = 0; row < ROWS; row += 1) {
    const base = row * COLS;
    let dot = 0;
    for (let col = 0; col < COLS; col += 1) {
      dot += matrix[base + col] * embedding[col];
    }
    if (dot >= 0) packed[row >> 3] |= (1 << (row & 7));
  }
  return packed;
}

/** Convenience: fingerprint an embedding and base64-encode it for transport. */
export function fingerprintEmbeddingB64(embedding) {
  return bytesToB64(fingerprintEmbedding(embedding));
}

/** True cosine similarity between two real (non-fingerprinted) embeddings. */
export function cosineSimilarity(a, b) {
  let dot = 0;
  let normA = 0;
  let normB = 0;
  for (let i = 0; i < a.length; i += 1) {
    dot += a[i] * b[i];
    normA += a[i] * a[i];
    normB += b[i] * b[i];
  }
  if (normA === 0 || normB === 0) return 0;
  return dot / (Math.sqrt(normA) * Math.sqrt(normB));
}
