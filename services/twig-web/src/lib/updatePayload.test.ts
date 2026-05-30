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

  it("preserves due when set", () => {
    const due: Timestamp = { seconds: 9999n, nanos: 0, $typeName: "google.protobuf.Timestamp" };
    const task = makeTask({ due });
    const payload = buildUpdatePayload(task, "Name", "");
    expect(payload.due).toBe(due);
  });

  it("preserves due as undefined when unset", () => {
    const task = makeTask({ due: undefined });
    const payload = buildUpdatePayload(task, "Name", "");
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
});
