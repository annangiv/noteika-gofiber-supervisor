import { createContext, useCallback, useContext, useMemo, useState, useEffect } from 'react';
import { deriveVaultKeys, fetchVaultParams, encryptCapturePayload, decryptCapturePayload } from '../lib/crypto';
import { preloadEmbedder } from '../lib/embeddings';
import { deriveAndSetMatrix, clearMatrix } from '../lib/fingerprint';

const VaultContext = createContext(null);

const VAULT_SETUP_KEY = 'noteika_vault_setup_done';

export function VaultProvider({ children }) {
  const [vaultKey, setVaultKey] = useState(null);
  const [salt, setSalt] = useState(null);
  const [verifier, setVerifier] = useState(null);
  const [needsSetup, setNeedsSetup] = useState(() => {
    return !localStorage.getItem(VAULT_SETUP_KEY);
  });
  const [loading, setLoading] = useState(false);
  const [embedLoading, setEmbedLoading] = useState(false);
  const [error, setError] = useState('');

  const ensureVaultParams = useCallback(async () => {
    if (salt) return { salt, verifier };
    const params = await fetchVaultParams();
    setSalt(params.salt);
    setVerifier(params.verifier);

    // If verifier exists, user has set up the vault
    if (params.verifier) {
      setNeedsSetup(false);
    } else {
      // If verifier is null, check if they have notes (capture_count > 0)
      if (!localStorage.getItem(VAULT_SETUP_KEY)) {
        try {
          const uRes = await fetch('/api/auth/me');
          if (uRes.ok) {
            const u = await uRes.json();
            if (u.capture_count > 0) {
              setNeedsSetup(false);
            }
          }
        } catch (e) {
          console.warn('Failed to fetch user count for needsSetup check:', e);
        }
      }
    }

    return params;
  }, [salt, verifier]);

  useEffect(() => {
    async function init() {
      try {
        await ensureVaultParams();
      } catch (err) {
        console.warn('Failed to fetch vault params on mount:', err);
      }
    }
    init();
  }, [ensureVaultParams]);

  const unlock = useCallback(async (passcode) => {
    setLoading(true);
    setError('');
    try {
      const { salt: s, verifier: v } = await ensureVaultParams();
      const { vaultKey: derivedKey, hkdfKey } = await deriveVaultKeys(passcode, s);

      if (v) {
        // Verify key against the stored verifier
        try {
          const dec = await decryptCapturePayload(derivedKey, v);
          if (!dec || !dec.verified) {
            throw new Error('Verifier structure mismatch');
          }
        } catch (err) {
          throw new Error('Incorrect vault passcode');
        }
      } else {
        // Legacy vault with no verifier. Silently create and upload one.
        try {
          const verifierPayload = { verified: true };
          const encryptedVerifier = await encryptCapturePayload(derivedKey, verifierPayload);

          await fetch('/api/vault/verifier', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ verifier: encryptedVerifier }),
          });

          setVerifier(encryptedVerifier);
        } catch (e) {
          console.warn('Failed to silently upload legacy verifier:', e);
        }
      }

      await deriveAndSetMatrix(hkdfKey);
      setVaultKey(derivedKey);
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
  }, [ensureVaultParams]);

  const setup = useCallback(async (passcode, confirm) => {
    if (passcode.length < 8) {
      setError('Passcode must be at least 8 characters');
      return false;
    }
    if (passcode !== confirm) {
      setError('Passcodes do not match');
      return false;
    }
    
    setLoading(true);
    setError('');
    try {
      const { salt: s } = await ensureVaultParams();
      const { vaultKey: derivedKey, hkdfKey } = await deriveVaultKeys(passcode, s);

      const verifierPayload = { verified: true };
      const encryptedVerifier = await encryptCapturePayload(derivedKey, verifierPayload);

      const res = await fetch('/api/vault/verifier', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ verifier: encryptedVerifier }),
      });
      if (!res.ok) {
        throw new Error('Failed to save vault verifier to server');
      }

      setVerifier(encryptedVerifier);
      await deriveAndSetMatrix(hkdfKey);
      setVaultKey(derivedKey);
      localStorage.setItem(VAULT_SETUP_KEY, '1');
      setNeedsSetup(false);

      setEmbedLoading(true);
      preloadEmbedder()
        .catch((err) => console.warn('Embedding model preload failed:', err))
        .finally(() => setEmbedLoading(false));
      return true;
    } catch (err) {
      setError(err.message || 'Setup failed');
      return false;
    } finally {
      setLoading(false);
    }
  }, [ensureVaultParams]);

  const lock = useCallback(() => {
    setVaultKey(null);
    clearMatrix();
  }, []);

  const value = useMemo(() => ({
    vaultKey,
    isUnlocked: vaultKey !== null,
    needsSetup,
    loading,
    embedLoading,
    error,
    setError,
    unlock,
    setup,
    lock,
    ensureSalt: ensureVaultParams,
  }), [vaultKey, needsSetup, loading, embedLoading, error, unlock, setup, lock, ensureVaultParams]);

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
