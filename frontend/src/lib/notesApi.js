import { embedText } from './embeddings';
import { decryptCaptureList, decryptCaptureRecord } from './crypto';

export const DEFAULT_SEARCH_MIN = 0.70;
export const SEARCH_MIN_FLOOR = 0.50;
export const SEARCH_MIN_CEILING = 0.85;

export const SIMILARITY = {
  SEARCH_MIN: DEFAULT_SEARCH_MIN,
  BANNER: 0.65,
  DUPLICATE_WARN: 0.55,
  DUPLICATE_SAVE: 0.70,
  RESURFACE: 0.45,
};

export const LAST_SEARCH_KEY = 'noteika_last_search';

export async function decryptSearchResults(vaultKey, results) {
  if (!vaultKey || !Array.isArray(results)) return results ?? [];
  return Promise.all(
    results.map(async (item) => ({
      ...item,
      capture: await decryptCaptureRecord(vaultKey, item.capture),
    })),
  );
}

export async function searchCaptures(query, { project = '', minSimilarity = DEFAULT_SEARCH_MIN, limit = 20, excludeId = '', queryEmbedding = null } = {}) {
  let embedding = queryEmbedding;
  if (!embedding?.length && query?.trim()) {
    embedding = await embedText(query.trim());
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
