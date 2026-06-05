export type ReorderAnchor =
  | { value: bigint; case: "beforeTaskId" }
  | { value: bigint; case: "afterTaskId" };

export function reorderAnchor(
  activeId: bigint,
  overId: bigint,
  siblings: bigint[],
): ReorderAnchor | null {
  if (activeId === overId) return null;
  const activeIndex = siblings.indexOf(activeId);
  const overIndex = siblings.indexOf(overId);
  if (activeIndex === -1 || overIndex === -1) return null;
  return activeIndex > overIndex
    ? { case: "beforeTaskId", value: overId }
    : { case: "afterTaskId", value: overId };
}
