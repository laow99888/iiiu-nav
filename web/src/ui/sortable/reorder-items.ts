import { reorder } from '@atlaskit/pragmatic-drag-and-drop/utils/reorder';

export function reorderItems<T>(
  items: readonly T[],
  startIndex: number,
  finishIndex: number,
) {
  return reorder({ list: [...items], startIndex, finishIndex });
}
