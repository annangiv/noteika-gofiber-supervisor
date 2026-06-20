import { embedQuery, embedPassage } from './embeddings';
import { decryptCaptureList, decryptCaptureRecord, decryptEmbedding } from './crypto';
import { buildCaptureEmbeddingText } from './captureContent';
import { fingerprintEmbeddingB64, cosineSimilarity } from './fingerprint';

export const DEFAULT_SEARCH_MIN = 0.70;
export const SEARCH_MIN_FLOOR = 0.50;
export const SEARCH_MIN_CEILING = 0.85;

export const SIMILARITY = {
  SEARCH_MIN: DEFAULT_SEARCH_MIN,
  BANNER: 0.65,
  DUPLICATE_WARN: 0.65,
  DUPLICATE_SAVE: 0.78,
};

function queryTerms(query) {
  return (query ?? '').trim().toLowerCase().split(/\s+/).filter(Boolean);
}

export function isNumericQuery(query) {
  return /^\d+$/.test((query ?? '').trim());
}

export function isSingleTermQuery(query) {
  return queryTerms(query).length === 1;
}

/**
 * Full-text score: every query term must appear (title > tags > body).
 * Returns 0 if any term is missing.
 */
export function captureTextMatchScore(query, capture) {
  const terms = queryTerms(query);
  if (!terms.length || !capture) return 0;

  const title = (capture.title ?? '').toLowerCase();
  const body = (capture.body ?? '').toLowerCase();
  const tags = (capture.tags ?? []).join(' ').toLowerCase();

  let score = 0;
  for (const term of terms) {
    if (title.includes(term)) score += 3;
    else if (tags.includes(term)) score += 2;
    else if (body.includes(term)) score += 1;
    else return 0;
  }
  return score;
}

async function fetchAllDecryptedCaptures(vaultKey) {
  if (!vaultKey) return [];
  const res = await fetch('/api/captures');
  if (!res.ok) return [];
  const raw = await res.json();
  return decryptCaptureList(vaultKey, Array.isArray(raw) ? raw : []);
}

/** Client-side full-text search on decrypted notes (E2E-safe). */
export function fullTextSearchCaptures(allCaptures, query, { limit = 20 } = {}) {
  const terms = queryTerms(query);
  if (!terms.length || !Array.isArray(allCaptures)) return [];

  return allCaptures
    .map((capture) => ({ capture, textMatch: captureTextMatchScore(query, capture) }))
    .filter((item) => item.textMatch > 0)
    .sort((a, b) => b.textMatch - a.textMatch)
    .slice(0, limit)
    .map(({ capture, textMatch }) => ({
      capture,
      similarity: Math.min(0.99, 0.72 + textMatch * 0.04),
      exactMatch: true,
    }));
}

/**
 * Merge server semantic hits with client full-text hits.
 * Exact matches first, then by text score, then embedding similarity.
 */
export function mergeHybridSearchResults(semanticResults, ftsResults, query, { limit = 20 } = {}) {
  const byId = new Map();

  for (const item of ftsResults) {
    byId.set(item.capture.id, { ...item, exactMatch: true });
  }

  for (const item of semanticResults) {
    const textMatch = captureTextMatchScore(query, item.capture);
    const existing = byId.get(item.capture.id);
    if (existing) {
      existing.similarity = Math.max(existing.similarity, item.similarity);
      if (textMatch > 0) existing.exactMatch = true;
    } else {
      byId.set(item.capture.id, {
        ...item,
        exactMatch: textMatch > 0,
      });
    }
  }

  return [...byId.values()]
    .sort((a, b) => {
      const aExact = a.exactMatch ? 1 : 0;
      const bExact = b.exactMatch ? 1 : 0;
      if (bExact !== aExact) return bExact - aExact;
      const aText = captureTextMatchScore(query, a.capture);
      const bText = captureTextMatchScore(query, b.capture);
      if (bText !== aText) return bText - aText;
      return b.similarity - a.similarity;
    })
    .slice(0, limit);
}

/**
 * Hybrid search: server semantic (vectors only) + client full-text on decrypted notes.
 */
export async function searchCapturesForUser(vaultKey, query, {
  minSimilarity = DEFAULT_SEARCH_MIN,
  limit = 20,
} = {}) {
  const trimmed = (query ?? '').trim();
  if (!trimmed || !vaultKey) return [];

  const [semantic, allCaptures] = await Promise.all([
    searchCaptures(vaultKey, trimmed, { minSimilarity, limit: Math.max(limit, 30) }),
    fetchAllDecryptedCaptures(vaultKey),
  ]);

  const fts = fullTextSearchCaptures(allCaptures, trimmed, { limit: Math.max(limit, 30) });

  return mergeHybridSearchResults(semantic, fts, trimmed, { limit });
}

/**
 * Server only ever sees a one-way-ish binarized fingerprint, so it can only
 * narrow candidates by approximate (Hamming-distance) similarity — it never
 * holds a real embedding. Alongside the fingerprint, each candidate also
 * carries its real embedding encrypted with the vault key (never a
 * server-held key, same guarantee as the note ciphertext); this decrypts
 * that instead of re-running the embedding model on every candidate, then
 * re-ranks by true cosine similarity for the exact final score.
 * minSimilarity is applied here, on the exact score — not server-side on
 * the approximate one, where LSH noise could wrongly drop a true
 * near-match before it's ever verified.
 */
export async function searchCaptures(vaultKey, query, { projectId = '', minSimilarity = DEFAULT_SEARCH_MIN, limit = 20, excludeId = '', queryEmbedding = null, asPassage = false } = {}) {
  let embedding = queryEmbedding;
  if (!embedding?.length && query?.trim()) {
    embedding = asPassage
      ? await embedPassage(query.trim())
      : await embedQuery(query.trim());
  }
  if (!embedding?.length || !vaultKey) return [];

  const res = await fetch('/api/captures/search', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      query_fingerprint: fingerprintEmbeddingB64(embedding),
      project_id: projectId,
      limit: Math.max(limit, 30),
      exclude_id: excludeId,
    }),
  });
  if (!res.ok) return [];
  const data = await res.json();
  const candidates = Array.isArray(data) ? data : [];

  const rescored = await Promise.all(candidates.map(async (item) => {
    const capture = await decryptCaptureRecord(vaultKey, item.capture);
    const candidateEmbedding = await decryptEmbedding(vaultKey, item.capture.encrypted_vector);
    return { capture, similarity: cosineSimilarity(embedding, candidateEmbedding) };
  }));

  return rescored
    .filter((item) => item.similarity >= minSimilarity)
    .sort((a, b) => b.similarity - a.similarity)
    .slice(0, limit);
}

/**
 * Duplicate check: embed draft as BGE query vs stored passage vectors (same text as save).
 * Asymmetric query↔passage avoids passage↔passage false positives between unrelated notes.
 */
export async function findDuplicateMatches(formBody, formTags, vaultKey, { limit = 5 } = {}) {
  const embedText = buildCaptureEmbeddingText(formBody, formTags);
  if (!embedText.trim() || !vaultKey) return [];

  const queryEmbedding = await embedQuery(embedText);
  if (!queryEmbedding.length) return [];

  return searchCaptures(vaultKey, '', {
    queryEmbedding,
    minSimilarity: SIMILARITY.DUPLICATE_WARN,
    limit,
  });
}

/** Plain-language hint for the search sensitivity slider (pct = 50–85). */
export function searchSensitivityHint(pct) {
  if (pct <= 55) return 'Broad — may show loosely related notes. Good if search feels too strict.';
  if (pct <= 69) return 'Balanced — wider net than default; some maybe-results may appear.';
  if (pct <= 74) return 'Focused — recommended default. Strong matches without much noise.';
  if (pct <= 79) return 'Strict — only closely related notes. May miss reworded captures.';
  return 'Near-duplicate — almost identical text only. Best for deduping, not discovery.';
}

export function formatRelativeTime(epoch) {
  const date = new Date(epoch * 1000);
  const now = new Date();
  const diffMs = now - date;
  const diffMins = Math.floor(diffMs / 60000);
  const diffHours = Math.floor(diffMs / 3600000);
  const diffDays = Math.floor(diffMs / 86400000);

  if (diffMins < 1) return 'just now';
  if (diffMins < 60) return `${diffMins}m ago`;
  if (diffHours < 24) return `${diffHours}h ago`;
  if (diffDays === 1) return 'yesterday';
  if (diffDays < 7) return `${diffDays}d ago`;
  return date.toLocaleDateString();
}

export { decryptCaptureList };
