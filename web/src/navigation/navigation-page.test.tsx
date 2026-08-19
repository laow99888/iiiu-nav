import { render, screen } from '@testing-library/preact';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it } from 'vitest';

import { navigationFixtures } from '../fixtures/navigation';
import { NavigationPage } from './navigation-page';
import { defaultSiteSettings } from './types';

describe('导航页面', () => {
  beforeEach(() => {
    localStorage.clear();
    document.documentElement.removeAttribute('data-theme');
  });

  it('展示站点身份、全部链接和安全的外链属性', () => {
    render(
      <NavigationPage categories={navigationFixtures} siteName="我的工作台" />,
    );

    expect(screen.getAllByText('我的工作台')).toHaveLength(2);
    expect(
      screen.getByRole('heading', { name: '全部链接' }),
    ).toBeInTheDocument();
    expect(screen.getByText('17 个链接')).toBeInTheDocument();

    const github = screen.getByRole('link', { name: /GitHub/ });
    expect(github).toHaveAttribute('target', '_blank');
    expect(github).toHaveAttribute('rel', 'noopener noreferrer');
  });

  it('切换分类后只呈现对应链接', async () => {
    const user = userEvent.setup();
    render(<NavigationPage categories={navigationFixtures} />);

    await user.click(screen.getAllByRole('button', { name: /开发/ })[0]!);
    expect(screen.getByRole('heading', { name: '开发' })).toBeInTheDocument();
    expect(screen.getByText('4 个链接')).toBeInTheDocument();
    expect(
      screen.getByRole('link', { name: /MDN Web Docs/ }),
    ).toBeInTheDocument();
    expect(
      screen.queryByRole('link', { name: /GitHub/ }),
    ).not.toBeInTheDocument();
  });

  it('空分类和完全空导航都有完整空状态', async () => {
    const user = userEvent.setup();
    const { rerender } = render(
      <NavigationPage categories={navigationFixtures} />,
    );

    await user.click(screen.getAllByRole('button', { name: /稍后看/ })[0]!);
    expect(screen.getByText('这个分类还是空的')).toBeInTheDocument();

    rerender(<NavigationPage categories={[]} />);
    expect(screen.getByText('还没有导航内容')).toBeInTheDocument();
    expect(screen.getByText('0 个链接')).toBeInTheDocument();
  });

  it('加载导航时展示保持卡片几何的骨架状态', () => {
    const { container } = render(
      <NavigationPage categories={[]} status="loading" />,
    );

    expect(
      screen.getByRole('status', { name: '正在载入导航' }),
    ).toHaveAttribute('aria-busy', 'true');
    expect(container.querySelectorAll('.link-card--skeleton')).toHaveLength(6);
    expect(screen.queryByText('还没有导航内容')).not.toBeInTheDocument();
  });

  it('主题选择立即应用并写入浏览器存储', async () => {
    const user = userEvent.setup();
    render(<NavigationPage categories={navigationFixtures} />);

    await user.click(
      screen.getAllByRole('button', { name: '使用暗色主题' })[0]!,
    );
    expect(document.documentElement).toHaveAttribute('data-theme', 'dark');
    expect(localStorage.getItem('iiiu-nav.theme')).toBe('dark');

    await user.click(
      screen.getAllByRole('button', { name: '跟随系统主题' })[0]!,
    );
    expect(document.documentElement).not.toHaveAttribute('data-theme');
    expect(localStorage.getItem('iiiu-nav.theme')).toBe('system');
  });

  it('应用站点名称、Logo、favicon、强调色和背景设置', () => {
    render(
      <NavigationPage
        categories={navigationFixtures}
        site={{
          ...defaultSiteSettings,
          name: '我的入口',
          logoUrl: '/uploads/site/logo.png',
          faviconUrl: '/uploads/site/favicon.png',
          accentColor: '#2f6f55',
          backgroundUrl: '/uploads/backgrounds/background.png',
          backgroundOverlay: 74,
        }}
      />,
    );

    expect(screen.getAllByText('我的入口')).toHaveLength(2);
    expect(document.title).toBe('我的入口');
    expect(document.querySelector('link[rel="icon"]')).toHaveAttribute(
      'href',
      '/uploads/site/favicon.png',
    );
    const shell = document.querySelector('.navigation-shell');
    expect(shell).toHaveClass('has-background');
    expect(shell).toHaveStyle('--site-accent: #2f6f55');
    expect(screen.getAllByRole('img', { hidden: true }).length).toBeGreaterThan(
      1,
    );
  });

  it('管理员菜单打开站点设置抽屉', async () => {
    const user = userEvent.setup();
    render(<NavigationPage administrator categories={navigationFixtures} />);

    await user.click(screen.getAllByRole('button', { name: /管理员菜单/ })[0]!);
    await user.click(screen.getByRole('menuitem', { name: '站点设置' }));
    expect(
      screen.getByRole('dialog', { name: '站点设置' }),
    ).toBeInTheDocument();
  });
});
