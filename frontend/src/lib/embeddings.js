import { pipeline, env } from '@huggingface/transformers';

env.allowLocalModels = false;
env.useBrowserCache = true;

const MODEL_ID = 'Xenova/bge-small-en-v1.5';

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

export async function embedText(text) {
  const trimmed = (text ?? '').trim();
  if (!trimmed) return [];
  const extractor = await getEmbedder();
  const output = await extractor(trimmed, { pooling: 'mean', normalize: true });
  return Array.from(output.data);
}

export function buildEmbeddingText(title, body, tags = []) {
  const parts = [];
  if (tags.length > 0) parts.push(tags.join(' '));
  if (title?.trim()) parts.push(title.trim());
  if (body?.trim()) parts.push(body.trim());
  return parts.join('\n');
}
