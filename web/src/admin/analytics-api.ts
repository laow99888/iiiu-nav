import { isRecord, requestJSON } from '../api/client';

export type DailyPageViews = {
  date: string;
  views: number;
};

export async function fetchPageViews(
  signal?: AbortSignal,
): Promise<DailyPageViews[]> {
  const value: unknown = await requestJSON(
    '/api/analytics/page-views',
    { signal },
    'Analytics request failed',
  );
  if (!isPageViewResponse(value)) {
    throw new Error('Analytics response is invalid');
  }
  return value.series;
}

function isPageViewResponse(
  value: unknown,
): value is { series: DailyPageViews[] } {
  return (
    isRecord(value) &&
    'series' in value &&
    Array.isArray(value.series) &&
    value.series.length === 30 &&
    value.series.every(
      (item) =>
        isRecord(item) &&
        'date' in item &&
        typeof item.date === 'string' &&
        /^\d{4}-\d{2}-\d{2}$/.test(item.date) &&
        'views' in item &&
        typeof item.views === 'number' &&
        Number.isInteger(item.views) &&
        item.views >= 0,
    )
  );
}
