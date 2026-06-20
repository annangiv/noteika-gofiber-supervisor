const PBKDF2_ITERATIONS = 310_000;

export function b64ToBytes(b64) {
  const bin = atob(b64);
  const out = new Uint8Array(bin.length);
  for (let i = 0; i < bin.length; i += 1) out[i] = bin.charCodeAt(i);
  return out;
}

export function bytesToB64(bytes) {
  let bin = '';
  bytes.forEach((b) => { bin += String.fromCharCode(b); });
  return btoa(bin);
}

export async function fetchVaultSalt() {
  const res = await fetch('/api/vault/salt');
  if (!res.ok) throw new Error('Failed to load vault salt');
  const data = await res.json();
  return b64ToBytes(data.salt);
}

/**
 * Derives a single root key from the passcode (slow, via PBKDF2), then fans
 * it out into the AES-GCM vault key (note encryption) and an HKDF base key
 * (embedding-fingerprint matrix derivation — see lib/fingerprint.js). Doing
 * one slow derivation and fanning out avoids running PBKDF2 twice.
 */
export async function deriveVaultKeys(passcode, saltBytes) {
  const enc = new TextEncoder();
  const keyMaterial = await crypto.subtle.importKey(
    'raw',
    enc.encode(passcode),
    'PBKDF2',
    false,
    ['deriveBits'],
  );
  const rootBits = await crypto.subtle.deriveBits(
    { name: 'PBKDF2', salt: saltBytes, iterations: PBKDF2_ITERATIONS, hash: 'SHA-256' },
    keyMaterial,
    256,
  );
  const vaultKey = await crypto.subtle.importKey(
    'raw', rootBits, { name: 'AES-GCM' }, false, ['encrypt', 'decrypt'],
  );
  const hkdfKey = await crypto.subtle.importKey(
    'raw', rootBits, 'HKDF', false, ['deriveBits'],
  );
  return { vaultKey, hkdfKey };
}

export async function encryptCapturePayload(key, payload) {
  const iv = crypto.getRandomValues(new Uint8Array(12));
  const encoded = new TextEncoder().encode(JSON.stringify(payload));
  const ciphertext = await crypto.subtle.encrypt({ name: 'AES-GCM', iv }, key, encoded);
  const combined = new Uint8Array(iv.length + ciphertext.byteLength);
  combined.set(iv, 0);
  combined.set(new Uint8Array(ciphertext), iv.length);
  return bytesToB64(combined);
}

export async function decryptCapturePayload(key, ciphertextB64) {
  if (!ciphertextB64) return null;
  const combined = b64ToBytes(ciphertextB64);
  const iv = combined.slice(0, 12);
  const data = combined.slice(12);
  const plain = await crypto.subtle.decrypt({ name: 'AES-GCM', iv }, key, data);
  return JSON.parse(new TextDecoder().decode(plain));
}

export async function decryptCaptureRecord(key, record) {
  if (!record?.ciphertext) return record;
  const plain = await decryptCapturePayload(key, record.ciphertext);
  return {
    ...record,
    title: plain.title ?? '',
    body: plain.body ?? '',
    source_url: plain.source_url ?? '',
    tags: plain.tags ?? [],
    _encrypted: true,
  };
}

export async function decryptCaptureList(key, records) {
  if (!key || !Array.isArray(records)) return records ?? [];
  return Promise.all(records.map((r) => decryptCaptureRecord(key, r)));
}

export function collectTagsFromCaptures(captures) {
  const seen = new Set();
  const tags = [];
  for (const cap of captures) {
    for (const tag of cap.tags ?? []) {
      const t = tag.trim().toLowerCase();
      if (t && !seen.has(t)) {
        seen.add(t);
        tags.push(t);
      }
    }
  }
  return tags.sort();
}
