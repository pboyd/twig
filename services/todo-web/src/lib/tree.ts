import type { Task } from "../gen/task/v1/task_pb";

export interface TaskNode {
  task: Task;
  children: TaskNode[];
  depth: number;
}

export function buildTree(tasks: Task[]): TaskNode[] {
  const nodeMap = new Map<bigint, TaskNode>();

  for (const task of tasks) {
    nodeMap.set(task.id, { task, children: [], depth: 0 });
  }

  const roots: TaskNode[] = [];

  for (const task of tasks) {
    const node = nodeMap.get(task.id)!;
    if (task.parentId !== undefined) {
      const parent = nodeMap.get(task.parentId);
      if (parent) {
        node.depth = parent.depth + 1;
        parent.children.push(node);
      } else {
        roots.push(node);
      }
    } else {
      roots.push(node);
    }
  }

  return roots;
}
