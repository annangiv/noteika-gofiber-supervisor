import PrivacyFlowDiagram, { PrivacyFlowIdle } from './PrivacyFlowDiagram';

export default function PrivacyTrustCard({ activeFlow }) {
  return (
    <div className="glass-card privacy-trust-card">
      <h2><i className="fa-solid fa-shield-halved" /> Your data</h2>
      {activeFlow
        ? <PrivacyFlowDiagram eventType={activeFlow.type} animationKey={activeFlow.id} />
        : <PrivacyFlowIdle />}
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
            <p>Save date and account info — not note content or project names.</p>
          </div>
        </li>
      </ul>
    </div>
  );
}
