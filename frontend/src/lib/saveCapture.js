import { encryptCapturePayload, encryptEmbedding } from './crypto';
import { embedPassage } from './embeddings';
import { resolveProjectId } from './projects';
import { fingerprintEmbeddingB64 } from './fingerprint';
import {
  generateAutoTitle,
  classifyContentType,
  parseTagsInput,
  formatTagsInput,
  mergeTags,
  parseHashtags,
  buildCaptureEmbeddingText,
} from './captureContent';

/**
 * Same E2E save path as NotesPage: client embed + encrypt → POST /api/captures.
 */
export async function saveCapture(vaultKey, {
  body,
  project = 'Inbox',
  tags = '',
  sourceUrl = '',
  type = null,
} = {}) {
  if (!vaultKey) throw new Error('Vault not unlocked');
  const trimmed = (body ?? '').trim();
  if (!trimmed) throw new Error('Empty note body');

  const mergedTags = mergeTags(parseTagsInput(tags), parseHashtags(trimmed));
  const cType = type ?? classifyContentType(trimmed);
  const title = generateAutoTitle(trimmed);
  const embedding = await embedPassage(
    buildCaptureEmbeddingText(trimmed, formatTagsInput(mergedTags)),
  );
  const ciphertext = await encryptCapturePayload(vaultKey, {
    title,
    body: trimmed,
    source_url: sourceUrl,
    tags: mergedTags,
  });
  const projectId = await resolveProjectId(vaultKey, project);
  const fingerprint = fingerprintEmbeddingB64(embedding);
  const encryptedVector = await encryptEmbedding(vaultKey, embedding);

  const res = await fetch('/api/captures', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      ciphertext,
      fingerprint,
      encrypted_vector: encryptedVector,
      project_id: projectId,
      type: cType,
    }),
  });

  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    const error = new Error(err.error || `Save failed (${res.status})`);
    error.status = res.status;
    error.payload = err;
    throw error;
  }
  return res.json();
}
