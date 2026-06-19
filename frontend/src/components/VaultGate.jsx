import { useState } from 'react';
import { useVault } from '../context/VaultContext';
import '../notes.css';

export default function VaultGate({ children }) {
  const {
    isUnlocked, needsSetup, loading, embedLoading, error, setError, unlock, setup,
  } = useVault();
  const [passcode, setPasscode] = useState('');
  const [confirm, setConfirm] = useState('');

  if (isUnlocked) {
    if (embedLoading) {
      return (
        <div className="page-loader">
          <div className="page-loader-inner">
            <div className="spinner" />
            <p>Loading semantic search model…</p>
          </div>
        </div>
      );
    }
    return children;
  }

  const onSubmit = async (e) => {
    e.preventDefault();
    setError('');
    if (needsSetup) {
      await setup(passcode, confirm);
    } else {
      await unlock(passcode);
    }
  };

  return (
    <div className="vault-gate">
      <div className="glass-card vault-card">
        <h1>
          <i className="fa-solid fa-lock" /> {needsSetup ? 'Create vault passcode' : 'Unlock your vault'}
        </h1>
        <p className="vault-desc">
          {needsSetup
            ? 'Your notes are encrypted on this device before they reach our servers. Choose a passcode — we cannot recover it if you lose it.'
            : 'Enter your vault passcode to decrypt and search your notes.'}
        </p>
        <form onSubmit={onSubmit}>
          <div className="form-group">
            <label htmlFor="vault-pass">Vault passcode</label>
            <input
              id="vault-pass"
              type="password"
              className="input"
              value={passcode}
              onChange={(e) => setPasscode(e.target.value)}
              autoComplete={needsSetup ? 'new-password' : 'current-password'}
              required
              minLength={8}
            />
          </div>
          {needsSetup && (
            <div className="form-group">
              <label htmlFor="vault-confirm">Confirm passcode</label>
              <input
                id="vault-confirm"
                type="password"
                className="input"
                value={confirm}
                onChange={(e) => setConfirm(e.target.value)}
                autoComplete="new-password"
                required
                minLength={8}
              />
            </div>
          )}
          {error && <p className="vault-error">{error}</p>}
          <button type="submit" className="btn btn-primary btn-block" disabled={loading}>
            {loading ? 'Working…' : needsSetup ? 'Create vault' : 'Unlock'}
          </button>
        </form>
      </div>
    </div>
  );
}
