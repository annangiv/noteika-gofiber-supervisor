/** Schematic only — no keys, ciphertext, or wire-format details. */
export default function PrivacyFlowDiagram({ eventType, animationKey }) {
  const isSave = eventType === 'save';
  const isSearch = eventType === 'search' || eventType === 'duplicate';

  return (
    <div
      key={animationKey}
      className={`privacy-flow privacy-flow-${eventType}`}
      aria-hidden="true"
    >
      <div className="privacy-flow-node privacy-flow-device">
        <div className="privacy-flow-node-icon">
          <i className="fa-solid fa-laptop" />
        </div>
        <span className="privacy-flow-node-label">Your device</span>
        <div className="privacy-flow-device-items">
          {isSave && (
            <>
              <span className="privacy-flow-chip chip-local">
                <i className="fa-solid fa-file-lines" /> note
              </span>
              <span className="privacy-flow-chip chip-lock chip-animate-lock">
                <i className="fa-solid fa-lock" /> encrypted
              </span>
              <span className="privacy-flow-chip chip-local">
                <i className="fa-solid fa-fingerprint" /> embedding
              </span>
            </>
          )}
          {isSearch && (
            <>
              <span className="privacy-flow-chip chip-local">
                <i className="fa-solid fa-magnifying-glass" /> query
              </span>
              <span className="privacy-flow-chip chip-local chip-animate-lock">
                <i className="fa-solid fa-fingerprint" /> query vector
              </span>
              <span className="privacy-flow-chip chip-unlock chip-animate-unlock">
                <i className="fa-solid fa-lock-open" /> decrypted here
              </span>
            </>
          )}
        </div>
      </div>

      <div className="privacy-flow-tunnel">
        <div className="privacy-flow-tls">
          <i className="fa-solid fa-shield-halved" />
          <span>TLS</span>
        </div>
        <div className="privacy-flow-track">
          {isSave && (
            <>
              <span className="privacy-flow-packet packet-cipher">
                <i className="fa-solid fa-lock" /> ciphertext
              </span>
              <span className="privacy-flow-packet packet-vector">
                <i className="fa-solid fa-lock" /> vector
              </span>
            </>
          )}
          {isSearch && (
            <span className="privacy-flow-packet packet-query">
              <i className="fa-solid fa-lock" /> vector
            </span>
          )}
        </div>
        {isSearch && (
          <div className="privacy-flow-track privacy-flow-track-return">
            <span className="privacy-flow-packet packet-return">
              <i className="fa-solid fa-lock" /> matches
            </span>
          </div>
        )}
      </div>

      <div className="privacy-flow-node privacy-flow-server">
        <div className="privacy-flow-node-icon">
          <i className="fa-solid fa-server" />
        </div>
        <span className="privacy-flow-node-label">Server</span>
        <div className="privacy-flow-server-store">
          {isSave && (
            <>
              <span className="privacy-flow-chip chip-stored chip-animate-store">
                <i className="fa-solid fa-lock" /> ciphertext at rest
              </span>
              <span className="privacy-flow-chip chip-stored chip-animate-store delay">
                <i className="fa-solid fa-lock" /> vector at rest
              </span>
            </>
          )}
          {isSearch && (
            <span className="privacy-flow-chip chip-stored">
              <i className="fa-solid fa-chart-simple" /> rank only
            </span>
          )}
        </div>
      </div>
    </div>
  );
}

export function PrivacyFlowIdle() {
  return (
    <div className="privacy-flow privacy-flow-idle" aria-hidden="true">
      <div className="privacy-flow-node privacy-flow-device">
        <div className="privacy-flow-node-icon"><i className="fa-solid fa-laptop" /></div>
        <span className="privacy-flow-node-label">Your device</span>
        <span className="privacy-flow-chip chip-local"><i className="fa-solid fa-lock" /> encrypt & embed</span>
      </div>
      <div className="privacy-flow-tunnel privacy-flow-tunnel-idle">
        <div className="privacy-flow-tls"><i className="fa-solid fa-shield-halved" /><span>TLS</span></div>
        <div className="privacy-flow-track-idle" />
      </div>
      <div className="privacy-flow-node privacy-flow-server">
        <div className="privacy-flow-node-icon"><i className="fa-solid fa-server" /></div>
        <span className="privacy-flow-node-label">Server</span>
        <span className="privacy-flow-chip chip-stored"><i className="fa-solid fa-lock" /> locked blobs</span>
      </div>
    </div>
  );
}
