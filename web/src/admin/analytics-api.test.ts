import { afterEach, describe, expect, it, vi } from 'vitest';

import { fetchPageViews } from './analytics-api';

describe('访客统计 API', () => {
  afterEach(() => vi.unstubAllGlobals());

  it('读取并校验 30 天页面浏览序列', async () => {
    const series = Array.from({ length: 30 }, (_, index) => ({
      date: `2026-08-${String(index + 1).padStart(2, '0')}`,
      views: index,
    }));
    const fetchMock = vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ series }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      }),
    );
    vi.stubGlobal('fetch', fetchMock);

    await expect(fetchPageViews()).resolves.toEqual(series);
    expect(fetchMock).toHaveBeenCalledWith(
      '/api/analytics/page-views',
      expect.objectContaining({ credentials: 'same-origin' }),
    );
  });

  it('拒绝非整数和缺失日期的无效响应', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue(
        new Response(
          JSON.stringify({
            series: Array.from({ length: 30 }, () => ({
              date: '',
              views: 0.5,
            })),
          }),
          { status: 200 },
        ),
      ),
    );

    await expect(fetchPageViews()).rejects.toThrow(
      'Analytics response is invalid',
    );
  });
});
