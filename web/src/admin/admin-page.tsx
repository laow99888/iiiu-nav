import { useEffect, useState } from 'preact/hooks';

import { AuthenticationControls } from '../auth/authentication-controls';
import { BackupDrawer } from '../backups/backup-drawer';
import type { CategoryManagerAction } from '../categories/category-manager';
import { BookmarkExportDrawer } from '../imports/bookmark-export';
import { BookmarkImportDrawer } from '../imports/bookmark-import';
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
import {
  AdminContentDialogs,
  type CategoryManagerState,
  type LinkEditorState,
} from './admin-content-dialogs';
import { AdminSectionContent } from './admin-section-content';
import {
  adminSectionFromPath,
  adminSectionPath,
  adminSectionTitle,
  type AdminSection,
} from './admin-section';

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
  // 模块选择写入 URL（/admin/links 等），刷新与直达保持在原模块。
  const [section, setSectionState] = useState<AdminSection>(
    () => adminSectionFromPath(window.location.pathname) ?? 'dashboard',
  );
  const setSection = (next: AdminSection) => {
    setSectionState((current) => {
      if (current !== next) {
        // 保留当前历史状态对象，避免抹掉弹层的 {uiModal} 返回哨兵。
        window.history.replaceState(
          window.history.state,
          '',
          adminSectionPath(next),
        );
      }
      return next;
    });
  };
  const [categoryManager, setCategoryManager] =
    useState<CategoryManagerState>(null);
  const [linkEditor, setLinkEditor] = useState<LinkEditorState>(null);
  const [deleteLink, setDeleteLink] = useState<NavigationLink | null>(null);
  const [linkOrderOpen, setLinkOrderOpen] = useState(false);
  const [siteSettingsOpen, setSiteSettingsOpen] = useState(false);
  const [searchSettingsOpen, setSearchSettingsOpen] = useState(false);
  const [importOpen, setImportOpen] = useState(false);
  const [exportOpen, setExportOpen] = useState(false);
  const [backupsOpen, setBackupsOpen] = useState(false);
  const { mode, setMode } = useThemeMode();
  const resolvedSite = site ?? defaultSiteSettings;
  const totalLinks = categories.reduce(
    (sum, category) => sum + category.links.length,
    0,
  );

  useEffect(() => {
    document.title = `${messages.admin.title} · ${resolvedSite.name}`;
  }, [resolvedSite.name]);

  const openNewLink = () => {
    const category = categories[0];
    if (category) setLinkEditor({ categoryID: category.id, link: null });
  };
  const openCategoryManager = (action: CategoryManagerAction | null = null) => {
    setCategoryManager(action ?? { type: 'list' });
  };
  const closeCategoryManager = () => {
    setCategoryManager(null);
  };
  const editLink = (link: NavigationLink) => {
    const category = categories.find((item) =>
      item.links.some((entry) => entry.id === link.id),
    );
    if (category) setLinkEditor({ categoryID: category.id, link });
  };

  return (
    <div class="admin-shell">
      <a class="skip-link" href="#main-content">
        {messages.ui.skipToContent}
      </a>
      <AdminSidebar
        section={section}
        site={resolvedSite}
        onChange={setSection}
        onSessionChanged={onSessionChanged}
      />

      <main class="admin-main" id="main-content" tabIndex={-1}>
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
                  onClick={() => openCategoryManager({ type: 'create' })}
                >
                  {messages.categories.add}
                </Button>
              ) : null}
              {section === 'links' ? (
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
                    {messages.links.reorder}
                  </Button>
                </>
              ) : null}
            </div>
          </header>

          <AdminSectionContent
            categories={categories}
            onCreateCategory={() => openCategoryManager({ type: 'create' })}
            onDeleteCategory={(category) =>
              openCategoryManager({ type: 'delete', categoryID: category.id })
            }
            onDeleteLink={setDeleteLink}
            onEditCategory={(category) =>
              openCategoryManager({ type: 'edit', categoryID: category.id })
            }
            onEditLink={editLink}
            onOpenBackups={() => setBackupsOpen(true)}
            onOpenCategories={() => openCategoryManager()}
            onOpenExport={() => setExportOpen(true)}
            onOpenImport={() => setImportOpen(true)}
            onOpenSearchSettings={() => setSearchSettingsOpen(true)}
            onOpenSiteSettings={() => setSiteSettingsOpen(true)}
            onCreateLink={openNewLink}
            searchEngines={searchEngines}
            section={section}
            site={resolvedSite}
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
      <AdminContentDialogs
        categories={categories}
        categoryManager={categoryManager}
        deleteLink={deleteLink}
        linkEditor={linkEditor}
        linkOrderOpen={linkOrderOpen}
        onCloseCategoryManager={closeCategoryManager}
        onCloseDeleteLink={() => setDeleteLink(null)}
        onCloseLinkEditor={() => setLinkEditor(null)}
        onCloseLinkOrder={() => setLinkOrderOpen(false)}
        onRefresh={onRetry}
      />
    </div>
  );
}
