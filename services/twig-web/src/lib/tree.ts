import type { Task } from "../gen/task/v1/task_pb";

export interface TaskNode {
  task: Task;
  children: TaskNode[];
  depth: number;
}

function sortSiblings(nodes: TaskNode[]): void {
  nodes.sort((a, b) => {
    const pd = Number(a.task.position) - Number(b.task.position);
    if (pd !== 0) return pd;
    return Number(a.task.id) - Number(b.task.id);
  });
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

  sortSiblings(roots);
  for (const node of nodeMap.values()) {
    if (node.children.length > 0) {
      sortSiblings(node.children);
    }
  }

  return roots;
}

export function findSiblingIds(tree: TaskNode[], taskId: bigint): bigint[] | null {
  if (tree.some((n) => n.task.id === taskId)) {
    return tree.map((n) => n.task.id);
  }
  for (const node of tree) {
    const result = findSiblingIds(node.children, taskId);
    if (result !== null) return result;
  }
  return null;
}
