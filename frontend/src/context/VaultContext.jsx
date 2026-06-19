import { createContext, useCallback, useContext, useMemo, useState } from 'react';
import { deriveVaultKey, fetchVaultSalt } from '../lib/crypto';
import { preloadEmbedder } from '../lib/embeddings';

const VaultContext = createContext(null);

const VAULT_SETUP_KEY = 'noteika_vault_setup_done';

export function VaultProvider({ children }) {
  const [vaultKey, setVaultKey] = useState(null);
  const [salt, setSalt] = useState(null);
  const [loading, setLoading] = useState(false);
  const [embedLoading, setEmbedLoading] = useState(false);
  const [error, setError] = useState('');

  const ensureSalt = useCallback(async () => {
    if (salt) return salt;
    const s = await fetchVaultSalt();
    setSalt(s);
    return s;
  }, [salt]);

  const unlock = useCallback(async (passcode) => {
    setLoading(true);
    setError('');
    try {
      const s = await ensureSalt();
      const key = await deriveVaultKey(passcode, s);
      setVaultKey(key);
      setEmbedLoading(true);
      preloadEmbedder()
        .catch((err) => console.warn('Embedding model preload failed:', err))
        .finally(() => setEmbedLoading(false));
      return true;
    } catch (err) {
      setError(err.message || 'Unlock failed');
      return false;
    } finally {
      setLoading(false);
    }
  }, [ensureSalt]);

  const setup = useCallback(async (passcode, confirm) => {
    if (passcode.length < 8) {
      setError('Passcode must be at least 8 characters');
      return false;
    }
    if (passcode !== confirm) {
      setError('Passcodes do not match');
      return false;
    }
    const ok = await unlock(passcode);
    if (ok) {
      localStorage.setItem(VAULT_SETUP_KEY, '1');
    }
    return ok;
  }, [unlock]);

  const lock = useCallback(() => {
    setVaultKey(null);
  }, []);

  const value = useMemo(() => ({
    vaultKey,
    isUnlocked: vaultKey !== null,
    needsSetup: !localStorage.getItem(VAULT_SETUP_KEY),
    loading,
    embedLoading,
    error,
    setError,
    unlock,
    setup,
    lock,
    ensureSalt,
  }), [vaultKey, loading, embedLoading, error, unlock, setup, lock, ensureSalt]);

  return (
    <VaultContext.Provider value={value}>
      {children}
    </VaultContext.Provider>
  );
}

export function useVault() {
  const ctx = useContext(VaultContext);
  if (!ctx) throw new Error('useVault must be used within VaultProvider');
  return ctx;
}
