import { messages } from '../i18n/messages';

export type AdminSection =
  'dashboard' | 'links' | 'categories' | 'data' | 'settings';

export function adminSectionTitle(section: AdminSection) {
  return {
    dashboard: messages.admin.dashboard,
    links: messages.admin.links,
    categories: messages.admin.categories,
    data: messages.admin.data,
    settings: messages.admin.settings,
  }[section];
}
