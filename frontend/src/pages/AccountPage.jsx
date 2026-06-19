import { useEffect, useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import {
  DEFAULT_SEARCH_MIN,
  SEARCH_MIN_FLOOR,
  SEARCH_MIN_CEILING,
  searchSensitivityHint,
} from '../lib/notesApi';

export default function AccountPage() {
  const [user, setUser] = useState(null);
  const [loading, setLoading] = useState(true);
  const [exporting, setExporting] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [confirmDelete, setConfirmDelete] = useState(false);
  const [deleteInput, setDeleteInput] = useState('');
  const [searchMinPct, setSearchMinPct] = useState(Math.round(DEFAULT_SEARCH_MIN * 100));
  const [settingsSaving, setSettingsSaving] = useState(false);
  const [settingsSaved, setSettingsSaved] = useState(false);
  const navigate = useNavigate();

  useEffect(() => {
    fetch('/api/auth/me')
      .then((r) => (r.ok ? r.json() : Promise.reject()))
      .then((data) => {
        setUser(data);
        if (data.search_min_similarity) {
          setSearchMinPct(Math.round(data.search_min_similarity * 100));
        }
      })
      .catch(() => navigate('/login'))
      .finally(() => setLoading(false));
  }, [navigate]);

  const handleSaveSettings = async () => {
    setSettingsSaving(true);
    setSettingsSaved(false);
    try {
      const res = await fetch('/api/account/settings', {
        method: 'PATCH',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ search_min_similarity: searchMinPct / 100 }),
      });
      if (!res.ok) throw new Error('save failed');
      const data = await res.json();
      setUser((u) => ({ ...u, search_min_similarity: data.search_min_similarity }));
      setSettingsSaved(true);
      setTimeout(() => setSettingsSaved(false), 3000);
    } catch {
      alert('Could not save settings. Please try again.');
    } finally {
      setSettingsSaving(false);
    }
  };

  const handleResetSearchDefault = () => {
    setSearchMinPct(Math.round(DEFAULT_SEARCH_MIN * 100));
  };

  const handleExport = async () => {
    setExporting(true);
    try {
      const res = await fetch('/api/account/export');
      if (!res.ok) throw new Error('export failed');
      const blob = await res.blob();
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a');
      a.href = url;
      a.download = 'noteika-export.json';
      a.click();
      URL.revokeObjectURL(url);
    } catch {
      alert('Export failed. Please try again.');
    } finally {
      setExporting(false);
    }
  };

  const handleDelete = async () => {
    if (deleteInput !== 'DELETE') return;
    setDeleting(true);
    try {
      const res = await fetch('/api/account', { method: 'DELETE' });
      if (!res.ok) throw new Error('delete failed');
      navigate('/');
    } catch {
      alert('Could not delete account. Please try again.');
      setDeleting(false);
    }
  };

  if (loading) {
    return (
      <div className="page-loader">
        <div className="page-loader-inner"><div className="spinner" /></div>
      </div>
    );
  }

  return (
    <div className="site">
      <header className="site-header">
        <div className="container site-header-inner">
          <Link to="/" className="brand">
            <span className="brand-mark">n</span>
            <span className="brand-name">noteika</span>
          </Link>
          <nav className="site-nav">
            <Link to="/notes" className="nav-link">← Back to notes</Link>
          </nav>
        </div>
      </header>

      <main className="section">
        <div className="container narrow">
          <p className="eyebrow">Account</p>
          <h1>Your account</h1>

          <div className="account-card">
            <h2>Profile</h2>
            <dl className="account-dl">
              <dt>Name</dt>
              <dd>{user?.full_name || '—'}</dd>
              <dt>Email</dt>
              <dd>{user?.email || '—'}</dd>
              <dt>Plan</dt>
              <dd className="capitalize">{user?.tier || 'free'}</dd>
              <dt>Member since</dt>
              <dd>{user?.created_at ? new Date(user.created_at * 1000).toLocaleDateString() : '—'}</dd>
            </dl>
          </div>

          <div className="account-card">
            <h2>Search sensitivity</h2>
            <p className="account-desc">
              How closely a capture must match your query to appear in search results.
              Duplicate warnings while typing use a separate, lower threshold.
            </p>
            <div className="settings-slider-row">
              <label htmlFor="search-min-slider" className="settings-slider-label">
                Minimum match: <strong>{searchMinPct}%</strong>
              </label>
              <input
                id="search-min-slider"
                type="range"
                min={SEARCH_MIN_FLOOR * 100}
                max={SEARCH_MIN_CEILING * 100}
                step={5}
                value={searchMinPct}
                onChange={(e) => setSearchMinPct(Number(e.target.value))}
                className="settings-slider"
              />
              <div className="settings-slider-ticks">
                <span>50% broad</span>
                <span>70% default</span>
                <span>85% strict</span>
              </div>
            </div>
            <p className="settings-hint">{searchSensitivityHint(searchMinPct)}</p>
            <div className="settings-actions">
              <button
                type="button"
                className="btn btn-secondary btn-sm"
                onClick={handleResetSearchDefault}
              >
                Reset to 70%
              </button>
              <button
                type="button"
                className="btn btn-primary btn-sm"
                onClick={handleSaveSettings}
                disabled={settingsSaving}
              >
                {settingsSaving ? 'Saving…' : 'Save search settings'}
              </button>
              {settingsSaved && <span className="settings-saved">Saved</span>}
            </div>
          </div>

          <div className="account-card">
            <h2>Data</h2>
            <p className="account-desc">Download all your captures as a JSON file.</p>
            <button type="button" className="btn btn-secondary" onClick={handleExport} disabled={exporting}>
              {exporting ? 'Preparing export…' : 'Download my data'}
            </button>
          </div>

          <div className="account-card">
            <h2>Session</h2>
            <p className="account-desc">Sign out on this device.</p>
            <a href="/auth/logout" className="btn btn-secondary">Log out</a>
          </div>

          <div className="account-card account-danger">
            <h2>Delete account</h2>
            <p className="account-desc">
              Permanently delete your account and all captures. This cannot be undone.
            </p>
            {!confirmDelete ? (
              <button type="button" className="btn btn-danger" onClick={() => setConfirmDelete(true)}>
                Delete my account
              </button>
            ) : (
              <div className="delete-confirm">
                <p>Type <strong>DELETE</strong> to confirm:</p>
                <input
                  type="text"
                  value={deleteInput}
                  onChange={(e) => setDeleteInput(e.target.value)}
                  placeholder="DELETE"
                  className="input"
                />
                <div className="delete-actions">
                  <button type="button" className="btn btn-secondary" onClick={() => { setConfirmDelete(false); setDeleteInput(''); }}>
                    Cancel
                  </button>
                  <button
                    type="button"
                    className="btn btn-danger"
                    disabled={deleteInput !== 'DELETE' || deleting}
                    onClick={handleDelete}
                  >
                    {deleting ? 'Deleting…' : 'Permanently delete'}
                  </button>
                </div>
              </div>
            )}
          </div>
        </div>
      </main>
    </div>
  );
}
