import { render, screen, waitFor, within } from '@testing-library/preact';
import userEvent from '@testing-library/user-event';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

import { navigationFixtures } from '../fixtures/navigation';
import type { NavigationCategory } from '../navigation/types';
import { defaultSearchEngines } from '../search/search-engines';
import { AdminPage } from './admin-page';

describe('管理后台', () => {
  beforeEach(() => {
    document.documentElement.removeAttribute('data-theme');
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        new Response(
          JSON.stringify({
            series: Array.from({ length: 30 }, (_, index) => ({
              date: `2026-07-${String(index + 1).padStart(2, '0')}`,
              views: index + 1,
            })),
          }),
          { status: 200, headers: { 'Content-Type': 'application/json' } },
        ),
      ),
    );
  });

  afterEach(() => vi.unstubAllGlobals());

  // 放在最前：此时 jsdom 历史尚未被其他测试的抽屉卸载遍历（异步
  // history.back()）污染，历史状态断言才是确定性的。
  it('分区切换写入模块 URL 且保留历史状态对象', async () => {
    const user = userEvent.setup();
    window.history.replaceState({ marker: 'kept' }, '', '/admin');
    const replaceState = vi.spyOn(window.history, 'replaceState');
    render(
      <AdminPage
        categories={navigationFixtures}
        onRetry={vi.fn()}
        onSessionChanged={vi.fn()}
      />,
    );

    const navigation = screen.getByRole('navigation', { name: '后台导航' });
    await user.click(
      within(navigation).getByRole('button', { name: '链接管理' }),
    );

    expect(replaceState).toHaveBeenCalled();
    const [state, , url] = replaceState.mock.calls.at(-1)!;
    expect(url).toBe('/admin/links');
    expect(state).toEqual({ marker: 'kept' });
    replaceState.mockRestore();
    // jsdom 的 URL 跨测试保留，还原路径让后续测试从公共首页开始。
    window.history.replaceState(null, '', '/');
  });

  it('提供带访客趋势的仪表盘并支持周期切换', async () => {
    const user = userEvent.setup();
    render(
      <AdminPage
        categories={navigationFixtures}
        onRetry={vi.fn()}
        onSessionChanged={vi.fn()}
      />,
    );

    const desktopNavigation = screen.getByRole('navigation', {
      name: '后台导航',
    });
    expect(desktopNavigation).toBeInTheDocument();
    expect(
      screen.getByRole('navigation', { name: '后台模块导航' }),
    ).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: '仪表盘' })).toBeInTheDocument();
    expect(
      screen.getByRole('region', { name: '每日浏览趋势' }),
    ).toBeInTheDocument();
    expect(screen.getByText('每日 PV')).toBeInTheDocument();

    await waitFor(() =>
      expect(screen.getByTestId('visitor-total')).toHaveTextContent('189'),
    );
    await user.click(screen.getByRole('button', { name: '近 30 天' }));
    expect(screen.getByTestId('visitor-total')).toHaveTextContent('465');

    await user.click(
      within(desktopNavigation).getByRole('button', { name: '数据管理' }),
    );
    expect(
      screen.getByRole('heading', { name: '数据管理' }),
    ).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: '数据备份' }),
    ).toBeInTheDocument();
  });

  it('链接管理使用独立数据表展示完整字段和隐私状态', async () => {
    const user = userEvent.setup();
    const privateCategory: NavigationCategory = {
      id: 'private',
      icon: null,
      links: [
        {
          description: '仅管理员使用',
          id: 'private-link',
          logoText: 'P',
          logoTone: 'ink',
          name: 'Private Console',
          url: 'https://private.example.com',
        },
      ],
      name: '私有工具',
      visibility: 'private',
    };
    render(
      <AdminPage
        categories={[navigationFixtures[0]!, privateCategory]}
        onRetry={vi.fn()}
        onSessionChanged={vi.fn()}
      />,
    );

    const navigation = screen.getByRole('navigation', { name: '后台导航' });
    await user.click(
      within(navigation).getByRole('button', { name: '链接管理' }),
    );

    const table = screen.getByRole('table', { name: '链接管理列表' });
    for (const column of [
      '序号',
      'URL 地址',
      '站点名称',
      '简介',
      '分类',
      '隐私',
      '操作',
    ]) {
      expect(
        within(table).getByRole('columnheader', { name: column }),
      ).toBeInTheDocument();
    }
    expect(within(table).getByText('GitHub')).toBeInTheDocument();
    expect(within(table).getByText('Private Console')).toBeInTheDocument();
    expect(within(table).getAllByText('公开').length).toBeGreaterThan(0);
    expect(within(table).getByText('私有')).toBeInTheDocument();
    expect(
      within(table).getByRole('button', { name: '编辑 GitHub' }),
    ).toBeInTheDocument();
  });

  it('分类管理使用数据表并从行操作直接打开编辑表单', async () => {
    const user = userEvent.setup();
    render(
      <AdminPage
        categories={navigationFixtures}
        onRetry={vi.fn()}
        onSessionChanged={vi.fn()}
      />,
    );

    const navigation = screen.getByRole('navigation', { name: '后台导航' });
    await user.click(
      within(navigation).getByRole('button', { name: '分类管理' }),
    );

    const table = screen.getByRole('table', { name: '分类管理列表' });
    for (const column of [
      '序号',
      '图标',
      '分类名称',
      '链接数',
      '可见性',
      '操作',
    ]) {
      expect(
        within(table).getByRole('columnheader', { name: column }),
      ).toBeInTheDocument();
    }
    const developmentRow = within(table).getByRole('row', { name: /开发/ });
    expect(within(developmentRow).getByText('02')).toBeInTheDocument();
    await user.click(
      within(developmentRow).getByRole('button', { name: '编辑“开发”' }),
    );
    expect(
      screen.getByRole('dialog', { name: '编辑分类' }),
    ).toBeInTheDocument();
  });

  it('链接列表区分无内容与筛选无结果状态', async () => {
    const user = userEvent.setup();
    const emptyCategory: NavigationCategory = {
      id: 'empty',
      icon: null,
      links: [],
      name: '空分类',
      visibility: 'public',
    };
    const view = render(
      <AdminPage
        categories={[emptyCategory]}
        onRetry={vi.fn()}
        onSessionChanged={vi.fn()}
      />,
    );
    const navigation = screen.getByRole('navigation', { name: '后台导航' });
    await user.click(
      within(navigation).getByRole('button', { name: '链接管理' }),
    );
    expect(screen.getByText('还没有链接')).toBeInTheDocument();
    expect(
      screen.getByRole('button', { name: '添加第一个链接' }),
    ).toBeInTheDocument();

    view.rerender(
      <AdminPage
        categories={navigationFixtures}
        onRetry={vi.fn()}
        onSessionChanged={vi.fn()}
      />,
    );
    await user.type(
      screen.getByRole('searchbox', { name: '搜索链接' }),
      '不存在',
    );
    expect(screen.getByText('没有匹配的链接')).toBeInTheDocument();
  });

  it('数据管理和系统设置使用工作区摘要并复用既有抽屉', async () => {
    const user = userEvent.setup();
    render(
      <AdminPage
        categories={navigationFixtures}
        onRetry={vi.fn()}
        onSessionChanged={vi.fn()}
        searchEngines={defaultSearchEngines}
        site={{
          accentColor: '#0f766e',
          backgroundOverlay: 78,
          backgroundUrl: '',
          faviconUrl: '',
          indexingEnabled: false,
          logoUrl: '',
          name: '我的导航',
        }}
      />,
    );

    const navigation = screen.getByRole('navigation', { name: '后台导航' });
    await user.click(
      within(navigation).getByRole('button', { name: '数据管理' }),
    );
    expect(
      screen.getByRole('region', { name: '数据操作' }),
    ).toBeInTheDocument();
    expect(
      within(screen.getByRole('region', { name: '数据操作' })).getAllByText(
        'HTML',
      ),
    ).toHaveLength(2);
    await user.click(screen.getByRole('button', { name: '导入书签' }));
    expect(
      screen.getByRole('dialog', { name: '导入书签' }),
    ).toBeInTheDocument();
    await user.click(screen.getByRole('button', { name: '关闭抽屉' }));

    await user.click(
      within(navigation).getByRole('button', { name: '系统设置' }),
    );
    const settings = screen.getByRole('region', { name: '站点配置' });
    expect(within(settings).getByText('我的导航')).toBeInTheDocument();
    expect(
      within(settings).getByRole('heading', { name: '版本与更新' }),
    ).toBeInTheDocument();
    expect(within(settings).getByText('已启用 4 个')).toBeInTheDocument();
    expect(within(settings).getByText('禁止搜索引擎收录')).toBeInTheDocument();
    await user.click(
      within(settings).getByRole('button', { name: '编辑站点设置' }),
    );
    expect(
      screen.getByRole('dialog', { name: '站点设置' }),
    ).toBeInTheDocument();
  });
});
