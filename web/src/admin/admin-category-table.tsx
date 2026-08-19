import { useMemo, useState } from 'preact/hooks';

import { messages } from '../i18n/messages';
import { CategoryIcon } from '../ui/icons/category-icons';
import {
  ListOrdered,
  Lock,
  Pencil,
  Search,
  Settings,
  Trash2,
} from '../ui/icons/interface-icons';
import { Button, EmptyState } from '../ui/primitives';
import type { NavigationCategory } from '../navigation/types';

type CategoryRow = {
  category: NavigationCategory;
  order: number;
};

type Props = {
  categories: readonly NavigationCategory[];
  onCreate: () => void;
  onDelete: (category: NavigationCategory) => void;
  onEdit: (category: NavigationCategory) => void;
  onOpenOrder: () => void;
};

export function AdminCategoryTable({
  categories,
  onCreate,
  onDelete,
  onEdit,
  onOpenOrder,
}: Props) {
  const [query, setQuery] = useState('');
  const rows = useMemo(
    () => categories.map((category, index) => ({ category, order: index + 1 })),
    [categories],
  );
  const visibleRows = useMemo(() => filterRows(rows, query), [query, rows]);

  return (
    <section class="admin-table-panel admin-category-table">
      <div class="admin-panel__header">
        <div>
          <h2>{messages.admin.categoryPanelTitle}</h2>
          <p>{messages.categories.manageDescription}</p>
        </div>
        <Button
          icon={ListOrdered}
          onClick={onOpenOrder}
          disabled={!categories.length}
        >
          {messages.categories.adjustOrder}
        </Button>
      </div>
      {categories.length > 0 ? (
        <div class="admin-table-toolbar">
          <label class="admin-table-search">
            <span class="sr-only">{messages.admin.categorySearch}</span>
            <Search aria-hidden="true" />
            <input
              type="search"
              value={query}
              aria-label={messages.admin.categorySearch}
              placeholder={messages.admin.categorySearchPlaceholder}
              onInput={(event) => setQuery(event.currentTarget.value)}
            />
          </label>
          <span class="admin-table-toolbar__count">
            {messages.admin.categoryResults(visibleRows.length)}
          </span>
        </div>
      ) : null}

      {visibleRows.length > 0 ? (
        <div class="admin-table-scroll">
          <table aria-label={messages.admin.categoryTableLabel}>
            <thead>
              <tr>
                <th>{messages.admin.categoryOrder}</th>
                <th>{messages.admin.categoryIcon}</th>
                <th>{messages.categories.name}</th>
                <th>{messages.admin.categoryLinks}</th>
                <th>{messages.admin.categoryVisibility}</th>
                <th>{messages.admin.categoryActions}</th>
              </tr>
            </thead>
            <tbody>
              {visibleRows.map(({ category, order }) => (
                <tr key={category.id}>
                  <td>
                    <span class="admin-category-order">
                      {String(order).padStart(2, '0')}
                    </span>
                  </td>
                  <td>
                    <span
                      class="admin-category-icon"
                      title={
                        category.icon ? undefined : messages.categories.noIcon
                      }
                      aria-label={
                        category.icon ? undefined : messages.categories.noIcon
                      }
                    >
                      <CategoryIcon name={category.icon} />
                    </span>
                  </td>
                  <td>
                    <strong class="admin-category-name">{category.name}</strong>
                  </td>
                  <td>
                    <span class="admin-category-link-count">
                      {category.links.length} {messages.categories.links}
                    </span>
                  </td>
                  <td>
                    <span
                      class={`admin-visibility admin-visibility--${category.visibility}`}
                    >
                      {category.visibility === 'private' ? (
                        <Lock aria-hidden="true" />
                      ) : null}
                      {category.visibility === 'private'
                        ? messages.admin.privateVisibility
                        : messages.admin.publicVisibility}
                    </span>
                  </td>
                  <td>
                    <div class="admin-row-actions">
                      <Button
                        size="small"
                        variant="ghost"
                        icon={Pencil}
                        aria-label={messages.categories.editNamed(
                          category.name,
                        )}
                        title={messages.categories.editNamed(category.name)}
                        onClick={() => onEdit(category)}
                      />
                      <Button
                        size="small"
                        variant="ghost"
                        icon={Trash2}
                        aria-label={messages.categories.deleteNamed(
                          category.name,
                        )}
                        title={messages.categories.deleteNamed(category.name)}
                        onClick={() => onDelete(category)}
                      />
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : (
        <EmptyState
          icon={Settings}
          title={
            categories.length > 0
              ? messages.admin.noFilteredCategoriesTitle
              : messages.admin.noCategoriesTitle
          }
          description={
            categories.length > 0
              ? messages.admin.noFilteredCategoriesDescription
              : messages.admin.noCategoriesDescription
          }
          action={
            categories.length === 0 ? (
              <Button icon={Pencil} variant="primary" onClick={onCreate}>
                {messages.admin.createFirstCategory}
              </Button>
            ) : undefined
          }
        />
      )}
    </section>
  );
}

function filterRows(rows: readonly CategoryRow[], query: string) {
  const normalizedQuery = query.trim().toLocaleLowerCase();
  if (!normalizedQuery) return rows;
  return rows.filter(({ category }) =>
    category.name.toLocaleLowerCase().includes(normalizedQuery),
  );
}
