import { render, screen, waitFor, within } from '@testing-library/preact';
import userEvent from '@testing-library/user-event';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { navigationFixtures } from '../fixtures/navigation';
import { NavigationPage } from '../navigation/navigation-page';

function renderAdministrator(onRetry = vi.fn()) {
  render(
    <NavigationPage
      administrator
      categories={navigationFixtures}
      onRetry={onRetry}
    />,
  );
  return onRetry;
}

describe('链接管理', () => {
  afterEach(() => vi.unstubAllGlobals());

  it('创建链接时校验 URL 并保存手动覆盖内容', async () => {
    const user = userEvent.setup();
    const fetchMock = vi
      .fn()
      .mockResolvedValue(new Response('{}', { status: 201 }));
    vi.stubGlobal('fetch', fetchMock);
    const onRetry = renderAdministrator();

    await user.click(screen.getByRole('button', { name: '添加链接' }));
    const drawer = screen.getByRole('dialog', { name: '添加链接' });
    await user.type(
      within(drawer).getByLabelText(/链接地址/),
      'ftp://invalid.example',
    );
    expect(within(drawer).getByRole('alert')).toHaveTextContent(
      'http 或 https',
    );
    expect(within(drawer).getByRole('button', { name: '保存' })).toBeDisabled();
    await user.clear(within(drawer).getByLabelText(/链接地址/));
    await user.type(
      within(drawer).getByLabelText(/链接地址/),
      'https://example.com/docs',
    );
    await user.type(within(drawer).getByLabelText(/名称/), '示例文档');
    await user.type(within(drawer).getByLabelText(/简介/), '手动简介');
    await user.type(within(drawer).getByLabelText(/Logo 文本/), '示例');
    await user.click(within(drawer).getByRole('button', { name: '保存' }));

    await waitFor(() => expect(onRetry).toHaveBeenCalledOnce());
    expect(JSON.parse(String(fetchMock.mock.calls[0]?.[1]?.body))).toEqual({
      categoryId: 'work',
      url: 'https://example.com/docs',
      name: '示例文档',
      description: '手动简介',
      logoText: '示例',
      iconSource: 'generated',
      iconValue: '示例',
    });
  });

  it('裸地址自动补全 HTTPS 后可以直接保存', async () => {
    const user = userEvent.setup();
    const fetchMock = vi
      .fn()
      .mockResolvedValue(new Response('{}', { status: 201 }));
    vi.stubGlobal('fetch', fetchMock);
    renderAdministrator();

    await user.click(screen.getByRole('button', { name: '添加链接' }));
    const drawer = screen.getByRole('dialog', { name: '添加链接' });
    const url = within(drawer).getByLabelText(/链接地址/);
    await user.type(url, 'example.com/docs');
    await user.tab();
    expect(url).toHaveValue('https://example.com/docs');
    await user.type(within(drawer).getByLabelText(/名称/), '示例站点');
    await user.click(within(drawer).getByRole('button', { name: '保存' }));

    await waitFor(() => expect(fetchMock).toHaveBeenCalledOnce());
    expect(JSON.parse(String(fetchMock.mock.calls[0]?.[1]?.body))).toEqual(
      expect.objectContaining({ url: 'https://example.com/docs' }),
    );
  });

  it('识别站点信息后仍允许手动调整并保存缓存 Logo', async () => {
    const user = userEvent.setup();
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(
        new Response(
          JSON.stringify({
            name: 'Example Docs',
            description: 'Recognized description',
            iconSource: 'auto',
            iconValue: '/uploads/logos/recognized.png',
          }),
          { status: 200, headers: { 'Content-Type': 'application/json' } },
        ),
      )
      .mockResolvedValueOnce(new Response('{}', { status: 201 }));
    vi.stubGlobal('fetch', fetchMock);
    const onRetry = renderAdministrator();

    await user.click(screen.getByRole('button', { name: '添加链接' }));
    const drawer = screen.getByRole('dialog', { name: '添加链接' });
    await user.type(
      within(drawer).getByLabelText(/链接地址/),
      'https://example.com/docs',
    );
    await user.click(
      within(drawer).getByRole('button', { name: '识别站点信息' }),
    );

    await waitFor(() =>
      expect(within(drawer).getByLabelText(/名称/)).toHaveValue('Example Docs'),
    );
    expect(drawer.querySelector('img')).toHaveAttribute(
      'src',
      '/uploads/logos/recognized.png',
    );
    const name = within(drawer).getByLabelText(/名称/);
    await user.clear(name);
    await user.type(name, '自定义名称');
    await user.click(within(drawer).getByRole('button', { name: '保存' }));

    await waitFor(() => expect(onRetry).toHaveBeenCalledOnce());
    expect(JSON.parse(String(fetchMock.mock.calls[1]?.[1]?.body))).toEqual(
      expect.objectContaining({
        name: '自定义名称',
        iconSource: 'auto',
        iconValue: '/uploads/logos/recognized.png',
      }),
    );
  });

  it('识别失败不清空表单，且可以上传手动 Logo', async () => {
    const user = userEvent.setup();
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(new Response('', { status: 422 }))
      .mockResolvedValueOnce(
        new Response(JSON.stringify({ url: '/uploads/logos/manual.png' }), {
          status: 201,
          headers: { 'Content-Type': 'application/json' },
        }),
      )
      .mockResolvedValueOnce(new Response('{}', { status: 201 }));
    vi.stubGlobal('fetch', fetchMock);
    renderAdministrator();

    await user.click(screen.getByRole('button', { name: '添加链接' }));
    const drawer = screen.getByRole('dialog', { name: '添加链接' });
    await user.type(
      within(drawer).getByLabelText(/链接地址/),
      'https://offline.example',
    );
    await user.type(within(drawer).getByLabelText(/名称/), '保留名称');
    await user.click(
      within(drawer).getByRole('button', { name: '识别站点信息' }),
    );
    expect(await within(drawer).findByRole('alert')).toHaveTextContent(
      '现有内容未改变',
    );
    expect(within(drawer).getByRole('textbox', { name: /^名称/ })).toHaveValue(
      '保留名称',
    );

    await user.upload(
      within(drawer).getByLabelText('上传 Logo'),
      new File(['png'], 'manual.png', { type: 'image/png' }),
    );
    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(2));
    await user.click(within(drawer).getByRole('button', { name: '保存' }));
    await waitFor(() => expect(fetchMock).toHaveBeenCalledTimes(3));
    expect(JSON.parse(String(fetchMock.mock.calls[2]?.[1]?.body))).toEqual(
      expect.objectContaining({
        iconSource: 'upload',
        iconValue: '/uploads/logos/manual.png',
      }),
    );
  });

  it('编辑链接时可移动分类并覆盖字段', async () => {
    const user = userEvent.setup();
    const fetchMock = vi
      .fn()
      .mockResolvedValue(new Response('{}', { status: 200 }));
    vi.stubGlobal('fetch', fetchMock);
    renderAdministrator();

    await user.click(screen.getByRole('button', { name: '编辑“GitHub”' }));
    const drawer = screen.getByRole('dialog', { name: '编辑链接' });
    await user.selectOptions(
      within(drawer).getByLabelText('所属分类'),
      'development',
    );
    const name = within(drawer).getByLabelText(/名称/);
    await user.clear(name);
    await user.type(name, 'GitHub 工作区');
    const logo = within(drawer).getByLabelText(/Logo 文本/);
    await user.clear(logo);
    await user.type(logo, 'GH');
    await user.click(within(drawer).getByRole('button', { name: '保存' }));

    await waitFor(() => expect(fetchMock).toHaveBeenCalledOnce());
    expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/links/github');
    expect(JSON.parse(String(fetchMock.mock.calls[0]?.[1]?.body))).toEqual(
      expect.objectContaining({
        categoryId: 'development',
        name: 'GitHub 工作区',
        logoText: 'GH',
      }),
    );
  });

  it('删除链接需要确认且失败不会关闭', async () => {
    const user = userEvent.setup();
    const fetchMock = vi
      .fn()
      .mockResolvedValue(new Response('', { status: 500 }));
    vi.stubGlobal('fetch', fetchMock);
    renderAdministrator();

    await user.click(screen.getByRole('button', { name: '编辑“GitHub”' }));
    await user.click(screen.getByRole('button', { name: '删除链接' }));
    const dialog = screen.getByRole('dialog', { name: '删除“GitHub”' });
    expect(within(dialog).getByText('https://github.com/')).toBeInTheDocument();
    await user.click(within(dialog).getByRole('button', { name: '确认删除' }));
    expect(await within(dialog).findByRole('alert')).toHaveTextContent(
      '删除失败',
    );
    expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/links/github');
  });

  it('按分类调整链接顺序并提交完整 ID 集合', async () => {
    const user = userEvent.setup();
    const fetchMock = vi
      .fn()
      .mockResolvedValue(new Response(null, { status: 204 }));
    vi.stubGlobal('fetch', fetchMock);
    renderAdministrator();

    await user.click(screen.getByRole('button', { name: '管理链接' }));
    await user.click(screen.getByRole('button', { name: '下移“GitHub”' }));
    await user.click(screen.getByRole('button', { name: '保存排序' }));

    await waitFor(() => expect(fetchMock).toHaveBeenCalledOnce());
    expect(fetchMock.mock.calls[0]?.[0]).toBe(
      '/api/categories/work/links/order',
    );
    const body = JSON.parse(String(fetchMock.mock.calls[0]?.[1]?.body));
    expect(body.ids).toEqual(['linear', 'github', 'notion', 'gmail']);
  });

  it('可批量重试全部链接的站点信息识别', async () => {
    const user = userEvent.setup();
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ updated: 3, failed: 1 }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      }),
    );
    vi.stubGlobal('fetch', fetchMock);
    renderAdministrator();

    await user.click(screen.getByRole('button', { name: '管理链接' }));
    await user.click(screen.getByRole('button', { name: '全部刷新' }));

    await waitFor(() => expect(fetchMock).toHaveBeenCalledOnce());
    expect(fetchMock.mock.calls[0]?.[0]).toBe('/api/links/refresh');
    expect(
      await screen.findByText('成功 3 个，失败 1 个。'),
    ).toBeInTheDocument();
  });
});
