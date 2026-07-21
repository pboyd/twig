import { describe, it, expect } from "vitest";
import { GoalState } from "../gen/goal/v1/goal_pb";
import { goalGroups } from "./goalGroups";

function goal(id: bigint, state: GoalState, position: bigint = 0n) {
  return { id, name: `Goal ${id}`, description: "", state, position, $typeName: "goal.v1.Goal" as const };
}

describe("goalGroups", () => {
  it("groups goals by state in order: In Progress, Incubating", () => {
    const goals = [
      goal(1n, GoalState.INCUBATING),
      goal(2n, GoalState.IN_PROGRESS),
    ];
    const groups = goalGroups(goals, false);
    expect(groups).toHaveLength(2);
    expect(groups[0].label).toBe("In Progress");
    expect(groups[1].label).toBe("Incubating");
  });

  it("hides Completed and Archived when showHidden is false", () => {
    const goals = [
      goal(1n, GoalState.IN_PROGRESS),
      goal(2n, GoalState.COMPLETED),
      goal(3n, GoalState.ARCHIVED),
    ];
    const groups = goalGroups(goals, false);
    expect(groups).toHaveLength(1);
    expect(groups[0].label).toBe("In Progress");
  });

  it("shows Completed and Archived when showHidden is true", () => {
    const goals = [
      goal(1n, GoalState.IN_PROGRESS),
      goal(2n, GoalState.COMPLETED),
      goal(3n, GoalState.ARCHIVED),
    ];
    const groups = goalGroups(goals, true);
    expect(groups).toHaveLength(3);
    expect(groups[0].label).toBe("In Progress");
    expect(groups[1].label).toBe("Completed");
    expect(groups[2].label).toBe("Archived");
  });

  it("omits empty groups", () => {
    const goals = [goal(1n, GoalState.IN_PROGRESS)];
    const groups = goalGroups(goals, true);
    expect(groups).toHaveLength(1);
    expect(groups[0].label).toBe("In Progress");
  });

  it("returns empty array for no goals", () => {
    expect(goalGroups([], false)).toEqual([]);
  });
});
