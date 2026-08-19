import { render, screen } from '@testing-library/preact';
import { describe, expect, it } from 'vitest';

import { AdminLoadingPage } from './admin-loading-page';

describe('后台加载页', () => {
  it('使用与管理表格一致的七列骨架保持布局稳定', () => {
    const { container } = render(<AdminLoadingPage />);

    expect(
      screen.getByRole('status', { name: '正在载入管理后台' }),
    ).toBeInTheDocument();
    expect(container.querySelector('.admin-shell')).toHaveAttribute(
      'aria-busy',
      'true',
    );
    expect(
      container.querySelectorAll('.admin-table-skeleton__header > span'),
    ).toHaveLength(7);
    expect(
      container.querySelectorAll('.admin-table-skeleton__row'),
    ).toHaveLength(5);
  });
});
