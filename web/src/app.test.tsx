import { render, screen, waitFor } from '@testing-library/preact';
import userEvent from '@testing-library/user-event';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { App } from './app';

const navigationBody = {
  administrator: false,
  searchEngines: [
    { id: 'google', enabled: true },
    { id: 'baidu', enabled: true },
    { id: 'bing', enabled: true },
    { id: 'duckduckgo', enabled: true },
  ],
  categories: [
    {
      id: '1',
      name: '公开分类',
      slug: 'public',
      iconName: 'globe',
      visibility: 'public',
      links: [
        {
          id: '2',
          name: '公开链接',
          description: '来自真实 API 形状',
          url: 'https://example.com/',
          iconSource: 'generated',
          iconValue: '公',
        },
      ],
    },
  ],
};

describe('应用导航接入', () => {
  afterEach(() => vi.unstubAllGlobals());

  it('从加载态进入 API 数据视图', async () => {
    let resolveResponse: ((response: Response) => void) | undefined;
    vi.stubGlobal(
      'fetch',
      vi.fn().mockReturnValue(
        new Promise<Response>((resolve) => {
          resolveResponse = resolve;
        }),
      ),
    );
    render(<App />);

    expect(screen.getByText('正在载入导航')).toBeInTheDocument();
    resolveResponse?.(
      new Response(JSON.stringify(navigationBody), { status: 200 }),
    );

    expect(
      await screen.findByRole('link', { name: /公开链接/ }),
    ).toBeInTheDocument();
    expect(screen.queryByText('正在载入导航')).not.toBeInTheDocument();
  });

  it('请求失败后可重试恢复', async () => {
    const user = userEvent.setup();
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(new Response('', { status: 503 }))
      .mockResolvedValueOnce(
        new Response(JSON.stringify(navigationBody), { status: 200 }),
      );
    vi.stubGlobal('fetch', fetchMock);
    render(<App />);

    expect(await screen.findByText('暂时无法载入导航')).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: '重新加载' }));
    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(2));
    expect(
      await screen.findByRole('link', { name: /公开链接/ }),
    ).toBeInTheDocument();
  });
});
