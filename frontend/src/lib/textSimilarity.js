/**
 * Text similarity for duplicate detection — mirrors utils/text_similarity.go.
 * Catches copy-paste duplicates when fingerprints are missing (pre-v2 notes).
 */

function normalizeText(s) {
  const trimmed = (s ?? '').trim().toLowerCase();
  const parts = trimmed.split(/\s+/).filter(Boolean);
  return parts.join(' ');
}

/** @returns {number} 0–1 score */
export function textSimilarity(a, b) {
  const na = normalizeText(a);
  const nb = normalizeText(b);
  if (!na || !nb) return 0;
  if (na === nb) return 1;

  if (na.includes(nb) || nb.includes(na)) {
    const shorter = na.length <= nb.length ? na.length : nb.length;
    const longer = na.length > nb.length ? na.length : nb.length;
    const ratio = shorter / longer;
    if (ratio > 0.85) return ratio;
  }

  const wordsA = na.split(' ');
  const wordsB = nb.split(' ');
  if (!wordsA.length || !wordsB.length) return 0;

  const setA = new Set(wordsA);
  const setB = new Set(wordsB);
  let intersection = 0;
  for (const w of setB) {
    if (setA.has(w)) intersection += 1;
  }
  const union = setA.size + setB.size - intersection;
  if (union === 0) return 0;
  return intersection / union;
}

/** Build comparable text blob from a decrypted capture (title + tags + body). */
export function captureComparableText(capture) {
  const parts = [];
  const tags = (capture?.tags ?? []).join(' ');
  if (tags) parts.push(tags);
  const title = (capture?.title ?? '').trim();
  if (title) parts.push(title);
  const body = (capture?.body ?? '').trim();
  if (body) parts.push(body);
  return parts.join('\n');
}
