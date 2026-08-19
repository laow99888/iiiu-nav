import { useEffect, useMemo, useState } from 'preact/hooks';

import { AuthenticationControls } from '../auth/authentication-controls';
import { BackupDrawer } from '../backups/backup-drawer';
import { CategoryManager } from '../categories/category-manager';
import { BookmarkExportDrawer } from '../imports/bookmark-export';
import { BookmarkImportDrawer } from '../imports/bookmark-import';
import { LinkEditor } from '../links/link-editor';
import { LinkOrderManager } from '../links/link-order-manager';
import { messages } from '../i18n/messages';
import { SearchEngineSettings } from '../search/search-engine-settings';
import { SiteSettingsDrawer } from '../settings/site-settings';
import { ThemeSwitcher } from '../theme/theme-switcher';
import { useThemeMode } from '../theme/use-theme-mode';
import { ListOrdered, Plus } from '../ui/icons/interface-icons';
import { Button, ToastProvider } from '../ui/primitives';
import { SiteIdentity } from '../navigation/site-identity';
import {
  defaultSiteSettings,
  type NavigationCategory,
  type NavigationLink,
  type SiteSettings,
} from '../navigation/types';
import type { SearchEngine } from '../search/search-engines';
import { AdminMobileNavigation, AdminSidebar } from './admin-navigation';
import { AdminSectionContent } from './admin-section-content';
import { adminSectionTitle, type AdminSection } from './admin-section';

type Props = {
  categories: readonly NavigationCategory[];
  onRetry: () => void;
  onSessionChanged: () => void;
  searchEngines?: readonly SearchEngine[];
  site?: SiteSettings;
};

export function AdminPage(props: Props) {
  return (
    <ToastProvider>
      <AdminPageContent {...props} />
    </ToastProvider>
  );
}

function AdminPageContent({
  categories,
  onRetry,
  onSessionChanged,
  searchEngines = [],
  site,
}: Props) {
  const [section, setSection] = useState<AdminSection>('overview');
  const [activeCategoryID, setActiveCategoryID] = useState('');
  const [categoryManagerOpen, setCategoryManagerOpen] = useState(false);
  const [linkEditor, setLinkEditor] = useState<{
    categoryID: string;
    link: NavigationLink | null;
  } | null>(null);
  const [linkOrderOpen, setLinkOrderOpen] = useState(false);
  const [siteSettingsOpen, setSiteSettingsOpen] = useState(false);
  const [searchSettingsOpen, setSearchSettingsOpen] = useState(false);
  const [importOpen, setImportOpen] = useState(false);
  const [exportOpen, setExportOpen] = useState(false);
  const [backupsOpen, setBackupsOpen] = useState(false);
  const { mode, setMode } = useThemeMode();
  const resolvedSite = site ?? defaultSiteSettings;
  const activeCategory =
    categories.find((category) => category.id === activeCategoryID) ?? null;
  const links = useMemo(
    () =>
      activeCategory
        ? activeCategory.links
        : categories.flatMap((item) => item.links),
    [activeCategory, categories],
  );
  const totalLinks = categories.reduce(
    (sum, category) => sum + category.links.length,
    0,
  );
  const privateCategories = categories.filter(
    (category) => category.visibility === 'private',
  ).length;

  useEffect(() => {
    document.title = `${messages.admin.title} · ${resolvedSite.name}`;
  }, [resolvedSite.name]);

  const openNewLink = () => {
    const category = activeCategory ?? categories[0];
    if (category) setLinkEditor({ categoryID: category.id, link: null });
  };
  const editLink = (link: NavigationLink) => {
    const category = categories.find((item) =>
      item.links.some((entry) => entry.id === link.id),
    );
    if (category) setLinkEditor({ categoryID: category.id, link });
  };

  return (
    <div class="admin-shell">
      <AdminSidebar
        section={section}
        site={resolvedSite}
        onChange={setSection}
        onSessionChanged={onSessionChanged}
      />

      <main class="admin-main">
        <header class="admin-mobile-header">
          <SiteIdentity
            name={resolvedSite.name}
            logoUrl={resolvedSite.logoUrl}
            compact
          />
          <div>
            <ThemeSwitcher mode={mode} onChange={setMode} />
            <AuthenticationControls
              administrator
              onSessionChanged={onSessionChanged}
              compact
            />
          </div>
        </header>
        <AdminMobileNavigation section={section} onChange={setSection} />
        <div class="admin-content">
          <header class="admin-heading">
            <div>
              <p>{messages.admin.eyebrow}</p>
              <h1>{adminSectionTitle(section)}</h1>
              <span>
                {messages.admin.summary(totalLinks, categories.length)}
              </span>
            </div>
            <div class="admin-heading__actions">
              {section === 'categories' ? (
                <Button
                  icon={Plus}
                  variant="primary"
                  onClick={() => setCategoryManagerOpen(true)}
                >
                  {messages.categories.add}
                </Button>
              ) : null}
              {section === 'links' || section === 'overview' ? (
                <>
                  <Button
                    icon={Plus}
                    variant="primary"
                    disabled={categories.length === 0}
                    onClick={openNewLink}
                  >
                    {messages.links.create}
                  </Button>
                  <Button
                    icon={ListOrdered}
                    disabled={categories.length === 0}
                    onClick={() => setLinkOrderOpen(true)}
                  >
                    {messages.links.manage}
                  </Button>
                </>
              ) : null}
            </div>
          </header>

          <AdminSectionContent
            activeCategory={activeCategory}
            categories={categories}
            links={links}
            onCategoryChange={setActiveCategoryID}
            onEditLink={editLink}
            onOpenBackups={() => setBackupsOpen(true)}
            onOpenCategories={() => setCategoryManagerOpen(true)}
            onOpenExport={() => setExportOpen(true)}
            onOpenImport={() => setImportOpen(true)}
            onOpenSearchSettings={() => setSearchSettingsOpen(true)}
            onOpenSiteSettings={() => setSiteSettingsOpen(true)}
            privateCategories={privateCategories}
            searchEngines={searchEngines}
            section={section}
            totalLinks={totalLinks}
          />
        </div>
      </main>

      <SearchEngineSettings
        open={searchSettingsOpen}
        engines={searchEngines}
        onClose={() => setSearchSettingsOpen(false)}
        onSaved={onRetry}
      />
      <SiteSettingsDrawer
        open={siteSettingsOpen}
        initial={resolvedSite}
        onClose={() => setSiteSettingsOpen(false)}
        onSaved={() => {
          setSiteSettingsOpen(false);
          onRetry();
        }}
      />
      <BookmarkImportDrawer
        open={importOpen}
        onClose={() => setImportOpen(false)}
        onImported={() => {
          setImportOpen(false);
          onRetry();
        }}
      />
      <BookmarkExportDrawer
        open={exportOpen}
        onClose={() => setExportOpen(false)}
      />
      <BackupDrawer
        open={backupsOpen}
        onClose={() => setBackupsOpen(false)}
        onRestored={() => {
          setBackupsOpen(false);
          onSessionChanged();
        }}
      />
      <CategoryManager
        open={categoryManagerOpen}
        categories={categories}
        onClose={() => setCategoryManagerOpen(false)}
        onSaved={onRetry}
      />
      <LinkEditor
        key={`${linkEditor?.link?.id ?? 'new'}-${linkEditor?.categoryID ?? ''}`}
        open={linkEditor !== null}
        categories={categories}
        categoryID={linkEditor?.categoryID ?? ''}
        link={linkEditor?.link}
        onClose={() => setLinkEditor(null)}
        onSaved={() => {
          setLinkEditor(null);
          onRetry();
        }}
      />
      <LinkOrderManager
        open={linkOrderOpen}
        categories={categories}
        initialCategoryID={activeCategory?.id ?? categories[0]?.id ?? ''}
        onClose={() => setLinkOrderOpen(false)}
        onSaved={() => {
          setLinkOrderOpen(false);
          onRetry();
        }}
      />
    </div>
  );
}
