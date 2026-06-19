import React, { useState, useEffect, useRef } from 'react';

// Relative time formatting utility
function formatRelativeTime(epoch) {
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

function escapeHtml(text) {
  if (!text) return '';
  return text
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#039;');
}

export default function App() {
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
  const [formProject, setFormProject] = useState('');
  const [formSourceUrl, setFormSourceUrl] = useState('');
  const [isSaving, setIsSaving] = useState(false);

  // Proactive duplicate resurfacing states
  const [duplicateWarning, setDuplicateWarning] = useState([]);
  const debounceTimer = useRef(null);

  // Sticky banner for top search result
  const [stickyBannerCapture, setStickyBannerCapture] = useState(null);

  // Telemetry logs states
  const [telemetry, setTelemetry] = useState({ restarts: 0, mailbox_addr: '0x00000000', status: 'Unknown' });
  const [logs, setLogs] = useState([]);
  const prevRestarts = useRef(0);
  const prevStatus = useRef('');

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

  const addLog = (message, type = 'info') => {
    const timeStr = new Date().toLocaleTimeString();
    setLogs((prev) => [...prev, { time: timeStr, message, type }]);
  };

  // 1. Authenticate on mount
  useEffect(() => {
    async function checkAuth() {
      try {
        const res = await fetch('/api/auth/me');
        if (res.ok) {
          const data = await res.json();
          setUser(data);
          addLog(`[System] Supervised user session loaded: ${data.email}`, 'system');
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
        setProjects(data);
      }
    } catch (err) {
      console.error('Failed to load projects:', err);
    }
  };

  const loadCaptures = async () => {
    try {
      let url = '/api/captures';
      if (selectedProject) {
        url += `?project=${encodeURIComponent(selectedProject)}`;
      }
      const res = await fetch(url);
      if (res.ok) {
        const data = await res.json();
        setCaptures(data);
      }
    } catch (err) {
      showToast('Failed to load captures', 'error');
    }
  };

  useEffect(() => {
    if (user) {
      loadProjects();
      loadCaptures();
      setSearchQuery('');
      setIsSearching(false);
      setSearchResults([]);
      setStickyBannerCapture(null);
    }
  }, [user, selectedProject]);

  // 3. Telemetry Log Polling
  useEffect(() => {
    if (!user) return;

    const interval = setInterval(async () => {
      try {
        const res = await fetch('/api/debug/stats');
        if (res.ok) {
          const stats = await res.json();
          setTelemetry(stats);

          // Log supervisor changes
          if (prevRestarts.current !== undefined && stats.restarts > prevRestarts.current) {
            addLog(`[WARNING] VaultActor restarted! Supervisor recovered state. Restarts: ${stats.restarts}`, 'warning');
            showToast('Database recovery occurred!', 'error');
            loadCaptures();
            loadProjects();
          }
          prevRestarts.current = stats.restarts;

          if (prevStatus.current && stats.status !== prevStatus.current) {
            addLog(`[SYSTEM] Supervisor state changed: ${prevStatus.current} -> ${stats.status}`, 'system');
          }
          prevStatus.current = stats.status;
        }
      } catch (err) {
        console.error('Telemetry fetch failed:', err);
      }
    }, 1500);

    return () => clearInterval(interval);
  }, [user]);

  // 4. Proactive duplicate warning check (debounced)
  useEffect(() => {
    if (formBody.trim().length <= 15) {
      setDuplicateWarning([]);
      return;
    }

    if (debounceTimer.current) clearTimeout(debounceTimer.current);

    debounceTimer.current = setTimeout(async () => {
      try {
        const res = await fetch('/api/captures/search', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ query: formBody }),
        });
        if (res.ok) {
          const data = await res.json();
          // Filter matching capture items > 65% similarity
          const warningItems = data.filter((item) => item.similarity > 0.65);
          setDuplicateWarning(warningItems);
        }
      } catch (err) {
        console.error('Proactive search failed:', err);
      }
    }, 600);

    return () => clearTimeout(debounceTimer.current);
  }, [formBody]);

  // 5. Semantic Search
  const runSemanticSearch = async (query) => {
    if (!query.trim()) {
      setIsSearching(false);
      setSearchResults([]);
      setStickyBannerCapture(null);
      return;
    }

    setIsSearching(true);
    addLog(`[API] Dispatching semantic query: "${query}"`, 'info');

    try {
      const res = await fetch('/api/captures/search', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ query }),
      });
      if (res.ok) {
        const results = await res.json();
        setSearchResults(results);

        // Sticky banner check (if top result similarity is > 65%)
        if (results.length > 0 && results[0].similarity > 0.65) {
          setStickyBannerCapture(results[0]);
        } else {
          setStickyBannerCapture(null);
        }
        addLog(`[API] Found ${results.length} semantic matches. Top match score: ${results[0] ? Math.round(results[0].similarity * 100) : 0}%`, 'info');
      } else {
        showToast('Semantic search query failed', 'error');
      }
    } catch (err) {
      showToast('Connection to embedding search failed', 'error');
    }
  };

  // 6. Save new capture
  const handleSaveCapture = async (e) => {
    e.preventDefault();
    if (!formBody.trim()) return;

    setIsSaving(true);
    addLog(`[API] Saving new capture to project "${formProject || 'Inbox'}"...`, 'info');

    try {
      const res = await fetch('/api/captures', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          body: formBody,
          project: formProject,
          source_url: formSourceUrl,
        }),
      });

      if (res.ok) {
        showToast('Capture saved successfully');
        setFormBody('');
        setFormProject('');
        setFormSourceUrl('');
        setDuplicateWarning([]);
        loadCaptures();
        loadProjects();
      } else {
        showToast('Failed to save capture', 'error');
      }
    } catch (err) {
      showToast('Server connection failed', 'error');
    } finally {
      setIsSaving(false);
    }
  };

  // 7. Update existing capture
  const handleUpdateCapture = async (e) => {
    e.preventDefault();
    if (!editingCapture || !editingCapture.body.trim()) return;

    addLog(`[API] Updating capture "${editingCapture.id}"...`, 'info');

    try {
      const res = await fetch(`/api/captures/${editingCapture.id}`, {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          title: editingCapture.title,
          body: editingCapture.body,
          project: editingCapture.Project, // note spelling: fiber maps it to Project on JSON output or input depending on tags
          source_url: editingCapture.source_url,
        }),
      });

      if (res.ok) {
        showToast('Capture updated');
        setEditingCapture(null);
        loadCaptures();
        loadProjects();
      } else {
        showToast('Failed to update capture', 'error');
      }
    } catch (err) {
      showToast('Server communication failed', 'error');
    }
  };

  // 8. Delete capture flow (Soft / Hard)
  const executeDelete = async () => {
    if (!deletingCapture) return;

    const isSoft = deletingCapture.deleted_at === 0 || !deletingCapture.deleted_at;
    addLog(`[API] Executing delete for "${deletingCapture.id}" (Soft: ${isSoft})...`, 'info');

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
    addLog(`[API] Restoring capture "${id}"...`, 'info');
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
    addLog('[API] Emptying entire Trash folder...', 'warning');
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

  // 11. Supervised crash trigger
  const triggerCrash = async () => {
    addLog('[SYSTEM] Dispatching debug/crash command to VaultActor...', 'warning');
    try {
      await fetch('/api/debug/crash', { method: 'POST' });
    } catch (err) {
      // The actor crashed, so fetch fails or gets closed. This is expected.
    }
  };

  // Loader state
  if (authLoading) {
    return (
      <main id="app">
        <div id="app-loader" className="loader-container">
          <div className="spinner"></div>
          <p>Bootstrapping supervised node...</p>
        </div>
      </main>
    );
  }

  // Unauthenticated: Login Screen
  if (!user) {
    return (
      <main id="app">
        <div className="stars"></div>
        <div className="ambient-glow g1"></div>
        <div className="ambient-glow g2"></div>

        <section id="auth-screen" className="glass-card fade-in">
          <div className="auth-header">
            <div className="logo">
              <i className="fa-solid fa-brain logo-icon"></i>
              <h1>noteika</h1>
            </div>
            <p className="subtitle">Notes that resurface before you duplicate yourself</p>
          </div>

          <div className="auth-body">
            <p className="instruction">Select a provider to authenticate securely</p>
            <div className="btn-group">
              <a href="/auth/login/github" className="btn btn-github">
                <i className="fa-brands fa-github"></i> Authenticate with GitHub
              </a>
              <a href="/auth/login/google" className="btn btn-google">
                <i className="fa-brands fa-google"></i> Authenticate with Google
              </a>
              <a href="/auth/login/github?mock=true" className="btn btn-mock" id="btn-dev-login">
                <i className="fa-solid fa-code"></i> Developer Mock Login
              </a>
            </div>
          </div>
          <div className="auth-footer">
            <p>Go Fiber OTP Supervisor App</p>
          </div>
        </section>
      </main>
    );
  }

  // Authenticated Screen
  const displayedItems = isSearching ? searchResults : captures;

  return (
    <main id="app">
      <div className="stars"></div>
      <div className="ambient-glow g1"></div>
      <div className="ambient-glow g2"></div>

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
      <nav className="navbar glass-card">
        <div className="nav-left">
          <div className="logo logo-sm" onClick={() => setSelectedProject('Inbox')}>
            <i className="fa-solid fa-brain logo-icon"></i>
            <h1>noteika</h1>
          </div>
          <span className="badge badge-success-glow" id="model-status-badge">
            <i className="fa-solid fa-circle-check"></i> Semantic Search Ready
          </span>
        </div>
        <div className="nav-right" id="user-profile">
          <div className="user-info">
            <span id="user-name" className="user-name">{user.full_name || 'Developer'}</span>
            <span id="user-email" className="user-email">{user.email || 'dev@example.com'}</span>
          </div>
          <a href="/auth/logout" className="btn-logout">
            <i className="fa-solid fa-right-from-bracket"></i>
          </a>
        </div>
      </nav>

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

          <div className="glass-card telemetry-stats-container">
            <h2><i className="fa-solid fa-gauge-high"></i> Actor Telemetry</h2>
            <div className="stats-grid">
              <div className="stat-card">
                <span className="stat-label">Supervisor Status</span>
                <span className={`stat-value ${telemetry.status === 'Running' ? 'text-glow-green' : 'text-glow-purple'}`}>
                  {telemetry.status}
                </span>
              </div>
              <div className="stat-card">
                <span className="stat-label">Actor Restarts</span>
                <span className="stat-value text-glow-purple">{telemetry.restarts}</span>
              </div>
              <div className="stat-card full-width">
                <span className="stat-label">Mailbox Channel Address</span>
                <span className="stat-value text-mono text-glow-blue">{telemetry.mailbox_addr}</span>
              </div>
              <button id="btn-crash-actor" className="btn btn-danger" onClick={triggerCrash}>
                <i className="fa-solid fa-skull-crossbones"></i> Trigger Actor Crash
              </button>
            </div>
          </div>
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
                runSemanticSearch(e.target.value);
              }}
              placeholder="Search captures by meaning (e.g. 'oauth redirect handling')..."
            />
            {searchQuery && (
              <button
                id="btn-clear-search"
                className="btn-clear-search"
                onClick={() => {
                  setSearchQuery('');
                  setIsSearching(false);
                  setSearchResults([]);
                  setStickyBannerCapture(null);
                }}
              >
                <i className="fa-solid fa-xmark"></i>
              </button>
            )}
          </div>

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
                      <span>Similar captures already exist:</span>
                    </div>
                    <div id="resurface-list" className="resurface-list">
                      {duplicateWarning.map((item) => (
                        <div
                          key={item.capture.id}
                          className="resurface-item"
                          onClick={() => setEditingCapture(item.capture)}
                        >
                          <span className="resurface-title">{item.capture.title}</span>
                          <span className="resurface-score">Match: {Math.round(item.similarity * 100)}%</span>
                        </div>
                      ))}
                    </div>
                  </div>
                )}

                <div className="capture-form-footer">
                  <div className="inline-inputs">
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

            {/* STICKY SEARCH RESURFACE BANNER */}
            {isSearching && stickyBannerCapture && (
              <div id="search-resurface-banner" className="resurface-banner">
                <div className="resurface-banner-left">
                  <i className="fa-solid fa-lightbulb"></i>
                  <span id="resurface-banner-text">
                    You saved a similar capture on {new Date(stickyBannerCapture.capture.created_at * 1000).toLocaleDateString()}:
                    '<strong>{stickyBannerCapture.capture.title}</strong>' (Match: {Math.round(stickyBannerCapture.similarity * 100)}%)
                  </span>
                </div>
                <button
                  id="btn-resurface-view"
                  className="btn-resurface-view"
                  onClick={() => setEditingCapture(stickyBannerCapture.capture)}
                >
                  View Original
                </button>
              </div>
            )}

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
                  const score = isSearching ? item.similarity : null;
                  const capType = cap.type || 'note';
                  const isHighMatch = score !== null && score > 0.65;

                  // Custom Card Rendering based on type
                  let bodyHtml = null;
                  const bodySnippet = cap.body.length > 250 ? cap.body.substring(0, 250) + '...' : cap.body;

                  if (capType === 'link') {
                    const urlVal = cap.body.trim();
                    bodyHtml = (
                      <div className="link-card-body">
                        <a href={urlVal} target="_blank" rel="noreferrer" className="btn-visit-link link-out">
                          <i className="fa-solid fa-square-arrow-up-right"></i> Visit Link
                        </a>
                      </div>
                    );
                  } else if (capType === 'code') {
                    let codeText = cap.body.trim();
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
                    let qText = cap.body;
                    let aText = '';
                    const lowerBody = cap.body.toLowerCase();
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
                      qText = cap.body.substring(0, splitIdx).trim();
                      aText = cap.body.substring(splitIdx).trim();

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
                            <span className="card-badge project-badge">{cap.Project}</span>
                            {score !== null && (
                              <span className={`card-badge score-badge ${isHighMatch ? 'glow-green' : ''}`}>
                                <i className="fa-solid fa-brain"></i> Match: {Math.round(score * 100)}%
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

          {/* TELEMETRY LOGS */}
          <div className="glass-card logs-panel">
            <div className="logs-header">
              <h2><i className="fa-solid fa-terminal"></i> Supervisor Event Log</h2>
              <button id="btn-clear-logs" className="btn-icon" onClick={() => setLogs([])}>
                <i className="fa-solid fa-trash-can"></i>
              </button>
            </div>
            <div id="log-terminal" className="terminal-body">
              {logs.length === 0 ? (
                <div className="log-line system">[SYSTEM] Supervisor connected. Monitoring actor 'VaultActor'.</div>
              ) : (
                logs.map((log, idx) => (
                  <div key={idx} className={`log-line ${log.type}`}>
                    [{log.time}] {log.message}
                  </div>
                ))
              )}
            </div>
          </div>
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
                <div className="modal-inline-row">
                  <div className="form-group">
                    <label htmlFor="edit-project">Project</label>
                    <input
                      type="text"
                      id="edit-project"
                      value={editingCapture.Project}
                      onChange={(e) => setEditingCapture({ ...editingCapture, Project: e.target.value })}
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
                className="btn btn-danger-action"
                style={{ background: 'var(--danger)', borderColor: 'var(--danger)' }}
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
                className="btn btn-danger-action"
                style={{ background: 'var(--danger)', borderColor: 'var(--danger)' }}
                onClick={handleEmptyTrash}
              >
                Empty Trash Permanently
              </button>
            </div>
          </div>
        </div>
      )}
    </main>
  );
}
