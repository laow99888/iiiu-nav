import { messages } from '../i18n/messages';

export type AdminSection =
  'overview' | 'links' | 'categories' | 'data' | 'settings';

export function adminSectionTitle(section: AdminSection) {
  return {
    overview: messages.admin.overview,
    links: messages.admin.links,
    categories: messages.admin.categories,
    data: messages.admin.data,
    settings: messages.admin.settings,
  }[section];
}
