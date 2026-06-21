import { useEffect, useState } from 'react';
import { Link, useNavigate, useSearchParams } from 'react-router-dom';
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
  const [billingLoading, setBillingLoading] = useState(false);
  const [billingNotice, setBillingNotice] = useState('');
  const [searchParams, setSearchParams] = useSearchParams();
  const navigate = useNavigate();

  const loadUser = () =>
    fetch('/api/auth/me')
      .then((r) => (r.ok ? r.json() : Promise.reject()))
      .then((data) => {
        setUser(data);
        if (data.search_min_similarity) {
          setSearchMinPct(Math.round(data.search_min_similarity * 100));
        }
      });

  useEffect(() => {
    loadUser()
      .catch(() => navigate('/login'))
      .finally(() => setLoading(false));
  }, [navigate]);

  useEffect(() => {
    const billing = searchParams.get('billing');
    if (billing === 'success') {
      setBillingNotice('Welcome to Pro — your account is upgraded.');
      loadUser().catch(() => {});
      setSearchParams({}, { replace: true });
    } else if (billing === 'cancel') {
      setBillingNotice('Checkout canceled.');
      setSearchParams({}, { replace: true });
    }
  }, [searchParams, setSearchParams]);

  const startCheckout = async () => {
    setBillingLoading(true);
    setBillingNotice('');
    try {
      const res = await fetch('/api/billing/checkout', { method: 'POST' });
      const data = await res.json();
      if (!res.ok) throw new Error(data.error || 'checkout failed');
      window.location.href = data.url;
    } catch (err) {
      setBillingNotice(err.message || 'Could not start checkout.');
      setBillingLoading(false);
    }
  };

  const openPortal = async () => {
    setBillingLoading(true);
    setBillingNotice('');
    try {
      const res = await fetch('/api/billing/portal', { method: 'POST' });
      const data = await res.json();
      if (!res.ok) throw new Error(data.error || 'portal failed');
      window.location.href = data.url;
    } catch (err) {
      setBillingNotice(err.message || 'Could not open billing portal.');
      setBillingLoading(false);
    }
  };

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

  const isPro = user?.pro_access;
  const atLimit = !isPro && user?.capture_count >= user?.capture_limit;

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

          {billingNotice && (
            <p className="settings-saved" style={{ marginBottom: '16px' }}>{billingNotice}</p>
          )}

          <div className="account-card">
            <h2>Profile</h2>
            <dl className="account-dl">
              <dt>Name</dt>
              <dd>{user?.full_name || '—'}</dd>
              <dt>Email</dt>
              <dd>{user?.email || '—'}</dd>
              <dt>Plan</dt>
              <dd className="capitalize">{isPro ? 'Pro' : 'Free'}</dd>
              <dt>Notes saved</dt>
              <dd>
                {user?.capture_count ?? 0}
                {!isPro && user?.capture_limit ? ` / ${user.capture_limit} free` : ' (unlimited)'}
              </dd>
              <dt>Member since</dt>
              <dd>{user?.created_at ? new Date(user.created_at * 1000).toLocaleDateString() : '—'}</dd>
            </dl>
          </div>

          <div className="account-card">
            <h2>Billing</h2>
            {isPro ? (
              <>
                <p className="account-desc">
                  You have Pro — unlimited encrypted captures and semantic search.
                </p>
                {user?.stripe_enabled && user?.has_stripe_customer && (
                  <button
                    type="button"
                    className="btn btn-secondary"
                    onClick={openPortal}
                    disabled={billingLoading}
                  >
                    {billingLoading ? 'Opening…' : 'Manage subscription'}
                  </button>
                )}
              </>
            ) : (
              <>
                <p className="account-desc">
                  Free plan includes {user?.capture_limit ?? 10} encrypted captures.
                  {atLimit ? ' You’ve reached the limit — upgrade to keep saving.' : ' Search stays unlimited on your notes.'}
                </p>
                {user?.stripe_enabled ? (
                  <button
                    type="button"
                    className="btn btn-primary"
                    onClick={startCheckout}
                    disabled={billingLoading}
                  >
                    {billingLoading ? 'Redirecting…' : 'Upgrade to Pro — $8/mo'}
                  </button>
                ) : (
                  <p className="settings-hint">Billing is not configured on this server yet.</p>
                )}
              </>
            )}
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
            <h2>Import{!isPro && ' (Pro)'}</h2>
            <p className="account-desc">
              {isPro
                ? 'Bulk-import up to 100 markdown files at once — handy for migrating an existing corpus.'
                : 'Bulk-import up to 100 markdown files at once. Upgrade to Pro to unlock this.'}
            </p>
            <Link to="/import" className="btn btn-secondary">
              {isPro ? 'Import notes' : 'Learn more'}
            </Link>
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
