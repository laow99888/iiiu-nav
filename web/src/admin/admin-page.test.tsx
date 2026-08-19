import { render, screen, within } from '@testing-library/preact';
import userEvent from '@testing-library/user-event';
import { beforeEach, describe, expect, it, vi } from 'vitest';

import { navigationFixtures } from '../fixtures/navigation';
import { AdminPage } from './admin-page';

describe('管理后台', () => {
  beforeEach(() => {
    document.documentElement.removeAttribute('data-theme');
  });

  it('提供独立的后台壳和模块导航', async () => {
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
    expect(screen.getByRole('heading', { name: '概览' })).toBeInTheDocument();
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

  it('默认显示全部分类中的链接', () => {
    render(
      <AdminPage
        categories={navigationFixtures}
        onRetry={vi.fn()}
        onSessionChanged={vi.fn()}
      />,
    );

    expect(screen.getByRole('combobox', { name: '当前分类' })).toHaveValue('');
    expect(screen.getAllByRole('link').length).toBeGreaterThanOrEqual(2);
  });
});
