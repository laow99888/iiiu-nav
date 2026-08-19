import { render, screen, waitFor, within } from '@testing-library/preact';
import userEvent from '@testing-library/user-event';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { BookmarkExportDrawer } from './bookmark-export';

describe('书签导出', () => {
  afterEach(() => {
    vi.restoreAllMocks();
    vi.unstubAllGlobals();
  });

  it('按范围下载浏览器 HTML 和应用 JSON', async () => {
    const user = userEvent.setup();
    const fetchMock = vi
      .fn()
      .mockResolvedValue(new Response('export', { status: 200 }));
    vi.stubGlobal('fetch', fetchMock);
    const createURL = vi.fn().mockReturnValue('blob:bookmark-export');
    const revokeURL = vi.fn();
    vi.stubGlobal('URL', {
      ...URL,
      createObjectURL: createURL,
      revokeObjectURL: revokeURL,
    });
    const click = vi
      .spyOn(HTMLAnchorElement.prototype, 'click')
      .mockImplementation(() => undefined);
    render(<BookmarkExportDrawer open onClose={() => undefined} />);
    const drawer = screen.getByRole('dialog', { name: '导出书签' });

    await user.selectOptions(
      within(drawer).getByLabelText('导出范围'),
      'private',
    );
    await user.click(
      within(drawer).getAllByRole('button', { name: /下载/ })[0]!,
    );
    await waitFor(() => expect(fetchMock).toHaveBeenCalledOnce());
    expect(fetchMock.mock.calls[0]?.[0]).toBe(
      '/api/bookmarks/export?format=html&scope=private',
    );
    expect(click).toHaveBeenCalledOnce();
    expect(createURL).toHaveBeenCalledOnce();
    expect(revokeURL).toHaveBeenCalledWith('blob:bookmark-export');
  });

  it('导出失败时显示错误且重新启用操作', async () => {
    const user = userEvent.setup();
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(new Response('', { status: 500 })),
    );
    render(<BookmarkExportDrawer open onClose={() => undefined} />);
    const drawer = screen.getByRole('dialog', { name: '导出书签' });
    const download = within(drawer).getAllByRole('button', {
      name: /下载/,
    })[1]!;
    await user.click(download);
    expect(await within(drawer).findByRole('alert')).toHaveTextContent(
      '导出失败',
    );
    expect(download).toBeEnabled();
  });
});
