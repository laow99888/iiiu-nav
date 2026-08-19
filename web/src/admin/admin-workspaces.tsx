import type { ComponentChildren } from 'preact';

import { messages } from '../i18n/messages';
import type { SiteSettings } from '../navigation/types';
import type { SearchEngine } from '../search/search-engines';
import {
  Archive,
  FileDown,
  FileUp,
  Search,
  Settings,
  SlidersHorizontal,
  type LucideIcon,
} from '../ui/icons/interface-icons';
import { Button } from '../ui/primitives';

type DataWorkspaceProps = {
  onOpenBackups: () => void;
  onOpenExport: () => void;
  onOpenImport: () => void;
};

export function AdminDataWorkspace({
  onOpenBackups,
  onOpenExport,
  onOpenImport,
}: DataWorkspaceProps) {
  return (
    <AdminWorkspace
      kicker={messages.admin.dataWorkspaceKicker}
      title={messages.admin.dataWorkspaceTitle}
      count={messages.admin.operationCount(3)}
    >
      <AdminWorkspaceRow
        icon={FileUp}
        title={messages.imports.menu}
        summary={messages.admin.importSummary}
        details={[messages.admin.formats.html, messages.admin.formats.json]}
        action={messages.imports.menu}
        onClick={onOpenImport}
      />
      <AdminWorkspaceRow
        icon={FileDown}
        tone="amber"
        title={messages.exports.menu}
        summary={messages.admin.exportSummary}
        details={[messages.admin.formats.html, messages.admin.formats.json]}
        action={messages.exports.menu}
        onClick={onOpenExport}
      />
      <AdminWorkspaceRow
        icon={Archive}
        tone="green"
        title={messages.backups.menu}
        summary={messages.admin.backupSummary}
        details={[messages.admin.formats.zip, messages.admin.completeBackup]}
        action={messages.backups.menu}
        onClick={onOpenBackups}
      />
    </AdminWorkspace>
  );
}

type SettingsWorkspaceProps = {
  engines: readonly SearchEngine[];
  onOpenSearchSettings: () => void;
  onOpenSiteSettings: () => void;
  site: SiteSettings;
};

export function AdminSettingsWorkspace({
  engines,
  onOpenSearchSettings,
  onOpenSiteSettings,
  site,
}: SettingsWorkspaceProps) {
  const enabledEngines = engines.filter((engine) => engine.enabled);
  const engineSummary = enabledEngines.length
    ? enabledEngines.map((engine) => engine.name).join('、')
    : messages.admin.noEnabledSearchEngines;

  return (
    <AdminWorkspace
      kicker={messages.admin.settingsWorkspaceKicker}
      title={messages.admin.settingsWorkspaceTitle}
      count={messages.admin.settingCount(3)}
    >
      <AdminWorkspaceRow
        icon={SlidersHorizontal}
        title={messages.admin.siteAppearance}
        summary={site.name}
        details={[
          site.logoUrl ? messages.admin.customLogo : messages.admin.defaultLogo,
          site.backgroundUrl
            ? messages.admin.customBackground
            : messages.admin.defaultBackground,
          site.accentColor.toUpperCase(),
        ]}
        accentColor={site.accentColor}
        action={messages.admin.editSiteSettings}
        onClick={onOpenSiteSettings}
      />
      <AdminWorkspaceRow
        icon={Search}
        tone="amber"
        title={messages.search.settingsTitle}
        summary={engineSummary}
        details={[messages.admin.enabledSearchEngines(enabledEngines.length)]}
        action={messages.search.configure}
        onClick={onOpenSearchSettings}
      />
      <AdminWorkspaceRow
        icon={Settings}
        tone={site.indexingEnabled ? 'green' : 'neutral'}
        title={messages.settings.privacy}
        summary={
          site.indexingEnabled
            ? messages.admin.indexingEnabled
            : messages.admin.indexingDisabled
        }
        details={[
          site.indexingEnabled
            ? messages.admin.publicIndexing
            : messages.admin.privateIndexing,
        ]}
        action={messages.admin.editIndexing}
        onClick={onOpenSiteSettings}
      />
    </AdminWorkspace>
  );
}

function AdminWorkspace({
  children,
  count,
  kicker,
  title,
}: {
  children: ComponentChildren;
  count: string;
  kicker: string;
  title: string;
}) {
  const titleID = `admin-workspace-${title}`;
  return (
    <section class="admin-workspace" aria-labelledby={titleID}>
      <header class="admin-workspace__header">
        <div>
          <span class="admin-section-kicker">{kicker}</span>
          <h2 id={titleID}>{title}</h2>
        </div>
        <span class="admin-workspace__count">{count}</span>
      </header>
      <div class="admin-workspace__list">{children}</div>
    </section>
  );
}

function AdminWorkspaceRow({
  accentColor,
  action,
  details,
  icon: Icon,
  onClick,
  summary,
  title,
  tone = 'blue',
}: {
  accentColor?: string;
  action: string;
  details: readonly string[];
  icon: LucideIcon;
  onClick: () => void;
  summary: string;
  title: string;
  tone?: 'amber' | 'blue' | 'green' | 'neutral';
}) {
  return (
    <article class="admin-workspace-row">
      <span
        class={`admin-workspace-row__icon admin-workspace-row__icon--${tone}`}
        aria-hidden="true"
      >
        <Icon />
      </span>
      <div class="admin-workspace-row__content">
        <h3>{title}</h3>
        <p>{summary}</p>
        <div class="admin-workspace-row__details">
          {accentColor ? (
            <span>
              <i style={{ background: accentColor }} />
              {messages.admin.accentColor}
            </span>
          ) : null}
          {details.map((detail) => (
            <span key={detail}>{detail}</span>
          ))}
        </div>
      </div>
      <Button icon={Icon} onClick={onClick}>
        {action}
      </Button>
    </article>
  );
}
