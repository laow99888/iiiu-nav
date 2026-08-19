import type { CategoryIconName } from '../ui/icons/category-icon-registry';

export type CategoryInput = {
  iconName: CategoryIconName | '';
  name: string;
  visibility: 'public' | 'private';
};

export function createCategory(input: CategoryInput) {
  return categoryRequest('/api/categories', {
    method: 'POST',
    body: JSON.stringify(input),
  });
}

export function updateCategory(id: string, input: CategoryInput) {
  return categoryRequest(`/api/categories/${encodeURIComponent(id)}`, {
    method: 'PUT',
    body: JSON.stringify(input),
  });
}

export function reorderCategories(ids: readonly string[]) {
  return categoryRequest('/api/categories/order', {
    method: 'PUT',
    body: JSON.stringify({ ids }),
  });
}

export function deleteCategory(
  id: string,
  mode: 'delete' | 'move',
  targetID?: string,
) {
  const query = new URLSearchParams({ links: mode });
  if (mode === 'move' && targetID) query.set('target', targetID);
  return categoryRequest(
    `/api/categories/${encodeURIComponent(id)}?${query.toString()}`,
    { method: 'DELETE' },
  );
}

async function categoryRequest(path: string, init: RequestInit) {
  const response = await fetch(path, {
    ...init,
    credentials: 'same-origin',
    headers: init.body
      ? { Accept: 'application/json', 'Content-Type': 'application/json' }
      : { Accept: 'application/json' },
  });
  if (!response.ok)
    throw new Error(`Category request failed: ${response.status}`);
}
