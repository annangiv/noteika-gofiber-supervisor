/** Triggers sidebar flow animation only — no step text (see Your data card). */
export function privacyFlowEvent(type) {
  return { id: `${Date.now()}-${type}`, type };
}
