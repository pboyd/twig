const STORAGE_KEY = "twig-show-completed";

export function readShowCompleted(): boolean {
  try {
    return localStorage.getItem(STORAGE_KEY) === "true";
  } catch {
    return false;
  }
}

export function writeShowCompleted(value: boolean): void {
  try {
    localStorage.setItem(STORAGE_KEY, String(value));
  } catch {}
}
