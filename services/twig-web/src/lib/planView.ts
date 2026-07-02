import type { PlanEntry } from "../gen/plan/v1/plan_pb";

export interface ResolvedEntry {
  id: number;
  displayName: string;
  kind: "task" | "event";
  taskId?: bigint;
  completed: boolean;
  timed: boolean;
  startMinute?: number;
  endMinute?: number;
  timeLabel?: string;
  durationMinute: number;
}

export interface GroupedPlan {
  timed: ResolvedEntry[];
  untimed: ResolvedEntry[];
  isEmpty: boolean;
}

export function todayString(): string {
  const d = new Date();
  const y = d.getFullYear();
  const m = String(d.getMonth() + 1).padStart(2, "0");
  const day = String(d.getDate()).padStart(2, "0");
  return `${y}-${m}-${day}`;
}

export function addDays(dayString: string, delta: number): string {
  const [y, m, d] = dayString.split("-").map(Number);
  const date = new Date(y, m - 1, d + delta);
  const ny = date.getFullYear();
  const nm = String(date.getMonth() + 1).padStart(2, "0");
  const nd = String(date.getDate()).padStart(2, "0");
  return `${ny}-${nm}-${nd}`;
}

export function isToday(dayString: string): boolean {
  return dayString === todayString();
}

export function formatDayLabel(dayString: string): string {
  const [y, m, d] = dayString.split("-").map(Number);
  const date = new Date(y, m - 1, d);
  const dayNames = ["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"];
  const monthNames = ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"];
  const dow = dayNames[date.getDay()];
  const mon = monthNames[date.getMonth()];
  const base = `${dow} ${mon} ${d}`;
  return isToday(dayString) ? `Today · ${base}` : base;
}

export function formatMinute(min: number): string {
  const h24 = Math.floor(min / 60);
  const mins = min % 60;
  const ampm = h24 < 12 ? "am" : "pm";
  const h12 = h24 % 12 === 0 ? 12 : h24 % 12;
  return `${h12}:${String(mins).padStart(2, "0")} ${ampm}`;
}

export function formatTimeRange(startMinute: number, durationMinute: number): string {
  const end = startMinute + durationMinute;
  return `${formatMinute(startMinute)} – ${formatMinute(end)}`;
}

export function resolveEntries(
  entries: PlanEntry[],
  taskNameById: Map<bigint, string>
): ResolvedEntry[] {
  return entries.map((e) => {
    const isTask = e.taskId !== 0n;
    const displayName =
      e.name.trim() ||
      (isTask ? taskNameById.get(e.taskId) ?? "Untitled entry" : "Untitled entry");
    const timed = e.startMinute !== undefined;
    const startMinute = timed ? e.startMinute! : undefined;
    const endMinute = timed ? startMinute! + e.durationMinute : undefined;
    const timeLabel = timed ? formatTimeRange(startMinute!, e.durationMinute) : undefined;
    return {
      id: e.id,
      displayName,
      kind: isTask ? "task" : "event",
      taskId: isTask ? e.taskId : undefined,
      completed: e.completed,
      timed,
      startMinute,
      endMinute,
      timeLabel,
      durationMinute: e.durationMinute,
    };
  });
}

export function groupPlan(entries: ResolvedEntry[]): GroupedPlan {
  const timed = entries.filter((e) => e.timed);
  const untimed = entries.filter((e) => !e.timed && !e.completed);
  return { timed, untimed, isEmpty: timed.length === 0 && untimed.length === 0 };
}

export const SLOT_MINUTES = 15;
export const SLOT_PX = 44;

export function snapDown15(min: number): number {
  return Math.floor(min / SLOT_MINUTES) * SLOT_MINUTES;
}

export function snapUp15(min: number): number {
  return Math.ceil(min / SLOT_MINUTES) * SLOT_MINUTES;
}

export interface TimelineWindow {
  startMinute: number;
  endMinute: number;
}

const DEFAULT_WINDOW_START = 480;
const DEFAULT_WINDOW_END = 1020;

export function computeWindow(timed: ResolvedEntry[]): TimelineWindow {
  if (timed.length === 0) {
    return { startMinute: DEFAULT_WINDOW_START, endMinute: DEFAULT_WINDOW_END };
  }

  let start = Infinity;
  let end = -Infinity;

  for (const entry of timed) {
    if (entry.startMinute !== undefined && entry.endMinute !== undefined) {
      const s = snapDown15(entry.startMinute);
      const e = snapUp15(entry.endMinute);
      if (s < start) start = s;
      if (e > end) end = e;
    }
  }

  start = Math.min(start, DEFAULT_WINDOW_START);
  end = Math.max(end, DEFAULT_WINDOW_END);

  start = Math.floor(start / 60) * 60;
  end = Math.ceil(end / 60) * 60;

  return { startMinute: start, endMinute: end };
}

export function slotIndex(window: TimelineWindow, minute: number): number {
  return Math.floor((minute - window.startMinute) / SLOT_MINUTES);
}

export function slotCount(window: TimelineWindow): number {
  return Math.floor((window.endMinute - window.startMinute) / SLOT_MINUTES);
}

export function hourLabels(
  window: TimelineWindow
): { minute: number; label: string }[] {
  const labels: { minute: number; label: string }[] = [];
  for (let m = window.startMinute; m < window.endMinute; m += 60) {
    labels.push({ minute: m, label: formatMinute(m) });
  }
  return labels;
}
