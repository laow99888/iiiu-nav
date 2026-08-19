import { render, screen, within } from '@testing-library/preact';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { navigationFixtures } from '../fixtures/navigation';
import type { NavigationCategory } from '../navigation/types';
import { AdminPage } from './admin-page';

describe('管理后台', () => {
  beforeEach(() => {
    document.documentElement.removeAttribute('data-theme');
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
      screen.getByRole('region', { name: '访客趋势' }),
    ).toBeInTheDocument();
    expect(screen.getAllByText('模拟数据').length).toBeGreaterThan(0);

    const initialVisits = screen.getByTestId('visitor-total').textContent;
    await user.click(screen.getByRole('button', { name: '近 30 天' }));
    expect(screen.getByTestId('visitor-total').textContent).not.toBe(
      initialVisits,
    );

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
});
