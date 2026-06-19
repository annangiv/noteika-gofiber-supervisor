import { pipeline, env } from '@huggingface/transformers';

env.allowLocalModels = false;
env.useBrowserCache = true;

const MODEL_ID = 'Xenova/bge-small-en-v1.5';

// BGE-small-en-v1.5 expects asymmetric query/passage prefixes for retrieval.
const QUERY_PREFIX = 'query: ';
const PASSAGE_PREFIX = 'passage: ';

let embedderPromise = null;
let loadError = null;

function getEmbedder() {
  if (loadError) return Promise.reject(loadError);
  if (!embedderPromise) {
    embedderPromise = pipeline('feature-extraction', MODEL_ID, {
      dtype: 'fp32',
    }).catch((err) => {
      loadError = err;
      embedderPromise = null;
      throw err;
    });
  }
  return embedderPromise;
}

export function isEmbedderReady() {
  return embedderPromise !== null && loadError === null;
}

export async function preloadEmbedder() {
  await getEmbedder();
}

async function embedPrefixed(text) {
  const trimmed = (text ?? '').trim();
  if (!trimmed) return [];
  const extractor = await getEmbedder();
  const output = await extractor(trimmed, { pooling: 'mean', normalize: true });
  return Array.from(output.data);
}

/** Embed a search query (short text like "breakfast shopping"). */
export async function embedQuery(text) {
  const trimmed = (text ?? '').trim();
  if (!trimmed) return [];
  return embedPrefixed(`${QUERY_PREFIX}${trimmed}`);
}

/** Embed note content stored for similarity against other notes. */
export async function embedPassage(text) {
  const trimmed = (text ?? '').trim();
  if (!trimmed) return [];
  return embedPrefixed(`${PASSAGE_PREFIX}${trimmed}`);
}

/** @deprecated use embedQuery or embedPassage */
export async function embedText(text) {
  return embedQuery(text);
}

export function buildEmbeddingText(title, body, tags = []) {
  const parts = [];
  if (tags.length > 0) parts.push(tags.join(' '));
  if (title?.trim()) parts.push(title.trim());
  if (body?.trim()) parts.push(body.trim());
  return parts.join('\n');
}
