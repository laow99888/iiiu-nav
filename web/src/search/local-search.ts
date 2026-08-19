import type { NavigationCategory, NavigationLink } from '../navigation/types';

export type LocalSearchResult = {
  categoryID: string;
  categoryName: string;
  link: NavigationLink;
};

export function searchLocalLinks(
  categories: readonly NavigationCategory[],
  query: string,
  limit = 6,
) {
  const normalized = query.trim().toLocaleLowerCase('zh-CN');
  if (!normalized) {
    return [];
  }

  const results: LocalSearchResult[] = [];
  for (const category of categories) {
    for (const link of category.links) {
      const searchable =
        `${link.name}\n${link.description}\n${link.url}`.toLocaleLowerCase(
          'zh-CN',
        );
      if (searchable.includes(normalized)) {
        results.push({
          categoryID: category.id,
          categoryName: category.name,
          link,
        });
        if (results.length === limit) {
          return results;
        }
      }
    }
  }
  return results;
}
