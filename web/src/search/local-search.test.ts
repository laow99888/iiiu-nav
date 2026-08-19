import { describe, expect, it } from 'vitest';

import { navigationFixtures } from '../fixtures/navigation';
import { searchLocalLinks } from './local-search';
import {
  buildSearchURL,
  defaultSearchEngines,
  enabledSearchEngines,
} from './search-engines';

describe('本地与网页搜索模型', () => {
  it('按名称、简介和网址匹配且不超过限制', () => {
    expect(searchLocalLinks(navigationFixtures, 'github')[0]?.link.name).toBe(
      'GitHub',
    );
    expect(searchLocalLinks(navigationFixtures, '知识整理')[0]?.link.name).toBe(
      'Notion',
    );
    expect(
      searchLocalLinks(navigationFixtures, 'sqlite.org')[0]?.link.name,
    ).toBe('SQLite');
    expect(searchLocalLinks(navigationFixtures, 'i', 2)).toHaveLength(2);
    expect(searchLocalLinks(navigationFixtures, '  ')).toEqual([]);
  });

  it('搜索引擎按配置过滤并安全编码查询词', () => {
    const configured = [
      { ...defaultSearchEngines[2]!, enabled: true },
      { ...defaultSearchEngines[0]!, enabled: false },
    ];
    expect(enabledSearchEngines(configured).map((engine) => engine.id)).toEqual(
      ['bing'],
    );
    expect(buildSearchURL(defaultSearchEngines[0]!, 'Go & SQLite')).toBe(
      'https://www.google.com/search?q=Go%20%26%20SQLite',
    );
  });
});
