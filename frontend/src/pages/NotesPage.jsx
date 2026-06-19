import React, { useState, useEffect, useRef, useCallback } from 'react';
import { Link, Navigate } from 'react-router-dom';
import {
  SIMILARITY, decryptCaptureList, findDuplicateMatches, formatRelativeTime, DEFAULT_SEARCH_MIN,
  searchCapturesForUser,
} from '../lib/notesApi';
import { useVault } from '../context/VaultContext';
import VaultGate from '../components/VaultGate';
import { saveCapture } from '../lib/saveCapture';
import {
  generateAutoTitle, classifyContentType, parseTagsInput, formatTagsInput,
  mergeTags, parseHashtags, buildCaptureEmbeddingText,
} from '../lib/captureContent';
import { embedPassage } from '../lib/embeddings';
import { encryptCapturePayload } from '../lib/crypto';
import PrivacyTrustCard, { PrivacyActivityPanel } from '../components/PrivacyPanel';
import {
  createSaveActivity, createSearchActivity, createDuplicateCheckActivity,
} from '../lib/privacyActivity';
import '../notes.css';

export default function NotesPage() {
  const { vaultKey } = useVault();
  // Authentication states
  const [user, setUser] = useState(null);
  const [authLoading, setAuthLoading] = useState(true);

  // App data states
  const [captures, setCaptures] = useState([]);
  const [projects, setProjects] = useState([]);
  const [selectedProject, setSelectedProject] = useState('Inbox');
  const [searchQuery, setSearchQuery] = useState('');
  const [searchResults, setSearchResults] = useState([]);
  const [isSearching, setIsSearching] = useState(false);

  // Form states
  const [formBody, setFormBody] = useState('');
  const [formTags, setFormTags] = useState('');
  const [formProject, setFormProject] = useState('');
  const [formSourceUrl, setFormSourceUrl] = useState('');
  const [knownTags, setKnownTags] = useState([]);
  const [isSaving, setIsSaving] = useState(false);

  // Proactive duplicate resurfacing states
  const [duplicateWarning, setDuplicateWarning] = useState([]);
  const [saveDuplicateConfirm, setSaveDuplicateConfirm] = useState(null);
  const duplicateDebounceTimer = useRef(null);
  const searchDebounceTimer = useRef(null);
  const lastDupPrivacyRef = useRef('');

  // Privacy activity trace (save / search transparency)
  const [privacyEvents, setPrivacyEvents] = useState([]);

  const pushPrivacyEvent = useCallback((event) => {
    if (!event) return;
    setPrivacyEvents((prev) => [event, ...prev].slice(0, 8));
  }, []);

  // Modals state
  const [editingCapture, setEditingCapture] = useState(null);
  const [deletingCapture, setDeletingCapture] = useState(null);
  const [emptyTrashConfirm, setEmptyTrashConfirm] = useState(false);

  // Toasts state
  const [toast, setToast] = useState(null);
  const toastTimer = useRef(null);

  const showToast = (message, type = 'success') => {
    if (toastTimer.current) clearTimeout(toastTimer.current);
    setToast({ message, type });
    toastTimer.current = setTimeout(() => {
      setToast(null);
    }, 4000);
  };

  // 1. Authenticate on mount
  useEffect(() => {
    async function checkAuth() {
      try {
        const res = await fetch('/api/auth/me');
        if (res.ok) {
          const data = await res.json();
          setUser(data);
        } else {
          setUser(null);
        }
      } catch (err) {
        setUser(null);
        console.error('Auth verification failed:', err);
      } finally {
        setAuthLoading(false);
      }
    }
    checkAuth();
  }, []);

  // 2. Fetch Projects and Captures when authenticated
  const loadProjects = async () => {
    try {
      const res = await fetch('/api/projects');
      if (res.ok) {
        const data = await res.json();
        setProjects(Array.isArray(data) ? data : []);
      }
    } catch (err) {
      console.error('Failed to load projects:', err);
    }
  };

  const loadCaptures = async () => {
    if (!vaultKey) return;
    try {
      let url = '/api/captures';
      if (selectedProject) {
        url += `?project=${encodeURIComponent(selectedProject)}`;
      }
      const res = await fetch(url);
      if (res.ok) {
        const data = await res.json();
        const decrypted = await decryptCaptureList(vaultKey, Array.isArray(data) ? data : []);
        setCaptures(decrypted);
        setKnownTags(
          [...new Set(decrypted.flatMap((c) => (c.tags ?? []).map((t) => t.toLowerCase())))].sort(),
        );
      }
    } catch (err) {
      showToast('Failed to load captures', 'error');
    }
  };

  useEffect(() => {
    if (user && vaultKey) {
      loadProjects();
      loadCaptures();
      setSearchQuery('');
      setIsSearching(false);
      setSearchResults([]);
    }
  }, [user, selectedProject, vaultKey]);

  // 3. Proactive duplicate warning while typing (whole docket, not project-scoped)
  useEffect(() => {
    const trimmed = formBody.trim();
    if (!vaultKey || trimmed.length < 8) {
      setDuplicateWarning([]);
      return;
    }

    if (duplicateDebounceTimer.current) clearTimeout(duplicateDebounceTimer.current);

    duplicateDebounceTimer.current = setTimeout(async () => {
      try {
        const matches = await findDuplicateMatches(trimmed, formTags, vaultKey, { limit: 5 });
        setDuplicateWarning(matches);
        const sig = matches.map((m) => m.id).join(',');
        if (matches.length > 0 && sig !== lastDupPrivacyRef.current) {
          lastDupPrivacyRef.current = sig;
          pushPrivacyEvent(createDuplicateCheckActivity(matches.length));
        } else if (matches.length === 0) {
          lastDupPrivacyRef.current = '';
        }
      } catch (err) {
        console.error('Proactive duplicate check failed:', err);
      }
    }, 500);

    return () => clearTimeout(duplicateDebounceTimer.current);
  }, [formBody, formTags, vaultKey, pushPrivacyEvent]);

  // 4. Semantic Search (debounced, whole docket)
  const searchMinSimilarity = user?.search_min_similarity ?? DEFAULT_SEARCH_MIN;

  const runSemanticSearch = useCallback(async (query) => {
    const trimmed = query.trim();
    if (!trimmed || !vaultKey) {
      setIsSearching(false);
      setSearchResults([]);
      return;
    }

    setIsSearching(true);

    try {
      const results = await searchCapturesForUser(vaultKey, trimmed, {
        minSimilarity: searchMinSimilarity,
        limit: 20,
      });
      setSearchResults(results);
      const exactCount = results.filter((r) => r.exactMatch).length;
      pushPrivacyEvent(createSearchActivity(trimmed, results.length, exactCount));
    } catch (err) {
      showToast('Semantic search failed — is the model loaded?', 'error');
    }
  }, [searchMinSimilarity, vaultKey, pushPrivacyEvent]);

  const scheduleSemanticSearch = (query) => {
    if (searchDebounceTimer.current) clearTimeout(searchDebounceTimer.current);
    if (!query.trim()) {
      runSemanticSearch('');
      return;
    }
    searchDebounceTimer.current = setTimeout(() => runSemanticSearch(query), 400);
  };

  // Check entire docket for duplicates before saving
  const findSimilarInDocket = async (body) =>
    findDuplicateMatches(body, formTags, vaultKey, { limit: 5 });

  const performSave = async () => {
    if (!vaultKey) return;
    setIsSaving(true);
    const project = formProject || selectedProject || 'Inbox';

    try {
      await saveCapture(vaultKey, {
        body: formBody,
        tags: formTags,
        project,
        sourceUrl: formSourceUrl,
      });
      showToast('Capture saved successfully');
      pushPrivacyEvent(createSaveActivity(project));
      setFormBody('');
      setFormTags('');
      setFormProject('');
      setFormSourceUrl('');
      setDuplicateWarning([]);
      setSaveDuplicateConfirm(null);
      setUser((u) => (u && !u.pro_access ? { ...u, capture_count: (u.capture_count ?? 0) + 1 } : u));
      loadCaptures();
      loadProjects();
    } catch (err) {
      if (err.status === 402) {
        showToast(`Free limit reached (${err.payload?.capture_count ?? '?'}/${err.payload?.capture_limit ?? '?'}). Upgrade in Account.`, 'error');
      } else {
        showToast('Save failed — check vault and model', 'error');
      }
    } finally {
      setIsSaving(false);
    }
  };

  // 6. Save new capture (with duplicate gate)
  const handleSaveCapture = async (e) => {
    e.preventDefault();
    if (!formBody.trim()) return;

    // Always re-check docket at save time — typing debounce may have missed a race
    const similar = await findSimilarInDocket(formBody);
    const strongMatches = similar.filter((item) => item.similarity >= SIMILARITY.DUPLICATE_SAVE);

    if (strongMatches.length > 0) {
      setSaveDuplicateConfirm(strongMatches);
      setDuplicateWarning(similar);
      return;
    }

    await performSave();
  };

  // 7. Update existing capture
  const handleUpdateCapture = async (e) => {
    e.preventDefault();
    if (!editingCapture || !editingCapture.body.trim() || !vaultKey) return;

    try {
      const title = editingCapture.title || generateAutoTitle(editingCapture.body);
      const tags = editingCapture.tags ?? [];
      const cType = classifyContentType(editingCapture.body);
      const embedding = await embedPassage(
        buildCaptureEmbeddingText(editingCapture.body, formatTagsInput(tags)),
      );
      const ciphertext = await encryptCapturePayload(vaultKey, {
        title,
        body: editingCapture.body,
        source_url: editingCapture.source_url ?? '',
        tags,
      });

      const res = await fetch(`/api/captures/${editingCapture.id}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          ciphertext,
          embedding,
          project: editingCapture.project,
          type: cType,
        }),
      });

      if (res.ok) {
        showToast('Capture updated');
        pushPrivacyEvent(createSaveActivity(editingCapture.project, { title: 'Capture updated' }));
        setEditingCapture(null);
        loadCaptures();
        loadProjects();
      } else {
        showToast('Failed to update capture', 'error');
      }
    } catch (err) {
      showToast('Update failed', 'error');
    }
  };

  // 8. Delete capture flow (Soft / Hard)
  const executeDelete = async () => {
    if (!deletingCapture) return;

    const isSoft = deletingCapture.deleted_at === 0 || !deletingCapture.deleted_at;

    try {
      const res = await fetch(`/api/captures/${deletingCapture.id}`, {
        method: 'DELETE',
      });

      if (res.ok) {
        showToast(isSoft ? 'Capture moved to Trash' : 'Capture permanently deleted');
        setDeletingCapture(null);
        setEditingCapture(null);
        loadCaptures();
        loadProjects();
      } else {
        showToast('Failed to delete capture', 'error');
      }
    } catch (err) {
      showToast('Database call failed', 'error');
    }
  };

  // 9. Restore capture
  const handleRestore = async (id) => {
    try {
      const res = await fetch(`/api/captures/restore/${id}`, {
        method: 'POST',
      });
      if (res.ok) {
        showToast('Capture restored from Trash');
        loadCaptures();
        loadProjects();
      } else {
        showToast('Failed to restore capture', 'error');
      }
    } catch (err) {
      showToast('Database call failed', 'error');
    }
  };

  // 10. Empty Trash
  const handleEmptyTrash = async () => {
    try {
      const res = await fetch('/api/captures/empty-trash', {
        method: 'POST',
      });
      if (res.ok) {
        const data = await res.json();
        showToast(`Trash emptied: ${data.count} items deleted permanently`);
        setEmptyTrashConfirm(false);
        loadCaptures();
        loadProjects();
      } else {
        showToast('Failed to empty Trash', 'error');
      }
    } catch (err) {
      showToast('Failed to empty Trash', 'error');
    }
  };

  if (authLoading) {
    return (
      <div className="page-loader">
        <div className="page-loader-inner">
          <div className="spinner" />
          <p>Loading your notes…</p>
        </div>
      </div>
    );
  }

  if (!user) return <Navigate to="/login" replace />;

  // Authenticated Screen
  const displayedItems = isSearching ? searchResults : captures;

  return (
    <VaultGate>
    <div className="notes-app">

      {/* TOAST CONTAINER */}
      {toast && (
        <div id="toast-container">
          <div className={`toast ${toast.type}`}>
            <i className={`fa-solid ${toast.type === 'success' ? 'fa-circle-check' : 'fa-circle-exclamation'}`}></i>
            <span>{toast.message}</span>
          </div>
        </div>
      )}

      {/* NAVBAR */}
      <header className="notes-header">
        <div className="notes-header-inner">
          <Link to="/" className="brand compact">
            <span className="brand-mark">n</span>
            <span className="brand-name">noteika</span>
          </Link>
          <button type="button" className="notes-project-btn" onClick={() => setSelectedProject('Inbox')}>
            Notes
          </button>
          <span className="badge badge-ready">
            Encrypted · hybrid search
          </span>
          <div className="notes-header-right">
            {user && !user.pro_access && user.capture_limit != null && (
              <span className="plan-usage-badge" title="Free plan capture limit">
                {user.capture_count ?? 0} / {user.capture_limit} notes
              </span>
            )}
            {user?.pro_access && (
              <span className="plan-usage-badge plan-usage-pro">Pro</span>
            )}
            <div className="user-info">
              <span className="user-name">{user.full_name || 'User'}</span>
              <span className="user-email">{user.email || ''}</span>
            </div>
            <Link to="/account" className="btn-icon-link" title="Account">
              <i className="fa-solid fa-user" />
            </Link>
            <a href="/auth/logout" className="btn-icon-link" title="Log out">
              <i className="fa-solid fa-right-from-bracket" />
            </a>
          </div>
        </div>
      </header>

      {/* DASHBOARD GRID */}
      <div className="dashboard-grid">
        {/* SIDEBAR */}
        <aside className="sidebar-column">
          <div className="glass-card sidebar-card">
            <h2><i className="fa-solid fa-folder-open"></i> Projects</h2>
            <ul className="project-list" id="project-list">
              {projects.map((proj) => (
                <li
                  key={proj}
                  className={selectedProject === proj ? 'active' : ''}
                  onClick={() => setSelectedProject(proj)}
                >
                  <span className="proj-name">
                    <i className="fa-regular fa-folder"></i> {proj}
                  </span>
                </li>
              ))}
              {/* Trash View Folder */}
              <li
                className={`project-trash ${selectedProject === 'Trash' ? 'active' : ''}`}
                onClick={() => setSelectedProject('Trash')}
                style={{ borderTop: '1px solid rgba(255, 255, 255, 0.08)', marginTop: '8px', paddingTop: '8px' }}
              >
                <span className="proj-name" style={{ color: 'var(--danger)' }}>
                  <i className="fa-regular fa-trash-can"></i> Trash
                </span>
              </li>
            </ul>
          </div>

          <PrivacyTrustCard />
        </aside>

        {/* FEED COLUMN */}
        <div className="feed-column">
          {/* SEARCH BAR */}
          <div className="search-container">
            <i className="fa-solid fa-magnifying-glass search-icon"></i>
            <input
              type="text"
              id="search-input"
              value={searchQuery}
              onChange={(e) => {
                setSearchQuery(e.target.value);
                scheduleSemanticSearch(e.target.value);
              }}
              placeholder="Search by meaning or exact words…"
            />
            {searchQuery && (
              <button
                id="btn-clear-search"
                className="btn-clear-search"
                onClick={() => {
                  setSearchQuery('');
                  setIsSearching(false);
                  setSearchResults([]);
                }}
              >
                <i className="fa-solid fa-xmark"></i>
              </button>
            )}
          </div>

          {isSearching && (
            <p className="search-scope-note">
              Searching all projects — meaning match from server, exact words on your device.
            </p>
          )}

          {/* CAPTURE FORM (Only show if not in Trash view) */}
          {selectedProject !== 'Trash' ? (
            <div className="glass-card capture-form-container">
              <form id="capture-form" onSubmit={handleSaveCapture}>
                <div className="form-group no-margin">
                  <textarea
                    id="capture-body"
                    rows="4"
                    value={formBody}
                    onChange={(e) => setFormBody(e.target.value)}
                    placeholder="Paste AI prompt, link, notes, Q&A, or code snippets here..."
                    required
                  ></textarea>
                </div>

                {/* PROACTIVE DUPLICATE WARNING PANEL */}
                {duplicateWarning.length > 0 && (
                  <div id="resurface-alert" className="resurface-alert">
                    <div className="resurface-alert-header">
                      <i className="fa-solid fa-triangle-exclamation"></i>
                      <span>Already in your docket — similar captures exist:</span>
                    </div>
                    <div id="resurface-list" className="resurface-list">
                      {duplicateWarning.map((item) => (
                        <div
                          key={item.capture.id}
                          className="resurface-item"
                          onClick={() => setEditingCapture(item.capture)}
                        >
                          <span className="resurface-item-title">{item.capture.title}</span>
                          <span className="resurface-item-score">
                            {item.capture.project} · {Math.round(item.similarity * 100)}% match
                          </span>
                        </div>
                      ))}
                    </div>
                  </div>
                )}

                <div className="capture-form-footer">
                  <div className="inline-inputs">
                    <div className="inline-input-group">
                      <i className="fa-solid fa-hashtag input-icon"></i>
                      <input
                        type="text"
                        id="capture-tags"
                        value={formTags}
                        onChange={(e) => setFormTags(e.target.value)}
                        placeholder="Tags (comma-separated, or use #hashtags in text)"
                        list="tag-suggestions"
                      />
                      <datalist id="tag-suggestions">
                        {knownTags.map((t) => (
                          <option key={t} value={t} />
                        ))}
                      </datalist>
                    </div>
                    <div className="inline-input-group">
                      <i className="fa-solid fa-tag input-icon"></i>
                      <input
                        type="text"
                        id="capture-project"
                        value={formProject}
                        onChange={(e) => setFormProject(e.target.value)}
                        placeholder="Project (default: Inbox)"
                        list="project-suggestions"
                      />
                      <datalist id="project-suggestions">
                        {projects.map((p) => (
                          <option key={p} value={p} />
                        ))}
                      </datalist>
                    </div>
                    <div className="inline-input-group">
                      <i className="fa-solid fa-link input-icon"></i>
                      <input
                        type="url"
                        id="capture-source"
                        value={formSourceUrl}
                        onChange={(e) => setFormSourceUrl(e.target.value)}
                        placeholder="Source URL (optional)"
                      />
                    </div>
                  </div>
                  <button type="submit" className="btn btn-primary" id="btn-save" disabled={isSaving}>
                    {isSaving ? (
                      <>
                        <i className="fa-solid fa-spinner fa-spin"></i> Saving...
                      </>
                    ) : (
                      <>
                        <i className="fa-solid fa-cloud-arrow-up"></i> Capture
                      </>
                    )}
                  </button>
                </div>
              </form>
            </div>
          ) : (
            /* TRASH VIEW HEADER EXTRA BUTTON */
            <div
              className="glass-card capture-form-container"
              style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', padding: '20px' }}
            >
              <div>
                <h3 style={{ color: 'var(--danger)', display: 'flex', alignItems: 'center', gap: '8px' }}>
                  <i className="fa-solid fa-trash-can"></i> Trash Bin
                </h3>
                <p style={{ fontSize: '13px', color: 'var(--text-muted)' }}>
                  Items here are soft-deleted. You can restore them or empty them permanently.
                </p>
              </div>
              <button
                className="btn btn-danger"
                disabled={captures.length === 0}
                onClick={() => setEmptyTrashConfirm(true)}
              >
                <i className="fa-solid fa-dumpster"></i> Empty Trash
              </button>
            </div>
          )}

          {/* CAPTURES SECTION */}
          <div className="captures-section">
            <div className="section-header">
              <h2 id="feed-title">
                <i className="fa-solid fa-stream"></i> {isSearching ? 'Search Results' : `${selectedProject} Feed`}
              </h2>
              <span className="captures-count" id="captures-count">
                {displayedItems.length} item{displayedItems.length !== 1 ? 's' : ''}
              </span>
            </div>

            {/* FEED LIST */}
            <div id="captures-feed" className="captures-feed-layout">
              {displayedItems.length === 0 ? (
                <div className="no-notes-placeholder">
                  <i className="fa-regular fa-folder-open"></i>
                  <p>
                    {isSearching
                      ? 'No matching semantic results found.'
                      : selectedProject === 'Trash'
                      ? 'Trash is empty.'
                      : 'No captures found in this project. Paste something above!'}
                  </p>
                </div>
              ) : (
                displayedItems.map((item) => {
                  const cap = isSearching ? item.capture : item;
                  if (!cap) return null;
                  const score = isSearching ? item.similarity : null;
                  const capType = cap.type || 'note';
                  const isHighMatch = score !== null && score > SIMILARITY.BANNER;
                  const body = cap.body ?? '';

                  // Custom Card Rendering based on type
                  let bodyHtml = null;
                  const bodySnippet = body.length > 250 ? body.substring(0, 250) + '...' : body;

                  if (capType === 'link') {
                    const urlVal = body.trim();
                    bodyHtml = (
                      <div className="link-card-body">
                        <a href={urlVal} target="_blank" rel="noreferrer" className="btn-visit-link link-out">
                          <i className="fa-solid fa-square-arrow-up-right"></i> Visit Link
                        </a>
                      </div>
                    );
                  } else if (capType === 'code') {
                    let codeText = body.trim();
                    if (codeText.startsWith('```') && codeText.endsWith('```')) {
                      const firstNewline = codeText.indexOf('\n');
                      if (firstNewline !== -1) {
                        codeText = codeText.substring(firstNewline + 1, codeText.length - 3).trim();
                      } else {
                        codeText = codeText.substring(3, codeText.length - 3).trim();
                      }
                    } else {
                      codeText = codeText.replace(/^```(?:\w*)\n?/, '').replace(/\n?```$/, '').trim();
                    }
                    const snippet = codeText.length > 250 ? codeText.substring(0, 250) + '...' : codeText;
                    bodyHtml = (
                      <pre>
                        <code>{snippet}</code>
                      </pre>
                    );
                  } else if (capType === 'qa') {
                    let qText = body;
                    let aText = '';
                    const lowerBody = body.toLowerCase();
                    let splitIdx = -1;

                    if (lowerBody.includes('q:') && lowerBody.includes('a:')) {
                      splitIdx = lowerBody.indexOf('a:');
                    } else if (lowerBody.includes('user:') && lowerBody.includes('assistant:')) {
                      splitIdx = lowerBody.indexOf('assistant:');
                    } else if (lowerBody.includes('user:') && lowerBody.includes('ai:')) {
                      splitIdx = lowerBody.indexOf('ai:');
                    } else if (lowerBody.includes('human:') && lowerBody.includes('assistant:')) {
                      splitIdx = lowerBody.indexOf('assistant:');
                    }

                    if (splitIdx !== -1) {
                      qText = body.substring(0, splitIdx).trim();
                      aText = body.substring(splitIdx).trim();

                      if (qText.toLowerCase().startsWith('q:')) qText = qText.substring(2).trim();
                      else if (qText.toLowerCase().startsWith('user:')) qText = qText.substring(5).trim();
                      else if (qText.toLowerCase().startsWith('human:')) qText = qText.substring(6).trim();

                      if (aText.toLowerCase().startsWith('a:')) aText = aText.substring(2).trim();
                      else if (aText.toLowerCase().startsWith('assistant:')) aText = aText.substring(10).trim();
                      else if (aText.toLowerCase().startsWith('ai:')) aText = aText.substring(3).trim();

                      if (qText.length > 120) qText = qText.substring(0, 120) + '...';
                      if (aText.length > 150) aText = aText.substring(0, 150) + '...';

                      bodyHtml = (
                        <div className="qa-layout">
                          <div className="qa-question">
                            <i className="fa-regular fa-comments"></i> {qText}
                          </div>
                          <div className="qa-answer">{aText}</div>
                        </div>
                      );
                    } else {
                      bodyHtml = (
                        <p className="capture-card-content">
                          {bodySnippet.split('\n').map((line, idx) => (
                            <React.Fragment key={idx}>
                              {line}
                              <br />
                            </React.Fragment>
                          ))}
                        </p>
                      );
                    }
                  } else {
                    bodyHtml = (
                      <p className="capture-card-content">
                        {bodySnippet.split('\n').map((line, idx) => (
                          <React.Fragment key={idx}>
                            {line}
                            <br />
                          </React.Fragment>
                        ))}
                      </p>
                    );
                  }

                  return (
                    <div
                      key={cap.id}
                      className={`glass-card capture-card fade-in capture-card-${capType} ${isHighMatch ? 'high-match' : ''}`}
                      onClick={() => setEditingCapture(cap)}
                    >
                      <div className="capture-card-header">
                        <div className="card-title-area">
                          <h3>{cap.title}</h3>
                          <div className="card-badges">
                            <span className="card-badge project-badge">{cap.project}</span>
                            {(cap.tags || []).map((tag) => (
                              <span key={tag} className="capture-tag">
                                <i className="fa-solid fa-hashtag" />{tag}
                              </span>
                            ))}
                            {score !== null && (
                              <span className={`card-badge score-badge ${isHighMatch ? 'glow-green' : ''}`}>
                                <i className="fa-solid fa-brain"></i> Match: {Math.round(score * 100)}%
                              </span>
                            )}
                            {isSearching && item.exactMatch && (
                              <span className="card-badge exact-badge">
                                <i className="fa-solid fa-font"></i> Exact
                              </span>
                            )}
                          </div>
                        </div>
                        <div className="card-actions">
                          {cap.source_url && (
                            <a
                              href={cap.source_url}
                              target="_blank"
                              rel="noreferrer"
                              className="link-out"
                              title="Visit Source"
                              onClick={(e) => e.stopPropagation()}
                            >
                              <i className="fa-solid fa-arrow-up-right-from-square"></i>
                            </a>
                          )}
                          {selectedProject === 'Trash' ? (
                            <>
                              <button
                                className="btn-restore"
                                title="Restore Capture"
                                style={{
                                  background: 'none',
                                  border: 'none',
                                  color: 'var(--success)',
                                  cursor: 'pointer',
                                  marginRight: '8px',
                                  fontSize: '14px',
                                }}
                                onClick={(e) => {
                                  e.stopPropagation();
                                  handleRestore(cap.id);
                                }}
                              >
                                <i className="fa-solid fa-rotate-left"></i>
                              </button>
                              <button
                                className="btn-delete"
                                title="Delete Permanently"
                                onClick={(e) => {
                                  e.stopPropagation();
                                  setDeletingCapture(cap);
                                }}
                              >
                                <i className="fa-solid fa-trash-can"></i>
                              </button>
                            </>
                          ) : (
                            <button
                              className="btn-delete"
                              title="Delete"
                              onClick={(e) => {
                                e.stopPropagation();
                                setDeletingCapture(cap);
                              }}
                            >
                              <i className="fa-regular fa-trash-can"></i>
                            </button>
                          )}
                        </div>
                      </div>

                      {bodyHtml}

                      <div className="capture-card-footer">
                        <span>
                          <i className="fa-regular fa-clock"></i> {formatRelativeTime(cap.created_at)}
                        </span>
                      </div>
                    </div>
                  );
                })
              )}
            </div>
          </div>

          <PrivacyActivityPanel events={privacyEvents} />
        </div>
      </div>

      {/* EDIT MODAL */}
      {editingCapture && (
        <div id="edit-modal" className="modal-overlay">
          <div className="glass-card modal-card fade-in">
            <div className="modal-header">
              <h2><i className="fa-solid fa-pen-to-square"></i> Edit Capture</h2>
              <button className="btn-close-modal" onClick={() => setEditingCapture(null)}>
                <i className="fa-solid fa-xmark"></i>
              </button>
            </div>
            <div className="modal-body">
              <form id="edit-form" onSubmit={handleUpdateCapture}>
                <div className="form-group">
                  <label htmlFor="edit-title">Title</label>
                  <input
                    type="text"
                    id="edit-title"
                    value={editingCapture.title}
                    onChange={(e) => setEditingCapture({ ...editingCapture, title: e.target.value })}
                    placeholder="Auto-generated title (leave blank to keep auto-generation)"
                  />
                </div>
                <div className="form-group">
                  <label htmlFor="edit-body">Content Body</label>
                  <textarea
                    id="edit-body"
                    rows="8"
                    value={editingCapture.body}
                    onChange={(e) => setEditingCapture({ ...editingCapture, body: e.target.value })}
                    required
                  ></textarea>
                </div>
                <div className="form-group">
                  <label htmlFor="edit-tags">Tags</label>
                  <input
                    type="text"
                    id="edit-tags"
                    value={formatTagsInput(editingCapture.tags)}
                    onChange={(e) =>
                      setEditingCapture({
                        ...editingCapture,
                        tags: parseTagsInput(e.target.value),
                      })
                    }
                    placeholder="rustfs, s3, oauth"
                    list="edit-tag-suggestions"
                  />
                  <datalist id="edit-tag-suggestions">
                    {knownTags.map((t) => (
                      <option key={t} value={t} />
                    ))}
                  </datalist>
                  <p className="tags-hint">Smart tags are added automatically when you save. Use #hashtags in the body too.</p>
                </div>
                <div className="modal-inline-row">
                  <div className="form-group">
                    <label htmlFor="edit-project">Project</label>
                    <input
                      type="text"
                      id="edit-project"
                      value={editingCapture.project || ''}
                      onChange={(e) => setEditingCapture({ ...editingCapture, project: e.target.value })}
                      required
                    />
                  </div>
                  <div className="form-group">
                    <label htmlFor="edit-source">Source URL</label>
                    <input
                      type="url"
                      id="edit-source"
                      value={editingCapture.source_url || ''}
                      onChange={(e) => setEditingCapture({ ...editingCapture, source_url: e.target.value })}
                      placeholder="https://..."
                    />
                  </div>
                </div>
                <div className="modal-actions">
                  <button
                    type="button"
                    className="btn btn-danger-action"
                    onClick={() => {
                      setDeletingCapture(editingCapture);
                    }}
                  >
                    <i className="fa-solid fa-trash-can"></i> Delete
                  </button>
                  <button type="submit" className="btn btn-primary" id="btn-modal-save">
                    <i className="fa-solid fa-save"></i> Save Changes
                  </button>
                </div>
              </form>
            </div>
          </div>
        </div>
      )}

      {/* SAVE DUPLICATE CONFIRMATION MODAL */}
      {saveDuplicateConfirm && (
        <div className="modal-overlay" style={{ zIndex: 1000 }}>
          <div className="glass-card modal-card fade-in" style={{ maxWidth: '520px' }}>
            <div className="modal-header" style={{ borderBottom: '1px solid rgba(245, 158, 11, 0.3)' }}>
              <h2 style={{ color: 'var(--warning)', display: 'flex', alignItems: 'center', gap: '8px' }}>
                <i className="fa-solid fa-clone"></i> Already in your docket
              </h2>
              <button className="btn-close-modal" onClick={() => setSaveDuplicateConfirm(null)}>
                <i className="fa-solid fa-xmark"></i>
              </button>
            </div>
            <div className="modal-body" style={{ padding: '16px 0' }}>
              <p style={{ marginBottom: '12px', color: 'var(--text-muted)' }}>
                You already saved something very similar. View the existing capture instead of duplicating yourself.
              </p>
              <div className="resurface-list">
                {saveDuplicateConfirm.map((item) => (
                  <div
                    key={item.capture.id}
                    className="resurface-item"
                    onClick={() => {
                      setSaveDuplicateConfirm(null);
                      setEditingCapture(item.capture);
                    }}
                  >
                    <span className="resurface-item-title">{item.capture.title}</span>
                    <span className="resurface-item-score">
                      {item.capture.project} · {Math.round(item.similarity * 100)}% match · {formatRelativeTime(item.capture.created_at)}
                    </span>
                  </div>
                ))}
              </div>
            </div>
            <div className="modal-actions" style={{ justifyContent: 'flex-end', gap: '12px' }}>
              <button className="btn" onClick={() => setSaveDuplicateConfirm(null)}>
                Cancel
              </button>
              <button
                className="btn btn-danger-action"
                style={{ background: 'transparent', borderColor: 'var(--text-muted)', color: 'var(--text-muted)' }}
                disabled={isSaving}
                onClick={() => performSave()}
              >
                Save anyway
              </button>
            </div>
          </div>
        </div>
      )}

      {/* DELETE CONFIRMATION MODAL */}
      {deletingCapture && (
        <div className="modal-overlay" style={{ zIndex: 1000 }}>
          <div className="glass-card modal-card fade-in" style={{ maxWidth: '450px' }}>
            <div className="modal-header" style={{ borderBottom: '1px solid rgba(239, 68, 68, 0.2)' }}>
              <h2 style={{ color: 'var(--danger)', display: 'flex', alignItems: 'center', gap: '8px' }}>
                <i className="fa-solid fa-triangle-exclamation"></i> Confirm Deletion
              </h2>
              <button className="btn-close-modal" onClick={() => setDeletingCapture(null)}>
                <i className="fa-solid fa-xmark"></i>
              </button>
            </div>
            <div className="modal-body" style={{ padding: '24px 0' }}>
              {deletingCapture.deleted_at > 0 ? (
                <p>
                  Are you sure you want to <strong>permanently delete</strong> the capture{' '}
                  <strong>"{deletingCapture.title}"</strong>?
                  <br />
                  <span style={{ color: 'var(--danger)', fontSize: '13px', marginTop: '8px', display: 'block' }}>
                    <i className="fa-solid fa-triangle-exclamation"></i> Warning: This action is irreversible.
                  </span>
                </p>
              ) : (
                <p>
                  Are you sure you want to delete <strong>"{deletingCapture.title}"</strong>?
                  <br />
                  <span style={{ color: 'var(--text-muted)', fontSize: '13px', marginTop: '8px', display: 'block' }}>
                    It will be moved to the Trash bin and can be restored later.
                  </span>
                </p>
              )}
            </div>
            <div className="modal-actions" style={{ justifyContent: 'flex-end', gap: '12px' }}>
              <button className="btn" onClick={() => setDeletingCapture(null)}>
                Cancel
              </button>
              <button
                className="btn btn-danger-solid"
                onClick={executeDelete}
              >
                Confirm Delete
              </button>
            </div>
          </div>
        </div>
      )}

      {/* EMPTY TRASH CONFIRMATION MODAL */}
      {emptyTrashConfirm && (
        <div className="modal-overlay" style={{ zIndex: 1000 }}>
          <div className="glass-card modal-card fade-in" style={{ maxWidth: '450px' }}>
            <div className="modal-header" style={{ borderBottom: '1px solid rgba(239, 68, 68, 0.2)' }}>
              <h2 style={{ color: 'var(--danger)', display: 'flex', alignItems: 'center', gap: '8px' }}>
                <i className="fa-solid fa-dumpster"></i> Empty Trash Bin?
              </h2>
              <button className="btn-close-modal" onClick={() => setEmptyTrashConfirm(false)}>
                <i className="fa-solid fa-xmark"></i>
              </button>
            </div>
            <div className="modal-body" style={{ padding: '24px 0' }}>
              <p>
                Are you sure you want to <strong>permanently delete all items</strong> in the Trash?
                <br />
                <span style={{ color: 'var(--danger)', fontSize: '13px', marginTop: '8px', display: 'block' }}>
                  <i className="fa-solid fa-triangle-exclamation"></i> Warning: All trash will be permanently cleared. This cannot be undone.
                </span>
              </p>
            </div>
            <div className="modal-actions" style={{ justifyContent: 'flex-end', gap: '12px' }}>
              <button className="btn" onClick={() => setEmptyTrashConfirm(false)}>
                Cancel
              </button>
              <button
                className="btn btn-danger-solid"
                onClick={handleEmptyTrash}
              >
                Empty Trash Permanently
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
    </VaultGate>
  );
}
