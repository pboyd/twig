import type { TaskNode } from "./tree";

export function filterTree(nodes: TaskNode[], showCompleted: boolean): TaskNode[] {
  if (showCompleted) return nodes;

  function keep(node: TaskNode): TaskNode | null {
    const filteredChildren = node.children.flatMap((c) => {
      const result = keep(c);
      return result ? [result] : [];
    });

    const hasKeptDescendant = filteredChildren.length > 0;
    const isIncomplete = node.task.completedAt === undefined;

    if (!isIncomplete && !hasKeptDescendant) return null;

    const childrenUnchanged =
      filteredChildren.length === node.children.length &&
      filteredChildren.every((c, i) => c === node.children[i]);

    return childrenUnchanged ? node : { ...node, children: filteredChildren };
  }

  return nodes.flatMap((n) => {
    const result = keep(n);
    return result ? [result] : [];
  });
}
