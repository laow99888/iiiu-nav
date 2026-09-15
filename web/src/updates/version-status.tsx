import { useCallback, useEffect, useRef, useState } from 'preact/hooks';

import { messages } from '../i18n/messages';
import { ArrowUpRight, RefreshCw } from '../ui/icons/interface-icons';
import { Button } from '../ui/primitives';
import { fetchUpdateStatus, type UpdateStatus } from './update-api';

type ViewState =
  | { kind: 'loading' }
  | { kind: 'error' }
  | { kind: 'ready'; status: UpdateStatus };

export function VersionStatus() {
  const [view, setView] = useState<ViewState>({ kind: 'loading' });
  const [refreshing, setRefreshing] = useState(false);
  // Guards against a slow cached response landing after a forced re-check and
  // overwriting fresher state.
  const requestRef = useRef(0);

  const load = useCallback(async (force: boolean, signal?: AbortSignal) => {
    const requestId = ++requestRef.current;
    if (force) setRefreshing(true);
    try {
      const status = await fetchUpdateStatus(force, signal);
      if (requestId !== requestRef.current) return;
      setView({ kind: 'ready', status });
    } catch (error) {
      if (error instanceof DOMException && error.name === 'AbortError') return;
      if (requestId !== requestRef.current) return;
      setView({ kind: 'error' });
    } finally {
      if (force && requestId === requestRef.current) setRefreshing(false);
    }
  }, []);

  useEffect(() => {
    const controller = new AbortController();
    void load(false, controller.signal);
    return () => controller.abort();
  }, [load]);

  const presentation = versionPresentation(view);
  const status = view.kind === 'ready' ? view.status : null;
  const updateAvailable = status?.state === 'update_available';

  return (
    <article class="admin-workspace-row admin-version-row">
      <span
        class="admin-workspace-row__icon admin-workspace-row__icon--neutral"
        aria-hidden="true"
      >
        <RefreshCw />
      </span>
      <div class="admin-workspace-row__content" aria-live="polite">
        <h3>{messages.updates.title}</h3>
        <p>{presentation.summary}</p>
        <div class="admin-workspace-row__details">
          {presentation.details.map((detail) => (
            <span key={detail}>{detail}</span>
          ))}
          {status?.releaseUrl ? (
            <a
              class="admin-version__release"
              href={status.releaseUrl}
              target="_blank"
              rel="noopener noreferrer"
            >
              {messages.updates.releaseNotes}
              <ArrowUpRight aria-hidden="true" />
            </a>
          ) : null}
        </div>
        {updateAvailable && !status.automaticUpdate ? (
          <div class="admin-version__manual">
            <strong>{messages.updates.manualUpdate}</strong>
            <code>{messages.updates.manualCommand}</code>
          </div>
        ) : null}
      </div>
      <Button
        icon={RefreshCw}
        loading={refreshing}
        onClick={() => void load(true)}
      >
        {messages.updates.checkNow}
      </Button>
    </article>
  );
}

function versionPresentation(view: ViewState) {
  if (view.kind === 'loading') {
    return {
      summary: messages.updates.loading,
      details: [messages.updates.stableChannel],
    };
  }
  if (view.kind === 'error') {
    return {
      summary: messages.updates.unavailable,
      details: [messages.updates.retryDescription],
    };
  }

  const { status } = view;
  switch (status.state) {
    case 'development':
      return {
        summary: messages.updates.development(status.currentVersion),
        details: [messages.updates.developmentDescription],
      };
    case 'up_to_date':
      return {
        summary: messages.updates.upToDate,
        details: [messages.updates.current(status.currentVersion)],
      };
    case 'update_available':
      return {
        summary: messages.updates.available(status.latestVersion ?? ''),
        details: [
          messages.updates.current(status.currentVersion),
          status.publishedAt
            ? messages.updates.published(formatDate(status.publishedAt))
            : messages.updates.stableChannel,
          messages.updates.executorUnavailable,
        ],
      };
    case 'rate_limited':
      return {
        summary: messages.updates.rateLimited,
        details: [messages.updates.retryDescription],
      };
    case 'invalid_release':
      return {
        summary: messages.updates.invalidRelease,
        details: [messages.updates.current(status.currentVersion)],
      };
    default:
      return {
        summary: messages.updates.unavailable,
        details: [messages.updates.retryDescription],
      };
  }
}

function formatDate(value: string) {
  return new Intl.DateTimeFormat('zh-CN', {
    dateStyle: 'medium',
  }).format(new Date(value));
}
