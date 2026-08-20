import { render, screen, waitFor, within } from '@testing-library/preact';
import userEvent from '@testing-library/user-event';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { defaultSiteSettings } from '../navigation/types';
import { ToastProvider } from '../ui/primitives';
import { SiteSettingsDrawer } from './site-settings';

function renderSettings(onSaved = vi.fn()) {
  render(
    <ToastProvider>
      <SiteSettingsDrawer
        open
        initial={defaultSiteSettings}
        onClose={() => undefined}
        onSaved={onSaved}
      />
    </ToastProvider>,
  );
  return onSaved;
}

describe('站点设置', () => {
  afterEach(() => vi.unstubAllGlobals());

  it('保存站点名称、强调色、遮罩和索引设置', async () => {
    const user = userEvent.setup({ applyAccept: false });
    const fetchMock = vi
      .fn()
      .mockResolvedValue(new Response('{}', { status: 200 }));
    vi.stubGlobal('fetch', fetchMock);
    const onSaved = renderSettings();
    const drawer = screen.getByRole('dialog', { name: '站点设置' });

    const name = within(drawer).getByLabelText(/站点名称/);
    await user.clear(name);
    await user.type(name, '我的导航');
    await user.click(
      within(drawer).getByRole('checkbox', { name: /允许搜索引擎收录/ }),
    );
    await user.click(within(drawer).getByRole('button', { name: '保存设置' }));

    await waitFor(() => expect(onSaved).toHaveBeenCalledOnce());
    expect(JSON.parse(String(fetchMock.mock.calls[0]?.[1]?.body))).toEqual(
      expect.objectContaining({
        name: '我的导航',
        accentColor: '#305880',
        backgroundOverlay: 78,
        indexingEnabled: true,
      }),
    );
  });

  it('上传三类图片并把服务器路径写入设置', async () => {
    const user = userEvent.setup();
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(
        jsonResponse({ url: '/uploads/site/logo.png' }, 201),
      )
      .mockResolvedValueOnce(
        jsonResponse({ url: '/uploads/site/favicon.png' }, 201),
      )
      .mockResolvedValueOnce(
        jsonResponse({ url: '/uploads/backgrounds/background.png' }, 201),
      )
      .mockResolvedValueOnce(new Response('{}', { status: 200 }));
    vi.stubGlobal('fetch', fetchMock);
    renderSettings();
    const drawer = screen.getByRole('dialog', { name: '站点设置' });

    await user.upload(
      within(drawer).getByLabelText('上传图片', { selector: '#site-logo' }),
      new File(['logo'], 'logo.png', { type: 'image/png' }),
    );
    await user.upload(
      within(drawer).getByLabelText('上传图片', { selector: '#site-favicon' }),
      new File(['favicon'], 'favicon.ico', { type: 'image/x-icon' }),
    );
    await user.upload(
      within(drawer).getByLabelText('上传图片', {
        selector: '#site-background',
      }),
      new File(['background'], 'background.webp', { type: 'image/webp' }),
    );
    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(3));
    expect(within(drawer).getAllByRole('img', { hidden: true })).toHaveLength(
      3,
    );

    await user.click(within(drawer).getByRole('button', { name: '保存设置' }));
    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(4));
    expect(JSON.parse(String(fetchMock.mock.calls[3]?.[1]?.body))).toEqual(
      expect.objectContaining({
        logoUrl: '/uploads/site/logo.png',
        faviconUrl: '/uploads/site/favicon.png',
        backgroundUrl: '/uploads/backgrounds/background.png',
      }),
    );
  });

  it('上传失败显示对应限制且不阻止保存其他设置', async () => {
    const user = userEvent.setup({ applyAccept: false });
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(new Response('', { status: 422 }))
      .mockResolvedValueOnce(new Response('{}', { status: 200 }));
    vi.stubGlobal('fetch', fetchMock);
    renderSettings();
    const drawer = screen.getByRole('dialog', { name: '站点设置' });
    await user.upload(
      within(drawer).getByLabelText('上传图片', { selector: '#site-favicon' }),
      new File(['invalid'], 'invalid.jpg', { type: 'image/jpeg' }),
    );
    expect(await within(drawer).findByRole('alert')).toHaveTextContent(
      'PNG 或 ICO',
    );
    await user.click(within(drawer).getByRole('button', { name: '保存设置' }));
    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(2));
  });
});

function jsonResponse(value: unknown, status: number) {
  return new Response(JSON.stringify(value), {
    status,
    headers: { 'Content-Type': 'application/json' },
  });
}
