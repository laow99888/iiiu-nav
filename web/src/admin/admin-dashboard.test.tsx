import { render, screen, waitFor, within } from '@testing-library/preact';
import userEvent from '@testing-library/user-event';
import { afterEach, describe, expect, it, vi } from 'vitest';

import { AdminDashboard } from './admin-dashboard';

function analyticsResponse(views: number) {
  return new Response(
    JSON.stringify({
      series: Array.from({ length: 30 }, (_, index) => ({
        date: `2026-07-${String(index + 1).padStart(2, '0')}`,
        views,
      })),
    }),
    { status: 200, headers: { 'Content-Type': 'application/json' } },
  );
}

describe('真实访客统计仪表盘', () => {
  afterEach(() => vi.unstubAllGlobals());

  it('为零访问量显示明确空状态', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(analyticsResponse(0)));
    render(<AdminDashboard categories={[]} />);

    expect(screen.getByText('正在读取浏览数据')).toBeInTheDocument();
    expect(await screen.findByText('暂无页面浏览记录')).toBeInTheDocument();
    expect(screen.getByTestId('visitor-total')).toHaveTextContent('0');
  });

  it('读取失败后可以重试并恢复真实数据', async () => {
    const fetchMock = vi
      .fn()
      .mockRejectedValueOnce(new Error('offline'))
      .mockResolvedValueOnce(analyticsResponse(2));
    vi.stubGlobal('fetch', fetchMock);
    const user = userEvent.setup();
    render(<AdminDashboard categories={[]} />);

    await user.click(
      await screen.findByRole('button', { name: '重新载入统计' }),
    );
    await waitFor(() =>
      expect(screen.getByTestId('visitor-total')).toHaveTextContent('14'),
    );
    expect(fetchMock).toHaveBeenCalledTimes(2);
  });
});

describe('访客图表的无障碍数据', () => {
  afterEach(() => vi.unstubAllGlobals());

  it('为读屏提供逐日数据表，并在柱体上显示数值浮层', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(analyticsResponse(3)));
    render(<AdminDashboard categories={[]} />);

    const table = await screen.findByRole('table', {
      hidden: true,
    });
    expect(table).toBeInTheDocument();
    const rows = within(table).getAllByRole('row');
    // 默认展示近 7 天：表头 + 7 天数据行
    expect(rows.length).toBe(8);
    // 每行 = 本地化日期（rowheader）+ 逐日 PV 数值（cell）
    expect(within(rows[1]).getByRole('rowheader')).toBeInTheDocument();
    expect(within(rows[1]).getByText('3')).toBeInTheDocument();
    expect(
      document.querySelector('.admin-visitor-chart__value'),
    ).toBeInTheDocument();
  });
});
