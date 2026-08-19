import { messages } from '../i18n/messages';
import { SiteIdentity } from '../navigation/site-identity';

const navigationRows = Array.from({ length: 5 });
const tableRows = Array.from({ length: 5 });
const tableColumns = Array.from({ length: 7 });

export function AdminLoadingPage() {
  return (
    <div class="admin-shell admin-shell--loading" aria-busy="true">
      <aside class="admin-sidebar" aria-hidden="true">
        <div class="admin-sidebar__brand">
          <SiteIdentity name="iiiu-nav" />
          <span>{messages.admin.badge}</span>
        </div>
        <div class="admin-loading-nav">
          {navigationRows.map((_, index) => (
            <span class="ui-skeleton admin-loading-nav__row" key={index} />
          ))}
        </div>
      </aside>

      <main class="admin-main">
        <header class="admin-mobile-header" aria-hidden="true">
          <SiteIdentity name="iiiu-nav" compact />
        </header>
        <div class="admin-content">
          <header
            class="admin-heading admin-heading--loading"
            aria-hidden="true"
          >
            <div>
              <span class="ui-skeleton admin-loading-heading__eyebrow" />
              <span class="ui-skeleton admin-loading-heading__title" />
              <span class="ui-skeleton admin-loading-heading__summary" />
            </div>
          </header>

          <section
            class="admin-table-panel admin-table-skeleton"
            role="status"
            aria-label={messages.admin.loadingWorkspace}
          >
            <span class="sr-only">{messages.admin.loadingWorkspace}</span>
            <div class="admin-table-toolbar" aria-hidden="true">
              <span class="ui-skeleton admin-table-skeleton__search" />
              <span class="ui-skeleton admin-table-skeleton__filter" />
              <span class="ui-skeleton admin-table-skeleton__count" />
            </div>
            <div class="admin-table-scroll" aria-hidden="true">
              <div class="admin-table-skeleton__grid">
                <div class="admin-table-skeleton__header">
                  {tableColumns.map((_, index) => (
                    <span class="ui-skeleton" key={index} />
                  ))}
                </div>
                {tableRows.map((_, rowIndex) => (
                  <div class="admin-table-skeleton__row" key={rowIndex}>
                    {tableColumns.map((_, columnIndex) => (
                      <span class="ui-skeleton" key={columnIndex} />
                    ))}
                  </div>
                ))}
              </div>
            </div>
          </section>
        </div>
      </main>
    </div>
  );
}
