export type DailyPageViews = {
  date: string;
  views: number;
};

export async function fetchPageViews(
  signal?: AbortSignal,
): Promise<DailyPageViews[]> {
  const response = await fetch('/api/analytics/page-views', {
    credentials: 'same-origin',
    headers: { Accept: 'application/json' },
    signal,
  });
  if (!response.ok) {
    throw new Error(`Analytics request failed: ${response.status}`);
  }
  const value: unknown = await response.json();
  if (!isPageViewResponse(value)) {
    throw new Error('Analytics response is invalid');
  }
  return value.series;
}

function isPageViewResponse(
  value: unknown,
): value is { series: DailyPageViews[] } {
  return (
    typeof value === 'object' &&
    value !== null &&
    'series' in value &&
    Array.isArray(value.series) &&
    value.series.length === 30 &&
    value.series.every(
      (item) =>
        typeof item === 'object' &&
        item !== null &&
        'date' in item &&
        typeof item.date === 'string' &&
        /^\d{4}-\d{2}-\d{2}$/.test(item.date) &&
        'views' in item &&
        Number.isInteger(item.views) &&
        item.views >= 0,
    )
  );
}
