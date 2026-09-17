import { messages } from '../i18n/messages';

export type AdminSection =
  'dashboard' | 'links' | 'categories' | 'data' | 'settings';

const adminSectionPathPrefix = '/admin';

export function adminSectionPath(section: AdminSection): string {
  return section === 'dashboard'
    ? adminSectionPathPrefix
    : `${adminSectionPathPrefix}/${section}`;
}

export function adminSectionFromPath(pathname: string): AdminSection | null {
  const rest = pathname
    .replace(/\/+$/, '')
    .slice(adminSectionPathPrefix.length)
    .replace(/^\//, '');
  if (rest === '') return 'dashboard';
  switch (rest) {
    case 'links':
    case 'categories':
    case 'data':
    case 'settings':
      return rest;
    default:
      return null;
  }
}

export function adminSectionTitle(section: AdminSection) {
  return {
    dashboard: messages.admin.dashboard,
    links: messages.admin.links,
    categories: messages.admin.categories,
    data: messages.admin.data,
    settings: messages.admin.settings,
  }[section];
}
