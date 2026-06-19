import { embedQuery, embedPassage } from './embeddings';
import { decryptCaptureList, decryptCaptureRecord } from './crypto';
import { buildCaptureEmbeddingText } from './captureContent';

export const DEFAULT_SEARCH_MIN = 0.70;
export const SEARCH_MIN_FLOOR = 0.50;
export const SEARCH_MIN_CEILING = 0.85;

export const SIMILARITY = {
  SEARCH_MIN: DEFAULT_SEARCH_MIN,
  BANNER: 0.65,
  DUPLICATE_WARN: 0.65,
  DUPLICATE_SAVE: 0.78,
};

export async function decryptSearchResults(vaultKey, results) {
  if (!vaultKey || !Array.isArray(results)) return results ?? [];
  return Promise.all(
    results.map(async (item) => ({
      ...item,
      capture: await decryptCaptureRecord(vaultKey, item.capture),
    })),
  );
}

function queryTerms(query) {
  return (query ?? '').trim().toLowerCase().split(/\s+/).filter(Boolean);
}

/** How strongly decrypted capture text mentions query terms (title > tags > body). */
function captureTextMatchScore(query, capture) {
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
  }
  return score;
}

/**
 * After decrypt: prefer notes that literally mention the query.
 * Single-word searches drop embedding-only noise when a text match exists
 * (e.g. "noteika" should not surface grocery notes at 63%).
 */
export function rerankSearchResults(query, results) {
  if (!Array.isArray(results) || results.length === 0) return results ?? [];

  const terms = queryTerms(query);
  if (!terms.length) return results;

  const scored = results.map((item) => ({
    ...item,
    textMatch: captureTextMatchScore(query, item.capture),
  }));

  const hasTextMatch = scored.some((item) => item.textMatch > 0);
  let ranked = scored;

  if (terms.length === 1 && hasTextMatch) {
    ranked = scored.filter((item) => item.textMatch > 0);
  }

  ranked.sort((a, b) => {
    if (b.textMatch !== a.textMatch) return b.textMatch - a.textMatch;
    return b.similarity - a.similarity;
  });

  return ranked.map(({ textMatch: _textMatch, ...item }) => item);
}

export async function searchCaptures(query, { project = '', minSimilarity = DEFAULT_SEARCH_MIN, limit = 20, excludeId = '', queryEmbedding = null, asPassage = false } = {}) {
  let embedding = queryEmbedding;
  if (!embedding?.length && query?.trim()) {
    embedding = asPassage
      ? await embedPassage(query.trim())
      : await embedQuery(query.trim());
  }
  if (!embedding?.length) return [];

  const res = await fetch('/api/captures/search', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      query_embedding: embedding,
      project,
      min_similarity: minSimilarity,
      limit,
      exclude_id: excludeId,
    }),
  });
  if (!res.ok) return [];
  const data = await res.json();
  return Array.isArray(data) ? data : [];
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

  const matches = await searchCaptures('', {
    queryEmbedding,
    minSimilarity: SIMILARITY.DUPLICATE_WARN,
    limit,
  });
  return decryptSearchResults(vaultKey, matches);
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
