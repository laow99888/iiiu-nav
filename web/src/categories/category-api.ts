import { ApiError, request } from '../api/client';
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
  const response = await request(path, init);
  if (!response.ok) {
    throw new ApiError(`Category request failed: ${response.status}`, {
      status: response.status,
    });
  }
}
