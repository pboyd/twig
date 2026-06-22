const STORAGE_KEY = "twig-show-hidden-goals";

export function readShowHiddenGoals(): boolean {
  try {
    return localStorage.getItem(STORAGE_KEY) === "true";
  } catch {
    return false;
  }
}

export function writeShowHiddenGoals(value: boolean): void {
  try {
    localStorage.setItem(STORAGE_KEY, String(value));
  } catch {}
}
