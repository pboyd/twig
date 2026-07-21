import { GoalState } from "../gen/goal/v1/goal_pb";
import type { Goal } from "../gen/goal/v1/goal_pb";

export interface GoalGroup {
  state: GoalState;
  label: string;
  goals: Goal[];
}

interface GroupDef {
  state: GoalState;
  label: string;
  hidden: boolean;
}

const GROUP_ORDER: GroupDef[] = [
  { state: GoalState.IN_PROGRESS, label: "In Progress", hidden: false },
  { state: GoalState.INCUBATING, label: "Incubating", hidden: false },
  { state: GoalState.COMPLETED, label: "Completed", hidden: true },
  { state: GoalState.ARCHIVED, label: "Archived", hidden: true },
];

export function goalGroups(goals: Goal[], showHidden: boolean): GoalGroup[] {
  const groups = new Map<GoalState, Goal[]>();

  for (const goal of goals) {
    const key = goal.state;
    if (!groups.has(key)) {
      groups.set(key, []);
    }
    groups.get(key)!.push(goal);
  }

  const result: GoalGroup[] = [];

  for (const def of GROUP_ORDER) {
    const gs = groups.get(def.state);
    if (!gs || gs.length === 0) continue;

    if (def.hidden && !showHidden) continue;

    result.push({ state: def.state, label: def.label, goals: gs });
  }

  return result;
}
