import { render, screen, waitFor } from '@testing-library/preact';
import userEvent from '@testing-library/user-event';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { SearchEngineSettings } from './search-engine-settings';
import { defaultSearchEngines } from './search-engines';

describe('搜索引擎设置', () => {
  afterEach(() => vi.unstubAllGlobals());

  it('管理员可启停、排序并保存固定引擎', async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();
    const onSaved = vi.fn();
    const fetchMock = vi
      .fn()
      .mockResolvedValue(new Response(null, { status: 204 }));
    vi.stubGlobal('fetch', fetchMock);

    render(
      <SearchEngineSettings
        open
        engines={defaultSearchEngines}
        onClose={onClose}
        onSaved={onSaved}
      />,
    );

    await user.click(screen.getByRole('checkbox', { name: 'Google' }));
    await user.click(screen.getByRole('button', { name: '上移“Bing”' }));
    await user.click(screen.getByRole('button', { name: '保存' }));

    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(1));
    expect(JSON.parse(String(fetchMock.mock.calls[0]?.[1]?.body))).toEqual({
      engines: [
        { id: 'google', enabled: false },
        { id: 'bing', enabled: true },
        { id: 'baidu', enabled: true },
        { id: 'duckduckgo', enabled: true },
      ],
    });
    expect(onSaved).toHaveBeenCalledOnce();
    expect(onClose).toHaveBeenCalledOnce();
  });

  it('至少保留一个启用引擎', async () => {
    const user = userEvent.setup();
    render(
      <SearchEngineSettings
        open
        engines={defaultSearchEngines}
        onClose={vi.fn()}
        onSaved={vi.fn()}
      />,
    );

    for (const engine of defaultSearchEngines) {
      await user.click(screen.getByRole('checkbox', { name: engine.name }));
    }

    expect(screen.getByRole('alert')).toHaveTextContent(
      '至少保留一个搜索引擎。',
    );
    expect(screen.getByRole('button', { name: '保存' })).toBeDisabled();
  });
});
