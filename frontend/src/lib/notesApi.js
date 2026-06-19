export const SIMILARITY = {
  SEARCH_MIN: 0.35,
  BANNER: 0.65,
  DUPLICATE_WARN: 0.55,
  DUPLICATE_SAVE: 0.70,
  RESURFACE: 0.45,
};

export const LAST_SEARCH_KEY = 'noteika_last_search';

export async function searchCaptures(query, { project = '', minSimilarity = SIMILARITY.SEARCH_MIN, limit = 20, excludeId = '' } = {}) {
  const res = await fetch('/api/captures/search', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      query,
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
