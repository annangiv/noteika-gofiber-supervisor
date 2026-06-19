import PrivacyFlowDiagram, { PrivacyFlowIdle } from './PrivacyFlowDiagram';

function formatTime(date) {
  return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
}

function whereLabel(where) {
  if (where === 'device') return 'Your device';
  if (where === 'server') return 'Server';
  return 'Note';
}

export default function PrivacyTrustCard() {
  return (
    <div className="glass-card privacy-trust-card">
      <h2><i className="fa-solid fa-shield-halved" /> Your data</h2>
      <PrivacyFlowIdle />
      <p className="privacy-flow-disclaimer">
        Schematic only — never shows your passcode, keys, or note text.
      </p>
      <ul className="privacy-trust-list">
        <li>
          <span className="privacy-trust-icon device"><i className="fa-solid fa-lock" /></span>
          <div>
            <strong>Notes stay encrypted</strong>
            <p>Readable text never stored on our servers — only your vault passcode can unlock it.</p>
          </div>
        </li>
        <li>
          <span className="privacy-trust-icon server"><i className="fa-solid fa-fingerprint" /></span>
          <div>
            <strong>Search uses fingerprints</strong>
            <p>Embedding vectors enable semantic search. Encrypted at rest; not your words.</p>
          </div>
        </li>
        <li>
          <span className="privacy-trust-icon plain"><i className="fa-solid fa-tag" /></span>
          <div>
            <strong>Plain metadata only</strong>
            <p>Project name, save date, and account info — not note content.</p>
          </div>
        </li>
      </ul>
    </div>
  );
}

export function PrivacyActivityPanel({ events }) {
  const latest = events[0] ?? null;

  return (
    <div className="glass-card privacy-activity-panel">
      <div className="privacy-activity-header">
        <h2><i className="fa-solid fa-route" /> Privacy activity</h2>
        <span className="privacy-activity-hint">Live trace when you save or search</span>
      </div>

      {events.length === 0 ? (
        <>
          <PrivacyFlowIdle />
          <p className="privacy-activity-empty">
            Save or search a note to see exactly what stays on your device and what the server stores.
          </p>
          <p className="privacy-flow-disclaimer">
            Safe to show — this is the public architecture, not secret keys or ciphertext.
          </p>
        </>
      ) : (
        <>
          <PrivacyFlowDiagram eventType={latest.type} animationKey={latest.id} />
          <p className="privacy-flow-disclaimer">
            Animated schematic — your passcode and plaintext never appear here.
          </p>
          <div className="privacy-activity-list">
            {events.map((event) => (
            <article key={event.id} className={`privacy-event privacy-event-${event.type}`}>
              <div className="privacy-event-top">
                <span className="privacy-event-type">
                  {event.type === 'save' && <i className="fa-solid fa-floppy-disk" />}
                  {event.type === 'search' && <i className="fa-solid fa-magnifying-glass" />}
                  {event.type === 'duplicate' && <i className="fa-solid fa-clone" />}
                  {event.title}
                </span>
                <time>{formatTime(event.time)}</time>
              </div>
              {event.subtitle && <p className="privacy-event-sub">{event.subtitle}</p>}
              <ol className="privacy-steps">
                {event.steps.map((step, i) => (
                  <li key={i} className={`privacy-step privacy-step-${step.where}`}>
                    <span className="privacy-step-badge">{whereLabel(step.where)}</span>
                    <div>
                      <strong>{step.label}</strong>
                      <p>{step.detail}</p>
                    </div>
                  </li>
                ))}
              </ol>
            </article>
            ))}
          </div>
        </>
      )}
    </div>
  );
}
