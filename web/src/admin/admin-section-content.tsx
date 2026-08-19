import { messages } from '../i18n/messages';
import { LinkGrid } from '../navigation/link-grid';
import type { NavigationCategory, NavigationLink } from '../navigation/types';
import { CombinedSearch } from '../search/combined-search';
import type { SearchEngine } from '../search/search-engines';
import {
  Archive,
  FileDown,
  FileUp,
  Settings,
  SlidersHorizontal,
} from '../ui/icons/interface-icons';
import { Button } from '../ui/primitives';
import type { AdminSection } from './admin-section';

type Props = {
  activeCategory: NavigationCategory | null;
  categories: readonly NavigationCategory[];
  links: readonly NavigationLink[];
  onCategoryChange: (id: string) => void;
  onEditLink: (link: NavigationLink) => void;
  onOpenBackups: () => void;
  onOpenCategories: () => void;
  onOpenExport: () => void;
  onOpenImport: () => void;
  onOpenSearchSettings: () => void;
  onOpenSiteSettings: () => void;
  privateCategories: number;
  searchEngines: readonly SearchEngine[];
  section: AdminSection;
  totalLinks: number;
};

export function AdminSectionContent(props: Props) {
  const {
    activeCategory,
    categories,
    links,
    onCategoryChange,
    onEditLink,
    onOpenBackups,
    onOpenCategories,
    onOpenExport,
    onOpenImport,
    onOpenSearchSettings,
    onOpenSiteSettings,
    privateCategories,
    searchEngines,
    section,
    totalLinks,
  } = props;

  if (section === 'overview') {
    return (
      <>
        <section class="admin-stat-grid" aria-label={messages.admin.statistics}>
          <AdminStat label={messages.admin.totalLinks} value={totalLinks} />
          <AdminStat
            label={messages.admin.totalCategories}
            value={categories.length}
          />
          <AdminStat
            label={messages.admin.privateCategories}
            value={privateCategories}
          />
        </section>
        <AdminNavigationContent
          categories={categories}
          activeCategory={activeCategory}
          links={links}
          onCategoryChange={onCategoryChange}
          onEdit={onEditLink}
          searchEngines={searchEngines}
        />
      </>
    );
  }

  if (section === 'categories') {
    return (
      <section class="admin-panel">
        <div class="admin-panel__header">
          <div>
            <h2>{messages.admin.categoryPanelTitle}</h2>
            <p>{messages.categories.manageDescription}</p>
          </div>
          <Button icon={Settings} onClick={onOpenCategories}>
            {messages.categories.manage}
          </Button>
        </div>
        <div class="admin-category-list">
          {categories.map((category) => (
            <button
              key={category.id}
              class="admin-category-row"
              onClick={() => {
                onCategoryChange(category.id);
                onOpenCategories();
              }}
            >
              <strong>{category.name}</strong>
              <span>
                {category.links.length} {messages.categories.links}
              </span>
            </button>
          ))}
        </div>
      </section>
    );
  }

  if (section === 'links') {
    return (
      <AdminNavigationContent
        categories={categories}
        activeCategory={activeCategory}
        links={links}
        onCategoryChange={onCategoryChange}
        onEdit={onEditLink}
        searchEngines={searchEngines}
      />
    );
  }

  if (section === 'data') {
    return (
      <section class="admin-tool-grid">
        <AdminTool
          icon={FileUp}
          title={messages.imports.menu}
          description={messages.imports.description}
          action={messages.imports.menu}
          onClick={onOpenImport}
        />
        <AdminTool
          icon={FileDown}
          title={messages.exports.menu}
          description={messages.exports.description}
          action={messages.exports.menu}
          onClick={onOpenExport}
        />
        <AdminTool
          icon={Archive}
          title={messages.backups.menu}
          description={messages.backups.description}
          action={messages.backups.menu}
          onClick={onOpenBackups}
        />
      </section>
    );
  }

  return (
    <section class="admin-tool-grid">
      <AdminTool
        icon={SlidersHorizontal}
        title={messages.settings.title}
        description={messages.settings.savedDescription}
        action={messages.settings.menu}
        onClick={onOpenSiteSettings}
      />
      <AdminTool
        icon={Settings}
        title={messages.search.settingsTitle}
        description={messages.search.settingsDescription}
        action={messages.search.configure}
        onClick={onOpenSearchSettings}
      />
    </section>
  );
}

function AdminNavigationContent({
  categories,
  activeCategory,
  links,
  onCategoryChange,
  onEdit,
  searchEngines,
}: {
  categories: readonly NavigationCategory[];
  activeCategory: NavigationCategory | null;
  links: readonly NavigationLink[];
  onCategoryChange: (id: string) => void;
  onEdit: (link: NavigationLink) => void;
  searchEngines: readonly SearchEngine[];
}) {
  return (
    <>
      <div class="admin-filter-row">
        <label>
          <span>{messages.admin.categoryFilter}</span>
          <select
            value={activeCategory?.id ?? ''}
            onChange={(event) => onCategoryChange(event.currentTarget.value)}
          >
            <option value="">{messages.navigation.allCategories}</option>
            {categories.map((category) => (
              <option key={category.id} value={category.id}>
                {category.name}
              </option>
            ))}
          </select>
        </label>
        <CombinedSearch categories={categories} engines={searchEngines} />
      </div>
      <LinkGrid category={activeCategory} links={links} onEdit={onEdit} />
    </>
  );
}

function AdminStat({ label, value }: { label: string; value: number }) {
  return (
    <div class="admin-stat">
      <span>{label}</span>
      <strong>{value}</strong>
    </div>
  );
}

function AdminTool({
  action,
  description,
  icon: Icon,
  onClick,
  title,
}: {
  action: string;
  description: string;
  icon: typeof Archive;
  onClick: () => void;
  title: string;
}) {
  return (
    <article class="admin-tool">
      <Icon aria-hidden="true" />
      <h2>{title}</h2>
      <p>{description}</p>
      <Button onClick={onClick}>{action}</Button>
    </article>
  );
}
