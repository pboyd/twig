import type { Timestamp } from "@bufbuild/protobuf/wkt";

function toDate(ts: Timestamp): Date {
  return new Date(Number(ts.seconds) * 1000 + ts.nanos / 1e6);
}

export function formatTimestamp(ts: Timestamp | undefined): string {
  if (!ts) return "";
  const d = toDate(ts);
  return new Intl.DateTimeFormat(undefined, {
    year: "numeric",
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  }).format(d);
}

export function formatDueDate(ts: Timestamp | undefined): string {
  if (!ts) return "";
  const d = toDate(ts);
  return new Intl.DateTimeFormat(undefined, {
    year: "numeric",
    month: "short",
    day: "numeric",
  }).format(d);
}
