import { useState } from 'preact/hooks';

import { messages } from '../i18n/messages';
import type { NavigationCategory } from '../navigation/types';
import {
  LayoutGrid,
  ListOrdered,
  Monitor,
  type LucideIcon,
} from '../ui/icons/interface-icons';

type Period = 7 | 30;

const visitorSeries: Record<Period, readonly number[]> = {
  7: [42, 58, 51, 73, 64, 81, 76],
  30: [
    34, 42, 38, 55, 49, 61, 58, 47, 66, 72, 64, 79, 83, 68, 74, 91, 86, 77, 95,
    88, 102, 97, 84, 108, 114, 99, 121, 116, 128, 137,
  ],
};

export function AdminDashboard({
  categories,
}: {
  categories: readonly NavigationCategory[];
}) {
  const [period, setPeriod] = useState<Period>(7);
  const series = visitorSeries[period];
  const visitors = series.reduce((total, value) => total + value, 0);
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
          value={formatNumber(visitors)}
          badge={messages.admin.mockData}
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
            <span class="admin-section-kicker">{messages.admin.mockData}</span>
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
        <VisitorChart period={period} series={series} />
      </section>
    </div>
  );
}

function AdminMetric({
  badge,
  icon: Icon,
  label,
  testID,
  value,
}: {
  badge?: string;
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
      {badge ? <span class="admin-data-badge">{badge}</span> : null}
    </article>
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
  series: readonly number[];
}) {
  const maximum = Math.max(...series);
  return (
    <div
      class={`admin-visitor-chart admin-visitor-chart--${period}`}
      role="img"
      aria-label={messages.admin.visitorChartLabel(period)}
    >
      {series.map((value, index) => {
        const height = Math.max(8, Math.round((value / maximum) * 100));
        return (
          <div
            class="admin-visitor-chart__item"
            key={`${period}-${index}`}
            title={`第 ${index + 1} 天：${value}`}
          >
            <span class="admin-visitor-chart__track" aria-hidden="true">
              <span style={{ height: `${height}%` }} />
            </span>
            <span class="admin-visitor-chart__label">
              {chartLabel(period, index)}
            </span>
          </div>
        );
      })}
    </div>
  );
}

function chartLabel(period: Period, index: number) {
  if (period === 7) return `D${index + 1}`;
  return index === 0 || (index + 1) % 5 === 0 ? `${index + 1}` : '';
}

function formatNumber(value: number) {
  return new Intl.NumberFormat('zh-CN').format(value);
}
