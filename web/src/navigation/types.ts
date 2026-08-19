import type { CategoryIconName } from '../ui/icons/category-icon-registry';

export type LogoTone = 'blue' | 'green' | 'amber' | 'red' | 'violet' | 'ink';
export type LogoSource = 'auto' | 'generated' | 'upload' | 'url';

export type NavigationLink = {
  description: string;
  id: string;
  logoText: string;
  logoTone: LogoTone;
  logoSource?: LogoSource;
  logoURL?: string | null;
  logoValue?: string;
  name: string;
  url: string;
};

export type NavigationCategory = {
  icon: CategoryIconName | null;
  id: string;
  links: readonly NavigationLink[];
  name: string;
  visibility: 'public' | 'private';
};

export type SiteSettings = {
  accentColor: string;
  backgroundOverlay: number;
  backgroundUrl: string;
  faviconUrl: string;
  indexingEnabled: boolean;
  logoUrl: string;
  name: string;
};

export const defaultSiteSettings: SiteSettings = {
  name: 'iiiu-nav',
  logoUrl: '',
  faviconUrl: '',
  accentColor: '#1769aa',
  backgroundUrl: '',
  backgroundOverlay: 78,
  indexingEnabled: false,
};
