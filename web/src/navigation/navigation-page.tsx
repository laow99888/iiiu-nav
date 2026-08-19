import type { JSX } from 'preact';
import { useEffect, useMemo, useState } from 'preact/hooks';

import { AuthenticationControls } from '../auth/authentication-controls';
import { BackupDrawer } from '../backups/backup-drawer';
import { CategoryManager } from '../categories/category-manager';
import { messages } from '../i18n/messages';
import { BookmarkImportDrawer } from '../imports/bookmark-import';
import { BookmarkExportDrawer } from '../imports/bookmark-export';
import { LinkEditor } from '../links/link-editor';
import { LinkOrderManager } from '../links/link-order-manager';
import { CombinedSearch } from '../search/combined-search';
import { SearchEngineSettings } from '../search/search-engine-settings';
import { SiteSettingsDrawer } from '../settings/site-settings';
import {
  defaultSearchEngines,
  type SearchEngine,
} from '../search/search-engines';
import { ThemeSwitcher } from '../theme/theme-switcher';
import { useThemeMode } from '../theme/use-theme-mode';
import { ListOrdered, Plus, Settings } from '../ui/icons/interface-icons';
import { Button, ToastProvider } from '../ui/primitives';
import { CategoryNavigation } from './category-navigation';
import { LinkGrid } from './link-grid';
import { SiteIdentity } from './site-identity';
import {
  defaultSiteSettings,
  type NavigationCategory,
  type NavigationLink,
  type SiteSettings,
} from './types';

type NavigationPageProps = {
  administrator?: boolean;
  categories: readonly NavigationCategory[];
  onRetry?: () => void;
  onSessionChanged?: () => void;
  searchEngines?: readonly SearchEngine[];
  site?: SiteSettings;
  siteName?: string;
  status?: 'ready' | 'loading' | 'error';
};

export function NavigationPage(props: NavigationPageProps) {
  return (
    <ToastProvider>
      <NavigationPageContent {...props} />
    </ToastProvider>
  );
}

function NavigationPageContent({
  administrator = false,
  categories,
  onRetry,
  onSessionChanged = () => undefined,
  searchEngines = defaultSearchEngines,
  site,
  siteName = 'iiiu-nav',
  status = 'ready',
}: NavigationPageProps) {
  const [activeId, setActiveId] = useState('all');
  const [searchSettingsOpen, setSearchSettingsOpen] = useState(false);
  const [siteSettingsOpen, setSiteSettingsOpen] = useState(false);
  const [bookmarkImportOpen, setBookmarkImportOpen] = useState(false);
  const [bookmarkExportOpen, setBookmarkExportOpen] = useState(false);
  const [backupsOpen, setBackupsOpen] = useState(false);
  const [categoryManagerOpen, setCategoryManagerOpen] = useState(false);
  const [linkEditor, setLinkEditor] = useState<{
    categoryID: string;
    link: NavigationLink | null;
  } | null>(null);
  const [linkOrderOpen, setLinkOrderOpen] = useState(false);
  const { mode, setMode } = useThemeMode();
  const resolvedSite = site ?? { ...defaultSiteSettings, name: siteName };
  useEffect(() => {
    document.title = resolvedSite.name;
    let icon = document.querySelector<HTMLLinkElement>('link[rel="icon"]');
    if (!icon) {
      icon = document.createElement('link');
      icon.rel = 'icon';
      document.head.append(icon);
    }
    icon.href = resolvedSite.faviconUrl || '/favicon.ico';
  }, [resolvedSite.faviconUrl, resolvedSite.name]);
  const activeCategory =
    categories.find((category) => category.id === activeId) ?? null;
  const links = useMemo(
    () =>
      activeCategory
        ? activeCategory.links
        : categories.flatMap((category) => category.links),
    [activeCategory, categories],
  );

  const title = activeCategory?.name ?? messages.navigation.allCategories;

  return (
    <div
      class={`navigation-shell${resolvedSite.backgroundUrl ? ' has-background' : ''}`}
      style={
        {
          '--site-accent': resolvedSite.accentColor,
          '--site-background': resolvedSite.backgroundUrl
            ? `url("${resolvedSite.backgroundUrl}")`
            : 'none',
          '--site-overlay': String(resolvedSite.backgroundOverlay / 100),
        } as JSX.CSSProperties
      }
    >
      <aside class="navigation-sidebar">
        <SiteIdentity name={resolvedSite.name} logoUrl={resolvedSite.logoUrl} />
        <CategoryNavigation
          variant="desktop"
          categories={categories}
          activeId={activeId}
          onSelect={setActiveId}
        />
        <p class="navigation-sidebar__summary">
          {messages.navigation.totalSummary(categories.length, links.length)}
        </p>
        {administrator ? (
          <p class="navigation-sidebar__mode">
            {messages.navigation.adminView}
          </p>
        ) : null}
        <AuthenticationControls
          administrator={administrator}
          onSessionChanged={onSessionChanged}
          onOpenSiteSettings={() => setSiteSettingsOpen(true)}
          onOpenBookmarkImport={() => setBookmarkImportOpen(true)}
          onOpenBookmarkExport={() => setBookmarkExportOpen(true)}
          onOpenBackups={() => setBackupsOpen(true)}
        />
      </aside>

      <main class="navigation-main">
        <header class="mobile-header">
          <SiteIdentity
            name={resolvedSite.name}
            logoUrl={resolvedSite.logoUrl}
            compact
          />
          <div class="mobile-header__actions">
            <ThemeSwitcher mode={mode} onChange={setMode} />
            <AuthenticationControls
              compact
              administrator={administrator}
              onSessionChanged={onSessionChanged}
              onOpenSiteSettings={() => setSiteSettingsOpen(true)}
              onOpenBookmarkImport={() => setBookmarkImportOpen(true)}
              onOpenBookmarkExport={() => setBookmarkExportOpen(true)}
              onOpenBackups={() => setBackupsOpen(true)}
            />
          </div>
        </header>

        <div class="navigation-content">
          <header class="navigation-heading">
            <div>
              <p>{messages.navigation.eyebrow}</p>
              <h1>{title}</h1>
              <span>{messages.navigation.linkCount(links.length)}</span>
            </div>
            <div class="navigation-heading__actions">
              {administrator ? (
                <>
                  <Button
                    icon={Plus}
                    aria-label={messages.links.create}
                    disabled={categories.length === 0}
                    onClick={() =>
                      setLinkEditor({
                        categoryID:
                          activeCategory?.id ?? categories[0]?.id ?? '',
                        link: null,
                      })
                    }
                  >
                    {messages.links.create}
                  </Button>
                  <Button
                    icon={ListOrdered}
                    aria-label={messages.links.manage}
                    disabled={categories.length === 0}
                    onClick={() => setLinkOrderOpen(true)}
                  >
                    {messages.links.manage}
                  </Button>
                  <Button
                    icon={Settings}
                    aria-label={messages.categories.manage}
                    onClick={() => setCategoryManagerOpen(true)}
                  >
                    {messages.categories.manage}
                  </Button>
                </>
              ) : null}
              <div class="desktop-theme-switcher">
                <ThemeSwitcher mode={mode} onChange={setMode} />
              </div>
            </div>
          </header>

          <CombinedSearch
            categories={categories}
            engines={searchEngines}
            onConfigure={
              administrator ? () => setSearchSettingsOpen(true) : undefined
            }
          />

          <CategoryNavigation
            variant="mobile"
            categories={categories}
            activeId={activeId}
            onSelect={setActiveId}
          />

          <LinkGrid
            category={activeCategory}
            links={links}
            status={status}
            onRetry={onRetry}
            onEdit={
              administrator
                ? (link) =>
                    setLinkEditor({
                      categoryID:
                        categories.find((category) =>
                          category.links.some((item) => item.id === link.id),
                        )?.id ?? '',
                      link,
                    })
                : undefined
            }
          />
        </div>
      </main>
      <SearchEngineSettings
        open={searchSettingsOpen}
        engines={searchEngines}
        onClose={() => setSearchSettingsOpen(false)}
        onSaved={() => onRetry?.()}
      />
      <SiteSettingsDrawer
        open={siteSettingsOpen}
        initial={resolvedSite}
        onClose={() => setSiteSettingsOpen(false)}
        onSaved={() => {
          setSiteSettingsOpen(false);
          onRetry?.();
        }}
      />
      <BookmarkImportDrawer
        open={bookmarkImportOpen}
        onClose={() => setBookmarkImportOpen(false)}
        onImported={() => {
          setBookmarkImportOpen(false);
          onRetry?.();
        }}
      />
      <BookmarkExportDrawer
        open={bookmarkExportOpen}
        onClose={() => setBookmarkExportOpen(false)}
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
        onSaved={() => onRetry?.()}
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
          onRetry?.();
        }}
      />
      <LinkOrderManager
        open={linkOrderOpen}
        categories={categories}
        initialCategoryID={activeCategory?.id ?? categories[0]?.id ?? ''}
        onClose={() => setLinkOrderOpen(false)}
        onSaved={() => {
          setLinkOrderOpen(false);
          onRetry?.();
        }}
      />
    </div>
  );
}
