import type {
  NavigationCategory,
  NavigationLink,
  SiteSettings,
} from '../navigation/types';
import type { SearchEngine } from '../search/search-engines';
import { AdminCategoryTable } from './admin-category-table';
import { AdminDashboard } from './admin-dashboard';
import { AdminLinkTable } from './admin-link-table';
import type { AdminSection } from './admin-section';
import { AdminDataWorkspace, AdminSettingsWorkspace } from './admin-workspaces';

type Props = {
  categories: readonly NavigationCategory[];
  onCreateCategory: () => void;
  onDeleteCategory: (category: NavigationCategory) => void;
  onDeleteLink: (link: NavigationLink) => void;
  onEditCategory: (category: NavigationCategory) => void;
  onEditLink: (link: NavigationLink) => void;
  onOpenBackups: () => void;
  onOpenCategories: () => void;
  onOpenExport: () => void;
  onOpenImport: () => void;
  onOpenSearchSettings: () => void;
  onOpenSiteSettings: () => void;
  onCreateLink: () => void;
  searchEngines: readonly SearchEngine[];
  section: AdminSection;
  site: SiteSettings;
};

export function AdminSectionContent(props: Props) {
  const {
    categories,
    onCreateCategory,
    onDeleteCategory,
    onDeleteLink,
    onEditCategory,
    onEditLink,
    onOpenBackups,
    onOpenCategories,
    onOpenExport,
    onOpenImport,
    onOpenSearchSettings,
    onOpenSiteSettings,
    onCreateLink,
    searchEngines,
    section,
    site,
  } = props;

  if (section === 'dashboard') {
    return <AdminDashboard categories={categories} />;
  }

  if (section === 'categories') {
    return (
      <AdminCategoryTable
        categories={categories}
        onCreate={onCreateCategory}
        onDelete={onDeleteCategory}
        onEdit={onEditCategory}
        onOpenOrder={onOpenCategories}
      />
    );
  }

  if (section === 'links') {
    return (
      <AdminLinkTable
        categories={categories}
        onCreate={onCreateLink}
        onDelete={onDeleteLink}
        onEdit={onEditLink}
      />
    );
  }

  if (section === 'data') {
    return (
      <AdminDataWorkspace
        onOpenBackups={onOpenBackups}
        onOpenExport={onOpenExport}
        onOpenImport={onOpenImport}
      />
    );
  }

  return (
    <AdminSettingsWorkspace
      engines={searchEngines}
      site={site}
      onOpenSearchSettings={onOpenSearchSettings}
      onOpenSiteSettings={onOpenSiteSettings}
    />
  );
}
