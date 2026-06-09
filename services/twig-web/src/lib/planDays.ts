import { todayString, addDays, formatDayLabel } from "./planView";

export function todayIso(): string {
  return todayString();
}

export function tomorrowIso(): string {
  return addDays(todayIso(), 1);
}

export function dayPickerLabel(iso: string): string {
  const today = todayIso();
  const tomorrow = tomorrowIso();
  if (iso === today) return "Today";
  if (iso === tomorrow) return "Tomorrow";
  return formatDayLabel(iso);
}
