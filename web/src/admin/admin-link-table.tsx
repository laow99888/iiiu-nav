import { useMemo, useState } from 'preact/hooks';

import { messages } from '../i18n/messages';
import { LinkLogo } from '../navigation/link-logo';
import type { NavigationCategory, NavigationLink } from '../navigation/types';
import {
  ListOrdered,
  Pencil,
  Plus,
  Search,
  Trash2,
} from '../ui/icons/interface-icons';
import { Button, EmptyState } from '../ui/primitives';

type LinkRow = {
  category: NavigationCategory;
  link: NavigationLink;
  order: number;
};

type Props = {
  categories: readonly NavigationCategory[];
  onCreate: () => void;
  onDelete: (link: NavigationLink) => void;
  onEdit: (link: NavigationLink) => void;
};

export function AdminLinkTable({
  categories,
  onCreate,
  onDelete,
  onEdit,
}: Props) {
  const [categoryID, setCategoryID] = useState('');
  const [query, setQuery] = useState('');
  const rows = useMemo(() => buildRows(categories), [categories]);
  const visibleRows = useMemo(
    () => filterRows(rows, categoryID, query),
    [categoryID, query, rows],
  );

  return (
    <section class="admin-table-panel">
      <div class="admin-table-toolbar">
        <label class="admin-table-search">
          <span class="sr-only">{messages.admin.linkSearch}</span>
          <Search aria-hidden="true" />
          <input
            type="search"
            value={query}
            aria-label={messages.admin.linkSearch}
            placeholder={messages.admin.linkSearchPlaceholder}
            onInput={(event) => setQuery(event.currentTarget.value)}
          />
        </label>
        <label class="admin-table-filter">
          <span>{messages.admin.linkCategoryFilter}</span>
          <select
            value={categoryID}
            onChange={(event) => setCategoryID(event.currentTarget.value)}
          >
            <option value="">{messages.navigation.allCategories}</option>
            {categories.map((category) => (
              <option key={category.id} value={category.id}>
                {category.name}
              </option>
            ))}
          </select>
        </label>
        <span class="admin-table-toolbar__count">
          {messages.admin.linkResults(visibleRows.length)}
        </span>
      </div>

      {visibleRows.length > 0 ? (
        <div class="admin-table-scroll">
          <table aria-label={messages.admin.linkTableLabel}>
            <thead>
              <tr>
                <th>{messages.admin.tableOrder}</th>
                <th>{messages.admin.tableURL}</th>
                <th>{messages.admin.tableName}</th>
                <th>{messages.admin.tableDescription}</th>
                <th>{messages.admin.tableCategory}</th>
                <th>{messages.admin.tablePrivacy}</th>
                <th>{messages.admin.tableActions}</th>
              </tr>
            </thead>
            <tbody>
              {visibleRows.map(({ category, link, order }) => (
                <tr key={link.id}>
                  <td>
                    <span
                      class="admin-link-order"
                      aria-label={`当前列表第 ${order} 位`}
                    >
                      {String(order).padStart(2, '0')}
                    </span>
                  </td>
                  <td>
                    <a
                      class="admin-link-url"
                      href={link.url}
                      target="_blank"
                      rel="noopener noreferrer"
                      title={link.url}
                    >
                      {link.url}
                    </a>
                  </td>
                  <td>
                    <div class="admin-link-name">
                      <LinkLogo
                        label={link.name}
                        text={link.logoText}
                        tone={link.logoTone}
                        url={link.logoURL}
                      />
                      <strong>{link.name}</strong>
                    </div>
                  </td>
                  <td>
                    <span
                      class="admin-link-description"
                      title={link.description || undefined}
                    >
                      {link.description || '—'}
                    </span>
                  </td>
                  <td>
                    <span class="admin-category-badge">{category.name}</span>
                  </td>
                  <td>
                    <span
                      class={`admin-visibility admin-visibility--${category.visibility}`}
                    >
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
                        aria-label={messages.admin.editLink(link.name)}
                        title={messages.admin.editLink(link.name)}
                        onClick={() => onEdit(link)}
                      />
                      <Button
                        size="small"
                        variant="ghost"
                        icon={Trash2}
                        aria-label={messages.admin.deleteLink(link.name)}
                        title={messages.admin.deleteLink(link.name)}
                        onClick={() => onDelete(link)}
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
          icon={ListOrdered}
          title={
            rows.length > 0
              ? messages.admin.noFilteredLinksTitle
              : messages.admin.noLinksTitle
          }
          description={
            rows.length > 0
              ? messages.admin.noFilteredLinksDescription
              : messages.admin.noLinksDescription
          }
          action={
            rows.length === 0 && categories.length > 0 ? (
              <Button icon={Plus} variant="primary" onClick={onCreate}>
                {messages.admin.createFirstLink}
              </Button>
            ) : undefined
          }
        />
      )}
    </section>
  );
}

function buildRows(categories: readonly NavigationCategory[]): LinkRow[] {
  let order = 0;
  return categories.flatMap((category) =>
    category.links.map((link) => ({
      category,
      link,
      order: (order += 1),
    })),
  );
}

function filterRows(
  rows: readonly LinkRow[],
  categoryID: string,
  query: string,
) {
  const normalizedQuery = query.trim().toLocaleLowerCase();
  return rows.filter(({ category, link }) => {
    if (categoryID && category.id !== categoryID) return false;
    if (!normalizedQuery) return true;
    return [link.name, link.url, link.description, category.name].some(
      (value) => value.toLocaleLowerCase().includes(normalizedQuery),
    );
  });
}
