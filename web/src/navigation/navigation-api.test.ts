import { afterEach, describe, expect, it, vi } from 'vitest';

import { fetchNavigation, saveSearchEngines } from './navigation-api';

const searchEngines = [
  { id: 'bing', enabled: true },
  { id: 'google', enabled: false },
  { id: 'baidu', enabled: true },
  { id: 'duckduckgo', enabled: true },
] as const;

describe('导航 API 适配器', () => {
  afterEach(() => vi.unstubAllGlobals());

  it('映射公开和私有分类并回退未知图标', async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(
        JSON.stringify({
          administrator: true,
          searchEngines,
          categories: [
            {
              id: '10',
              name: '私有工作',
              slug: 'private-work',
              iconName: 'removed-icon',
              visibility: 'private',
              links: [
                {
                  id: '20',
                  name: '内部系统',
                  description: '仅管理员可见',
                  url: 'https://internal.example/',
                  iconSource: 'generated',
                  iconValue: '内',
                },
              ],
            },
          ],
        }),
        { status: 200, headers: { 'Content-Type': 'application/json' } },
      ),
    );
    vi.stubGlobal('fetch', fetchMock);

    const result = await fetchNavigation();

    expect(fetchMock).toHaveBeenCalledWith(
      '/api/navigation',
      expect.objectContaining({ credentials: 'same-origin' }),
    );
    expect(result.administrator).toBe(true);
    expect(result.searchEngines.map((engine) => engine.id)).toEqual([
      'bing',
      'google',
      'baidu',
      'duckduckgo',
    ]);
    expect(result.searchEngines[1]?.enabled).toBe(false);
    expect(result.categories[0]).toMatchObject({
      icon: null,
      visibility: 'private',
      links: [{ logoText: '内' }],
    });
  });

  it('拒绝错误状态和结构不完整的响应', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(new Response('', { status: 503 })),
    );
    await expect(fetchNavigation()).rejects.toThrow('503');

    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        new Response(
          JSON.stringify({
            administrator: false,
            categories: [{ name: '缺字段' }],
          }),
        ),
      ),
    );
    await expect(fetchNavigation()).rejects.toThrow('invalid');
  });

  it('使用管理员设置端点保存引擎顺序和启停状态', async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValue(new Response(null, { status: 204 }));
    vi.stubGlobal('fetch', fetchMock);

    await saveSearchEngines(searchEngines);

    expect(fetchMock).toHaveBeenCalledWith(
      '/api/settings/search-engines',
      expect.objectContaining({
        method: 'PUT',
        credentials: 'same-origin',
        body: JSON.stringify({ engines: searchEngines }),
      }),
    );
  });
});
