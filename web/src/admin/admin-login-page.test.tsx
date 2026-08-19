import { render, screen } from '@testing-library/preact';
import { describe, expect, it } from 'vitest';

import { defaultSiteSettings } from '../navigation/types';
import { AdminLoginPage } from './admin-login-page';

describe('管理员登录页', () => {
  it('使用单一站点身份和紧凑登录层级', () => {
    const { container } = render(
      <AdminLoginPage site={{ ...defaultSiteSettings, name: '我的导航' }} />,
    );

    expect(screen.getAllByText('我的导航')).toHaveLength(1);
    expect(
      screen.getByRole('heading', { name: '进入管理后台' }),
    ).toBeInTheDocument();
    expect(container.querySelector('.admin-login-card__icon')).toBeNull();
    expect(screen.getByRole('link', { name: '返回公开导航' })).toHaveAttribute(
      'href',
      '/',
    );
  });

  it('站点信息加载期间禁用登录输入和操作', () => {
    render(<AdminLoginPage loading />);

    expect(screen.getByLabelText(/管理员密码/)).toBeDisabled();
    expect(screen.getByRole('button', { name: '管理员登录' })).toBeDisabled();
  });
});
