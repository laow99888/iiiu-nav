import { AuthenticationControls } from '../auth/authentication-controls';
import { messages } from '../i18n/messages';
import { SiteIdentity } from '../navigation/site-identity';
import type { SiteSettings } from '../navigation/types';
import {
  Archive,
  LayoutGrid,
  ListOrdered,
  Settings,
  SlidersHorizontal,
} from '../ui/icons/interface-icons';
import type { AdminSection } from './admin-section';

type NavigationProps = {
  onChange: (section: AdminSection) => void;
  section: AdminSection;
};

const navigationItems = [
  { section: 'overview', label: messages.admin.overview, icon: LayoutGrid },
  { section: 'categories', label: messages.admin.categories, icon: Settings },
  { section: 'links', label: messages.admin.links, icon: ListOrdered },
  { section: 'data', label: messages.admin.data, icon: Archive },
  {
    section: 'settings',
    label: messages.admin.settings,
    icon: SlidersHorizontal,
  },
] as const;

export function AdminSidebar({
  onChange,
  onSessionChanged,
  section,
  site,
}: NavigationProps & {
  onSessionChanged: () => void;
  site: SiteSettings;
}) {
  return (
    <aside class="admin-sidebar">
      <div class="admin-sidebar__brand">
        <SiteIdentity name={site.name} logoUrl={site.logoUrl} />
        <span>{messages.admin.badge}</span>
      </div>
      <AdminNavigation
        label={messages.admin.navigationLabel}
        onChange={onChange}
        section={section}
      />
      <div class="admin-sidebar__footer">
        <a href="/">{messages.admin.viewPublic}</a>
        <AuthenticationControls
          administrator
          onSessionChanged={onSessionChanged}
        />
      </div>
    </aside>
  );
}

export function AdminMobileNavigation(props: NavigationProps) {
  return (
    <AdminNavigation
      {...props}
      className="admin-mobile-nav"
      label={messages.admin.mobileNavigationLabel}
    />
  );
}

function AdminNavigation({
  className = 'admin-nav',
  label,
  onChange,
  section,
}: NavigationProps & { className?: string; label: string }) {
  return (
    <nav class={className} aria-label={label}>
      {navigationItems.map((item) => (
        <AdminNavButton
          key={item.section}
          active={section === item.section}
          icon={item.icon}
          onClick={() => onChange(item.section)}
        >
          {item.label}
        </AdminNavButton>
      ))}
    </nav>
  );
}

function AdminNavButton({
  active,
  children,
  icon: Icon,
  onClick,
}: {
  active: boolean;
  children: string;
  icon: typeof LayoutGrid;
  onClick: () => void;
}) {
  return (
    <button
      class={`admin-nav__item${active ? ' is-active' : ''}`}
      aria-current={active ? 'page' : undefined}
      onClick={onClick}
    >
      <Icon aria-hidden="true" />
      <span>{children}</span>
    </button>
  );
}
