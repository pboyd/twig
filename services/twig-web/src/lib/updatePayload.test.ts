import { describe, it, expect } from "vitest";
import { buildUpdatePayload } from "./updatePayload";
import type { Task } from "../gen/task/v1/task_pb";
import type { Timestamp } from "@bufbuild/protobuf/wkt";

function makeTask(overrides: Partial<Task> = {}): Task {
  return {
    id: 1n,
    name: "Original name",
    description: "Original description",
    due: undefined,
    parentId: undefined,
    completedAt: undefined,
    estimate: 0,
    $typeName: "task.v1.Task",
    ...overrides,
  } as unknown as Task;
}

describe("buildUpdatePayload", () => {
  it("uses edited name and description", () => {
    const task = makeTask({ name: "Old name", description: "Old desc" });
    const payload = buildUpdatePayload(task, "New name", "New desc");
    expect(payload.name).toBe("New name");
    expect(payload.description).toBe("New desc");
  });

  it("preserves the task id", () => {
    const task = makeTask({ id: 42n });
    const payload = buildUpdatePayload(task, "Name", "");
    expect(payload.id).toBe(42n);
  });

  it("preserves due when set and resubmitted unchanged", () => {
    const due: Timestamp = { seconds: 9999n, nanos: 0, $typeName: "google.protobuf.Timestamp" };
    const task = makeTask({ due });
    const payload = buildUpdatePayload(task, "Name", "", due);
    expect(payload.due).toBe(due);
  });

  it("preserves due as undefined when unset", () => {
    const task = makeTask({ due: undefined });
    const payload = buildUpdatePayload(task, "Name", "", undefined);
    expect(payload.due).toBeUndefined();
  });

  it("preserves parentId when set", () => {
    const task = makeTask({ parentId: 7n });
    const payload = buildUpdatePayload(task, "Name", "");
    expect(payload.parentId).toBe(7n);
  });

  it("preserves parentId as undefined for top-level tasks", () => {
    const task = makeTask({ parentId: undefined });
    const payload = buildUpdatePayload(task, "Name", "");
    expect(payload.parentId).toBeUndefined();
  });

  it("preserves snoozeUntil when set and resubmitted unchanged", () => {
    const snoozeUntil: Timestamp = { seconds: 12345n, nanos: 0, $typeName: "google.protobuf.Timestamp" };
    const task = makeTask({ snoozeUntil });
    const payload = buildUpdatePayload(task, "Name", "", undefined, snoozeUntil);
    expect(payload.snoozeUntil).toBe(snoozeUntil);
  });

  it("preserves snoozeUntil as undefined when unset", () => {
    const task = makeTask({ snoozeUntil: undefined });
    const payload = buildUpdatePayload(task, "Name", "", undefined, undefined);
    expect(payload.snoozeUntil).toBeUndefined();
  });

  it("an edited due value overrides task.due", () => {
    const taskDue: Timestamp = { seconds: 1n, nanos: 0, $typeName: "google.protobuf.Timestamp" };
    const editedDue: Timestamp = { seconds: 2n, nanos: 0, $typeName: "google.protobuf.Timestamp" };
    const task = makeTask({ due: taskDue });
    const payload = buildUpdatePayload(task, "Name", "", editedDue);
    expect(payload.due).toBe(editedDue);
  });

  it("an empty due value omits due from the payload", () => {
    const taskDue: Timestamp = { seconds: 1n, nanos: 0, $typeName: "google.protobuf.Timestamp" };
    const task = makeTask({ due: taskDue });
    const payload = buildUpdatePayload(task, "Name", "", undefined);
    expect(payload.due).toBeUndefined();
  });

  it("still carries parentId and snoozeUntil through unchanged when editing due", () => {
    const snoozeUntil: Timestamp = { seconds: 5n, nanos: 0, $typeName: "google.protobuf.Timestamp" };
    const task = makeTask({ parentId: 7n, snoozeUntil });
    const payload = buildUpdatePayload(task, "Name", "", undefined, snoozeUntil);
    expect(payload.parentId).toBe(7n);
    expect(payload.snoozeUntil).toEqual(snoozeUntil);
  });

  it("never includes goalId in the payload", () => {
    const task = makeTask({ id: 1n });
    const payload = buildUpdatePayload(task, "Name", "");
    expect("goalId" in payload).toBe(false);
  });

  it("an edited snooze value overrides task.snoozeUntil", () => {
    const taskSnooze: Timestamp = { seconds: 1n, nanos: 0, $typeName: "google.protobuf.Timestamp" };
    const editedSnooze: Timestamp = { seconds: 2n, nanos: 0, $typeName: "google.protobuf.Timestamp" };
    const task = makeTask({ snoozeUntil: taskSnooze });
    const payload = buildUpdatePayload(task, "Name", "", undefined, editedSnooze);
    expect(payload.snoozeUntil).toBe(editedSnooze);
  });

  it("an empty snooze value omits snoozeUntil (un-snoozing)", () => {
    const taskSnooze: Timestamp = { seconds: 1n, nanos: 0, $typeName: "google.protobuf.Timestamp" };
    const task = makeTask({ snoozeUntil: taskSnooze });
    const payload = buildUpdatePayload(task, "Name", "", undefined, undefined);
    expect(payload.snoozeUntil).toBeUndefined();
  });

  it("applies both an edited due and an edited snooze together", () => {
    const editedDue: Timestamp = { seconds: 10n, nanos: 0, $typeName: "google.protobuf.Timestamp" };
    const editedSnooze: Timestamp = { seconds: 20n, nanos: 0, $typeName: "google.protobuf.Timestamp" };
    const task = makeTask({ parentId: 3n });
    const payload = buildUpdatePayload(task, "Name", "", editedDue, editedSnooze);
    expect(payload.due).toBe(editedDue);
    expect(payload.snoozeUntil).toBe(editedSnooze);
    expect(payload.parentId).toBe(3n);
  });
});
