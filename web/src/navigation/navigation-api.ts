import {
  categoryIconRegistry,
  type CategoryIconName,
} from '../ui/icons/category-icon-registry';
import {
  configureSearchEngines,
  defaultSearchEngines,
  type SearchEngine,
  type SearchEngineConfig,
  type SearchEngineID,
} from '../search/search-engines';
import {
  defaultSiteSettings,
  type SiteSettings,
  LogoSource,
  LogoTone,
  NavigationCategory,
  NavigationLink,
} from './types';

type NavigationAPIResponse = {
  administrator: boolean;
  searchEngines: SearchEngineConfig[];
  site?: SiteSettings;
  categories: Array<{
    iconName: string;
    id: string;
    links: Array<{
      description: string;
      iconSource: string;
      iconValue: string;
      id: string;
      name: string;
      url: string;
    }>;
    name: string;
    slug: string;
    visibility: 'public' | 'private';
  }>;
};

export type NavigationSnapshot = {
  administrator: boolean;
  categories: NavigationCategory[];
  searchEngines: SearchEngine[];
  site: SiteSettings;
};

const logoTones: readonly LogoTone[] = [
  'blue',
  'green',
  'amber',
  'red',
  'violet',
  'ink',
];

export async function fetchNavigation(
  signal?: AbortSignal,
): Promise<NavigationSnapshot> {
  const response = await fetch('/api/navigation', {
    credentials: 'same-origin',
    headers: { Accept: 'application/json' },
    signal,
  });
  if (!response.ok) {
    throw new Error(`Navigation request failed: ${response.status}`);
  }
  const value: unknown = await response.json();
  if (!isNavigationResponse(value)) {
    throw new Error('Navigation response is invalid');
  }
  return {
    administrator: value.administrator,
    searchEngines: configureSearchEngines(value.searchEngines),
    site: isSiteSettings(value.site) ? value.site : defaultSiteSettings,
    categories: value.categories.map((category) => ({
      id: category.id,
      name: category.name,
      icon: isCategoryIconName(category.iconName) ? category.iconName : null,
      visibility: category.visibility,
      links: category.links.map(mapLink),
    })),
  };
}

function isSiteSettings(value: unknown): value is SiteSettings {
  return (
    isRecord(value) &&
    hasStrings(value, [
      'name',
      'logoUrl',
      'faviconUrl',
      'accentColor',
      'backgroundUrl',
    ]) &&
    typeof value.backgroundOverlay === 'number' &&
    typeof value.indexingEnabled === 'boolean'
  );
}

export async function saveSearchEngines(config: readonly SearchEngineConfig[]) {
  const response = await fetch('/api/settings/search-engines', {
    method: 'PUT',
    credentials: 'same-origin',
    headers: {
      Accept: 'application/json',
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ engines: config }),
  });
  if (!response.ok) {
    throw new Error(`Search engine update failed: ${response.status}`);
  }
}

function mapLink(
  link: NavigationAPIResponse['categories'][number]['links'][number],
): NavigationLink {
  const logoText =
    link.iconSource === 'generated' && link.iconValue.trim()
      ? link.iconValue.trim().slice(0, 3)
      : initialFromName(link.name);
  const logoSource = isLogoSource(link.iconSource)
    ? link.iconSource
    : 'generated';
  const logoURL =
    logoSource !== 'generated' && link.iconValue.startsWith('/uploads/logos/')
      ? link.iconValue
      : null;
  return {
    id: link.id,
    name: link.name,
    description: link.description,
    url: link.url,
    logoText,
    logoTone: toneFromID(link.id),
    logoSource,
    logoURL,
    logoValue: logoURL ?? logoText,
  };
}

function isLogoSource(value: string): value is LogoSource {
  return (
    value === 'auto' ||
    value === 'generated' ||
    value === 'upload' ||
    value === 'url'
  );
}

function initialFromName(name: string) {
  return Array.from(name.trim())[0]?.toLocaleUpperCase() ?? '?';
}

function toneFromID(id: string) {
  let hash = 0;
  for (const character of id) {
    hash = (hash * 31 + character.charCodeAt(0)) >>> 0;
  }
  return logoTones[hash % logoTones.length] ?? 'ink';
}

function isCategoryIconName(value: string): value is CategoryIconName {
  return value in categoryIconRegistry;
}

function isNavigationResponse(value: unknown): value is NavigationAPIResponse {
  if (
    !isRecord(value) ||
    typeof value.administrator !== 'boolean' ||
    !isSearchEngineConfig(value.searchEngines) ||
    !Array.isArray(value.categories)
  ) {
    return false;
  }
  return value.categories.every(
    (category) =>
      isRecord(category) &&
      hasStrings(category, ['id', 'name', 'slug', 'iconName']) &&
      (category.visibility === 'public' || category.visibility === 'private') &&
      Array.isArray(category.links) &&
      category.links.every(
        (link) =>
          isRecord(link) &&
          hasStrings(link, [
            'id',
            'name',
            'description',
            'url',
            'iconSource',
            'iconValue',
          ]),
      ),
  );
}

function isSearchEngineConfig(value: unknown): value is SearchEngineConfig[] {
  if (!Array.isArray(value) || value.length !== defaultSearchEngines.length) {
    return false;
  }
  const supported = new Set(defaultSearchEngines.map((engine) => engine.id));
  const seen = new Set<SearchEngineID>();
  return value.every((item) => {
    if (
      !isRecord(item) ||
      typeof item.id !== 'string' ||
      typeof item.enabled !== 'boolean' ||
      !supported.has(item.id as SearchEngineID) ||
      seen.has(item.id as SearchEngineID)
    ) {
      return false;
    }
    seen.add(item.id as SearchEngineID);
    return true;
  });
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null;
}

function hasStrings(value: Record<string, unknown>, keys: readonly string[]) {
  return keys.every((key) => typeof value[key] === 'string');
}
