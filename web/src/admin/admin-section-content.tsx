import { messages } from '../i18n/messages';
import type { NavigationCategory, NavigationLink } from '../navigation/types';
import {
  Archive,
  FileDown,
  FileUp,
  Settings,
  SlidersHorizontal,
} from '../ui/icons/interface-icons';
import { Button } from '../ui/primitives';
import { AdminDashboard } from './admin-dashboard';
import { AdminLinkTable } from './admin-link-table';
import type { AdminSection } from './admin-section';

type Props = {
  categories: readonly NavigationCategory[];
  onCategoryChange: (id: string) => void;
  onDeleteLink: (link: NavigationLink) => void;
  onEditLink: (link: NavigationLink) => void;
  onOpenBackups: () => void;
  onOpenCategories: () => void;
  onOpenExport: () => void;
  onOpenImport: () => void;
  onOpenSearchSettings: () => void;
  onOpenSiteSettings: () => void;
  section: AdminSection;
};

export function AdminSectionContent(props: Props) {
  const {
    categories,
    onCategoryChange,
    onDeleteLink,
    onEditLink,
    onOpenBackups,
    onOpenCategories,
    onOpenExport,
    onOpenImport,
    onOpenSearchSettings,
    onOpenSiteSettings,
    section,
  } = props;

  if (section === 'dashboard') {
    return <AdminDashboard categories={categories} />;
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
      <AdminLinkTable
        categories={categories}
        onDelete={onDeleteLink}
        onEdit={onEditLink}
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
