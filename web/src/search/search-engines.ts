export type SearchEngineID = 'google' | 'baidu' | 'bing' | 'duckduckgo';

export type SearchEngine = {
  enabled: boolean;
  id: SearchEngineID;
  mark: string;
  name: string;
  searchURL: string;
};

export type SearchEngineConfig = Pick<SearchEngine, 'id' | 'enabled'>;

export const defaultSearchEngines: readonly SearchEngine[] = [
  {
    id: 'google',
    name: 'Google',
    mark: 'G',
    searchURL: 'https://www.google.com/search?q={query}',
    enabled: true,
  },
  {
    id: 'baidu',
    name: '百度',
    mark: '百',
    searchURL: 'https://www.baidu.com/s?wd={query}',
    enabled: true,
  },
  {
    id: 'bing',
    name: 'Bing',
    mark: 'B',
    searchURL: 'https://www.bing.com/search?q={query}',
    enabled: true,
  },
  {
    id: 'duckduckgo',
    name: 'DuckDuckGo',
    mark: 'D',
    searchURL: 'https://duckduckgo.com/?q={query}',
    enabled: true,
  },
];

export function enabledSearchEngines(engines: readonly SearchEngine[]) {
  return engines.filter((engine) => engine.enabled);
}

export function buildSearchURL(engine: SearchEngine, query: string) {
  return engine.searchURL.replace('{query}', encodeURIComponent(query.trim()));
}

export function configureSearchEngines(config: readonly SearchEngineConfig[]) {
  const definitions = new Map(
    defaultSearchEngines.map((engine) => [engine.id, engine]),
  );
  return config.map((item) => ({
    ...definitions.get(item.id)!,
    enabled: item.enabled,
  }));
}
