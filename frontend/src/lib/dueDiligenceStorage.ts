/**
 * localStorage persistence for per-property due diligence checklist state.
 * Storage key format: `yg_dd_<propertyKey>`
 */

export type DueDiligenceState = Record<string, boolean>;

function storageKey(propertyKey: string): string {
  return `yg_dd_${propertyKey}`;
}

/**
 * Load checklist state for a given property key.
 * Returns an empty object if nothing is stored or parsing fails.
 */
export function getChecklist(propertyKey: string): DueDiligenceState {
  if (typeof window === "undefined") return {};
  try {
    const raw = localStorage.getItem(storageKey(propertyKey));
    if (!raw) return {};
    return JSON.parse(raw) as DueDiligenceState;
  } catch {
    return {};
  }
}

/**
 * Persist checklist state for a given property key.
 */
export function saveChecklist(propertyKey: string, state: DueDiligenceState): void {
  if (typeof window === "undefined") return;
  localStorage.setItem(storageKey(propertyKey), JSON.stringify(state));
}
