/** Decorative hex — not real ciphertext. */
const GLYPH = {
  note: 'note',
  blob: '7a3f…c2e1',
  vec: '9b02…4d8a',
  query: '1f8c…6e00',
  match: '4d1a…b903',
};

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
        <span className="privacy-flow-node-label">Device</span>
        {isSave && (
          <span className="privacy-flow-glyph chip-animate-lock">
            <i className="fa-solid fa-lock" />
            <span>{GLYPH.blob}</span>
          </span>
        )}
        {isSearch && (
          <span className="privacy-flow-glyph">
            <i className="fa-solid fa-magnifying-glass" />
            <span>{GLYPH.query}</span>
          </span>
        )}
      </div>

      <div className="privacy-flow-tunnel">
        <div className="privacy-flow-tls">
          <i className="fa-solid fa-shield-halved" />
          <span>TLS</span>
        </div>
        <div className="privacy-flow-track">
          {isSave && (
            <>
              <span className="privacy-flow-packet packet-cipher">{GLYPH.blob}</span>
              <span className="privacy-flow-packet packet-vector">{GLYPH.vec}</span>
            </>
          )}
          {isSearch && (
            <span className="privacy-flow-packet packet-query">{GLYPH.query}</span>
          )}
        </div>
        {isSearch && (
          <div className="privacy-flow-track privacy-flow-track-return">
            <span className="privacy-flow-packet packet-return">{GLYPH.match}</span>
          </div>
        )}
      </div>

      <div className="privacy-flow-node privacy-flow-server">
        <div className="privacy-flow-node-icon">
          <i className="fa-solid fa-server" />
        </div>
        <span className="privacy-flow-node-label">Server</span>
        {isSave && (
          <span className="privacy-flow-vault chip-animate-store">
            <i className="fa-solid fa-lock" />
            <span className="privacy-flow-blocks">████</span>
          </span>
        )}
        {isSearch && (
          <span className="privacy-flow-vault">
            <i className="fa-solid fa-chart-simple" />
            <span>rank</span>
          </span>
        )}
      </div>

      {isSearch && (
        <div className="privacy-flow-unlock chip-animate-unlock">
          <i className="fa-solid fa-lock-open" />
          <span>unlock here</span>
        </div>
      )}
    </div>
  );
}

export function PrivacyFlowIdle() {
  return (
    <div className="privacy-flow privacy-flow-idle" aria-hidden="true">
      <div className="privacy-flow-node privacy-flow-device">
        <div className="privacy-flow-node-icon"><i className="fa-solid fa-laptop" /></div>
        <span className="privacy-flow-node-label">Device</span>
        <span className="privacy-flow-glyph"><i className="fa-solid fa-lock" /><span>{GLYPH.blob}</span></span>
      </div>
      <div className="privacy-flow-tunnel privacy-flow-tunnel-idle">
        <div className="privacy-flow-tls"><i className="fa-solid fa-shield-halved" /><span>TLS</span></div>
        <div className="privacy-flow-track-idle" />
      </div>
      <div className="privacy-flow-node privacy-flow-server">
        <div className="privacy-flow-node-icon"><i className="fa-solid fa-server" /></div>
        <span className="privacy-flow-node-label">Server</span>
        <span className="privacy-flow-vault"><i className="fa-solid fa-lock" /><span className="privacy-flow-blocks">████</span></span>
      </div>
    </div>
  );
}
