// Navigation visibility control utility
// Used by toolbars on detail pages to toggle the global navigation bar

const NAV_HIDDEN_KEY = "flowforge-nav-hidden";
const NAV_TOGGLE_EVENT = "flowforge-nav-toggle";
const DETAIL_MODE_EVENT = "flowforge-detail-mode";

// Track if current page is a detail page (for skill editor which doesn't change URL)
let _isInDetailMode = false;

export function isNavHidden(): boolean {
  if (typeof window === "undefined") return false;
  return localStorage.getItem(NAV_HIDDEN_KEY) === "true";
}

export function setNavHidden(hidden: boolean): void {
  localStorage.setItem(NAV_HIDDEN_KEY, hidden ? "true" : "false");
  // Dispatch custom event for same-tab listeners
  window.dispatchEvent(new CustomEvent(NAV_TOGGLE_EVENT, { detail: { hidden } }));
}

export function toggleNav(): void {
  setNavHidden(!isNavHidden());
}

export function onNavToggle(callback: (hidden: boolean) => void): () => void {
  const handler = (e: Event) => {
    const detail = (e as CustomEvent).detail;
    callback(detail.hidden);
  };
  window.addEventListener(NAV_TOGGLE_EVENT, handler);
  return () => window.removeEventListener(NAV_TOGGLE_EVENT, handler);
}

/**
 * Set detail mode (used by skill editor which doesn't change URL)
 */
export function setDetailMode(isDetail: boolean): void {
  _isInDetailMode = isDetail;
  window.dispatchEvent(new CustomEvent(DETAIL_MODE_EVENT, { detail: { isDetail } }));
}

export function isInDetailMode(): boolean {
  return _isInDetailMode;
}

export function onDetailModeChange(callback: (isDetail: boolean) => void): () => void {
  const handler = (e: Event) => {
    const detail = (e as CustomEvent).detail;
    callback(detail.isDetail);
  };
  window.addEventListener(DETAIL_MODE_EVENT, handler);
  return () => window.removeEventListener(DETAIL_MODE_EVENT, handler);
}

/**
 * Check if current pathname is a detail page (workflow editor)
 * For skill editor, use setDetailMode() instead
 */
export function isDetailPage(pathname: string): boolean {
  // Workflow editor: /workflows/[id]
  if (pathname.startsWith("/workflows/") && pathname !== "/workflows") {
    return true;
  }
  return false;
}
