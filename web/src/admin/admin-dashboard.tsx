import type { ComponentChildren } from 'preact';
import { useEffect, useState } from 'preact/hooks';

import { messages } from '../i18n/messages';
import type { NavigationCategory } from '../navigation/types';
import {
  LayoutGrid,
  ListOrdered,
  Monitor,
  RefreshCw,
  type LucideIcon,
} from '../ui/icons/interface-icons';
import { Button } from '../ui/primitives';
import { fetchPageViews, type DailyPageViews } from './analytics-api';

type Period = 7 | 30;

type AnalyticsState =
  | { status: 'loading' }
  | { status: 'error' }
  | { status: 'ready'; series: DailyPageViews[] };

export function AdminDashboard({
  categories,
}: {
  categories: readonly NavigationCategory[];
}) {
  const [period, setPeriod] = useState<Period>(7);
  const [analytics, setAnalytics] = useState<AnalyticsState>({
    status: 'loading',
  });
  const [requestVersion, setRequestVersion] = useState(0);
  useEffect(() => {
    const controller = new AbortController();
    setAnalytics({ status: 'loading' });
    void fetchPageViews(controller.signal)
      .then((series) => setAnalytics({ status: 'ready', series }))
      .catch((error: unknown) => {
        if (!(error instanceof DOMException && error.name === 'AbortError')) {
          setAnalytics({ status: 'error' });
        }
      });
    return () => controller.abort();
  }, [requestVersion]);
  const series =
    analytics.status === 'ready' ? analytics.series.slice(-period) : [];
  const visitors = series.reduce((total, item) => total + item.views, 0);
  const totalLinks = categories.reduce(
    (total, category) => total + category.links.length,
    0,
  );

  return (
    <div class="admin-dashboard">
      <section class="admin-metric-grid" aria-label={messages.admin.statistics}>
        <AdminMetric
          icon={Monitor}
          label={messages.admin.visitorTotal}
          value={
            analytics.status === 'ready' ? formatNumber(visitors) : '\u2014'
          }
          testID="visitor-total"
        />
        <AdminMetric
          icon={ListOrdered}
          label={messages.admin.linkMetric}
          value={formatNumber(totalLinks)}
        />
        <AdminMetric
          icon={LayoutGrid}
          label={messages.admin.categoryMetric}
          value={formatNumber(categories.length)}
        />
      </section>

      <section
        class="admin-analytics-panel"
        aria-labelledby="visitor-trend-title"
      >
        <header class="admin-analytics-panel__header">
          <div>
            <span class="admin-section-kicker">{messages.admin.dailyPV}</span>
            <h2 id="visitor-trend-title">{messages.admin.visitorTrend}</h2>
          </div>
          <div
            class="admin-period-control"
            role="group"
            aria-label={messages.admin.periodLabel}
          >
            <PeriodButton active={period === 7} onClick={() => setPeriod(7)}>
              {messages.admin.last7Days}
            </PeriodButton>
            <PeriodButton active={period === 30} onClick={() => setPeriod(30)}>
              {messages.admin.last30Days}
            </PeriodButton>
          </div>
        </header>
        {analytics.status === 'loading' ? (
          <AnalyticsState
            status="status"
            title={messages.admin.analyticsLoading}
            description={messages.admin.analyticsLoadingDescription}
          />
        ) : analytics.status === 'error' ? (
          <AnalyticsState
            status="alert"
            title={messages.admin.analyticsFailed}
            description={messages.admin.analyticsFailedDescription}
            action={
              <Button
                icon={RefreshCw}
                onClick={() => setRequestVersion((current) => current + 1)}
              >
                {messages.admin.analyticsRetry}
              </Button>
            }
          />
        ) : visitors === 0 ? (
          <AnalyticsState
            title={messages.admin.analyticsEmpty}
            description={messages.admin.analyticsEmptyDescription}
          />
        ) : (
          <VisitorChart period={period} series={series} />
        )}
      </section>
    </div>
  );
}

function AdminMetric({
  icon: Icon,
  label,
  testID,
  value,
}: {
  icon: LucideIcon;
  label: string;
  testID?: string;
  value: string;
}) {
  return (
    <article class="admin-metric">
      <span class="admin-metric__icon" aria-hidden="true">
        <Icon />
      </span>
      <div>
        <span class="admin-metric__label">{label}</span>
        <strong data-testid={testID}>{value}</strong>
      </div>
    </article>
  );
}

function AnalyticsState({
  action,
  description,
  status,
  title,
}: {
  action?: ComponentChildren;
  description: string;
  status?: 'alert' | 'status';
  title: string;
}) {
  return (
    <div class="admin-analytics-state" role={status}>
      <span aria-hidden="true">
        <Monitor />
      </span>
      <strong>{title}</strong>
      <p>{description}</p>
      {action}
    </div>
  );
}

function PeriodButton({
  active,
  children,
  onClick,
}: {
  active: boolean;
  children: string;
  onClick: () => void;
}) {
  return (
    <button
      class={active ? 'is-active' : undefined}
      aria-pressed={active}
      onClick={onClick}
    >
      {children}
    </button>
  );
}

function VisitorChart({
  period,
  series,
}: {
  period: Period;
  series: readonly DailyPageViews[];
}) {
  const maximum = Math.max(...series.map((item) => item.views));
  const tableId = `visitor-chart-table-${period}`;
  return (
    <>
      <div
        class="admin-visitor-chart-frame"
        role="img"
        aria-label={messages.admin.visitorChartLabel(period)}
        aria-describedby={tableId}
        tabIndex={0}
      >
        <div class={`admin-visitor-chart admin-visitor-chart--${period}`}>
          {series.map((item) => {
            const height =
              item.views === 0
                ? 0
                : Math.max(8, Math.round((item.views / maximum) * 100));
            return (
              <div
                class="admin-visitor-chart__item"
                key={item.date}
                title={messages.admin.dailyViews(
                  formatChartDate(item.date),
                  item.views,
                )}
              >
                <span class="admin-visitor-chart__value" aria-hidden="true">
                  {item.views}
                </span>
                <span class="admin-visitor-chart__track" aria-hidden="true">
                  <span
                    class={item.views === 0 ? 'is-zero' : undefined}
                    style={{ height: `${height}%` }}
                  />
                </span>
                {period === 7 ? (
                  <span class="admin-visitor-chart__label">
                    {formatChartDate(item.date)}
                  </span>
                ) : null}
              </div>
            );
          })}
        </div>
        {period === 30 ? <ChartAxis series={series} /> : null}
      </div>
      <table class="sr-only" id={tableId}>
        <caption>{messages.admin.visitorChartLabel(period)}</caption>
        <thead>
          <tr>
            <th scope="col">{messages.admin.date}</th>
            <th scope="col">{messages.admin.dailyPV}</th>
          </tr>
        </thead>
        <tbody>
          {series.map((item) => (
            <tr key={item.date}>
              <th scope="row">{formatChartDate(item.date)}</th>
              <td>{item.views}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </>
  );
}

function ChartAxis({ series }: { series: readonly DailyPageViews[] }) {
  const middle = series[Math.floor(series.length / 2)];
  return (
    <div class="admin-visitor-chart-axis" aria-hidden="true">
      <span>{formatChartDate(series[0]?.date ?? '')}</span>
      <span>{formatChartDate(middle?.date ?? '')}</span>
      <span>{formatChartDate(series.at(-1)?.date ?? '')}</span>
    </div>
  );
}

function formatChartDate(date: string) {
  const [, month, day] = date.split('-');
  if (!month || !day) return '';
  return `${Number(month)}/${Number(day)}`;
}

function formatNumber(value: number) {
  return new Intl.NumberFormat('zh-CN').format(value);
}
