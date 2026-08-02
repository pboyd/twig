import type { Timestamp } from "@bufbuild/protobuf/wkt";

// Due and snooze dates are pinned to midnight UTC of the chosen calendar
// day, independent of the client's local timezone (research R3). This keeps
// the ISO day string ("YYYY-MM-DD") <-> Timestamp conversion lossless in
// both directions.

export function isoDayToTimestamp(day: string | undefined): Timestamp | undefined {
  if (!day) return undefined;
  const [year, month, date] = day.split("-").map(Number);
  return {
    seconds: BigInt(Date.UTC(year, month - 1, date) / 1000),
    nanos: 0,
    $typeName: "google.protobuf.Timestamp",
  };
}

export function timestampToIsoDay(ts: Timestamp | undefined): string {
  if (!ts) return "";
  const d = new Date(Number(ts.seconds) * 1000);
  const year = d.getUTCFullYear();
  const month = String(d.getUTCMonth() + 1).padStart(2, "0");
  const date = String(d.getUTCDate()).padStart(2, "0");
  return `${year}-${month}-${date}`;
}

// The edit form is day-granular, but a timestamp set elsewhere (e.g. the CLI's
// RFC 3339 --due) can carry a time-of-day. If the user didn't change the
// calendar day, keep the original timestamp byte-for-byte instead of
// truncating it to midnight UTC.
export function resolveDayEdit(
  original: Timestamp | undefined,
  editedDay: string | undefined
): Timestamp | undefined {
  if (timestampToIsoDay(original) === (editedDay ?? "")) return original;
  return isoDayToTimestamp(editedDay);
}
