export function createSaveActivity(project, { title = 'Capture saved' } = {}) {
  return {
    id: `${Date.now()}-save`,
    type: 'save',
    time: new Date(),
    title,
    subtitle: project || 'Inbox',
    steps: [
      {
        where: 'device',
        label: 'Encrypted on your device',
        detail: 'Title, body, tags, and URL wrapped with AES-GCM using your vault passcode.',
      },
      {
        where: 'device',
        label: 'Embedded locally',
        detail: 'Semantic fingerprint computed in-browser (BGE model) for search & duplicates.',
      },
      {
        where: 'server',
        label: 'Sent over TLS',
        detail: 'Ciphertext + embedding vector only — no readable note text.',
      },
      {
        where: 'server',
        label: 'Stored on server',
        detail: 'Encrypted blob + encrypted vector at rest. Project name & timestamps are plain metadata.',
      },
    ],
  };
}

export function createSearchActivity(query, resultCount, exactCount = 0) {
  const q = query.length > 48 ? `${query.slice(0, 48)}…` : query;
  return {
    id: `${Date.now()}-search`,
    type: 'search',
    time: new Date(),
    title: 'Search completed',
    subtitle: `"${q}"`,
    steps: [
      {
        where: 'device',
        label: 'Query embedded locally',
        detail: 'Your search words become a vector on this device — query text is not stored on the server.',
      },
      {
        where: 'server',
        label: 'Semantic ranking',
        detail: 'Server compares query vector to stored note vectors (cosine similarity). Vectors decrypted in RAM only.',
      },
      {
        where: 'device',
        label: 'Results decrypted here',
        detail: `${resultCount} match${resultCount !== 1 ? 'es' : ''} unlocked with your vault passcode on this device.`,
      },
      ...(exactCount > 0
        ? [{
            where: 'device',
            label: 'Exact-word merge',
            detail: `${exactCount} result${exactCount !== 1 ? 's' : ''} boosted by full-text match on decrypted notes.`,
          }]
        : []),
    ],
  };
}

export function createDuplicateCheckActivity(matchCount) {
  if (matchCount === 0) return null;
  return {
    id: `${Date.now()}-dup`,
    type: 'duplicate',
    time: new Date(),
    title: 'Duplicate check',
    subtitle: `${matchCount} similar capture${matchCount !== 1 ? 's' : ''}`,
    steps: [
      {
        where: 'device',
        label: 'Draft embedded as query',
        detail: 'Your typing is compared against stored vectors — still no plaintext sent.',
      },
      {
        where: 'server',
        label: 'Similarity ranked',
        detail: 'Server returns closest matches by vector distance.',
      },
      {
        where: 'device',
        label: 'Matches decrypted locally',
        detail: 'You see titles and snippets only after your device decrypts them.',
      },
    ],
  };
}
