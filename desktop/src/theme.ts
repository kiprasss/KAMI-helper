const SERVER_KEY = "kami-server";

export function loadServer(): string | null {
  try {
    return localStorage.getItem(SERVER_KEY);
  } catch {
    return null;
  }
}

export function saveServer(code: string) {
  try {
    localStorage.setItem(SERVER_KEY, code);
  } catch {
    return;
  }
}

export function isServerKey(key: string | null): boolean {
  return key === SERVER_KEY;
}

export function applyTheme(code: string | null) {
  if (code) {
    document.documentElement.dataset.theme = code;
  } else {
    delete document.documentElement.dataset.theme;
  }
}

export function applySurfaceAlpha(percent: number) {
  document.documentElement.style.setProperty("--surface-alpha", String(percent / 100));
}
