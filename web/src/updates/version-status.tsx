import { useCallback, useEffect, useRef, useState } from 'preact/hooks';

import { ApiError } from '../api/client';
import { messages } from '../i18n/messages';
import {
  useAsyncResource,
  type AsyncResourceState,
} from '../ui/hooks/use-async-resource';
import { ArrowUpRight, RefreshCw } from '../ui/icons/interface-icons';
import { Button, TextField } from '../ui/primitives';
import {
  fetchInstallStatus,
  fetchUpdateStatus,
  installUpdate,
  type InstallPhase,
  type UpdateStatus,
} from './update-api';

type ViewState = AsyncResourceState<UpdateStatus>;

// 一键更新是独立于版本检查的本地状态机：confirm 输入密码，running 轮询
// 执行器进度（应用重启期间请求会失败，属预期），succeeded 刷新页面。
type Install =
  | { kind: 'idle' }
  | { kind: 'confirm' }
  | { kind: 'running'; text: string; waiting: boolean }
  | { kind: 'succeeded' }
  | { kind: 'failed'; message: string };

export function VersionStatus() {
  const [refreshing, setRefreshing] = useState(false);
  // 手动“检查更新”走强制端点（POST /api/updates/check），其余加载读缓存。
  // loader 引用保持稳定，强制标志经 ref 传递给下一次加载。
  const forceRef = useRef(false);
  const load = useCallback((signal: AbortSignal) => {
    const force = forceRef.current;
    forceRef.current = false;
    return fetchUpdateStatus(force, signal);
  }, []);
  const resource = useAsyncResource(load);
  const view = resource.state;

  const [install, setInstall] = useState<Install>({ kind: 'idle' });
  const [password, setPassword] = useState('');
  const [installError, setInstallError] = useState<string | null>(null);

  const checkNow = () => {
    forceRef.current = true;
    setRefreshing(true);
    resource.reload();
  };
  // reload 结束（无论成败）都会产生新的 state 对象，用它在此时撤下按钮加载视觉。
  useEffect(() => {
    setRefreshing(false);
  }, [view]);

  const status = view.kind === 'ready' ? view.value : null;
  const updateAvailable = status?.state === 'update_available';
  const installable = updateAvailable && status.automaticUpdate === true;
  const busy =
    install.kind === 'running' ||
    install.kind === 'succeeded' ||
    install.kind === 'failed';

  // 轮询执行器进度；应用被重建期间请求会失败，此时保持轮询并提示等待。
  useEffect(() => {
    if (install.kind !== 'running') return undefined;
    let alive = true;
    const tick = async () => {
      try {
        const progress = await fetchInstallStatus();
        if (!alive) return;
        if (progress.state === 'succeeded') {
          setInstall({ kind: 'succeeded' });
          return;
        }
        if (progress.state === 'failed') {
          setInstall({
            kind: 'failed',
            message: progress.message || messages.updates.installFailed,
          });
          return;
        }
        if (progress.state === 'running') {
          setInstall({
            kind: 'running',
            text: phaseText(progress.phase),
            waiting: false,
          });
        }
      } catch {
        if (alive)
          setInstall((current) =>
            current.kind === 'running'
              ? { ...current, waiting: true }
              : current,
          );
      }
    };
    void tick();
    const timer = window.setInterval(() => void tick(), 2000);
    return () => {
      alive = false;
      window.clearInterval(timer);
    };
  }, [install.kind]);

  // 更新成功后短暂提示再刷新；卸载（测试、导航）会取消这次刷新。
  useEffect(() => {
    if (install.kind !== 'succeeded') return undefined;
    const timer = window.setTimeout(() => {
      window.location.reload();
    }, 2500);
    return () => window.clearTimeout(timer);
  }, [install.kind]);

  const startInstall = async () => {
    const version = status?.latestVersion;
    if (!version) return;
    try {
      await installUpdate(password, version);
      setPassword('');
      setInstallError(null);
      setInstall({
        kind: 'running',
        text: messages.updates.phaseQueued,
        waiting: false,
      });
    } catch (error) {
      if (
        error instanceof ApiError &&
        error.code === 'update_password_invalid'
      ) {
        setInstallError(messages.updates.invalidPassword);
        return;
      }
      if (
        error instanceof ApiError &&
        error.code === 'update_already_running'
      ) {
        setInstallError(messages.updates.alreadyRunning);
        return;
      }
      setInstallError(messages.updates.installStartFailed);
    }
  };

  const presentation = versionPresentation(
    view,
    status?.automaticUpdate === true,
  );

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
        {install.kind === 'confirm' ? (
          <form
            class="admin-version__install"
            onSubmit={(event) => {
              event.preventDefault();
              void startInstall();
            }}
          >
            <p>{messages.updates.installDescription}</p>
            <TextField
              error={installError ?? undefined}
              label={messages.updates.installPassword}
              onChange={(event) => setPassword(event.currentTarget.value)}
              required
              type="password"
              value={password}
            />
            <div class="admin-version__install-actions">
              <Button type="submit">{messages.updates.installStart}</Button>
              <Button
                onClick={() => {
                  setInstall({ kind: 'idle' });
                  setInstallError(null);
                  setPassword('');
                }}
                type="button"
                variant="secondary"
              >
                {messages.updates.cancel}
              </Button>
            </div>
          </form>
        ) : null}
        {install.kind === 'running' ? (
          <div class="admin-version__install" role="status">
            <p>
              <strong>{messages.updates.installRunning}</strong>
              {install.waiting
                ? ` ${messages.updates.installWaiting}`
                : ` ${install.text}`}
            </p>
          </div>
        ) : null}
        {install.kind === 'succeeded' ? (
          <div class="admin-version__install" role="status">
            <p>{messages.updates.installSucceeded}</p>
          </div>
        ) : null}
        {install.kind === 'failed' ? (
          <div
            class="admin-version__install admin-version__install--failed"
            role="alert"
          >
            <p>
              <strong>{messages.updates.installFailed}</strong>
              {install.message ? ` ${install.message}` : ''}
            </p>
            <div class="admin-version__install-actions">
              <Button
                onClick={() => {
                  setInstall({ kind: 'idle' });
                  setInstallError(null);
                }}
                type="button"
                variant="secondary"
              >
                {messages.updates.cancel}
              </Button>
            </div>
          </div>
        ) : null}
      </div>
      {busy ? null : (
        <div class="admin-version__actions">
          {installable ? (
            <Button
              onClick={() => {
                setInstallError(null);
                setInstall({ kind: 'confirm' });
              }}
              type="button"
            >
              {messages.updates.installTo(status?.latestVersion ?? '')}
            </Button>
          ) : null}
          <Button icon={RefreshCw} loading={refreshing} onClick={checkNow}>
            {messages.updates.checkNow}
          </Button>
        </div>
      )}
    </article>
  );
}

function phaseText(phase: InstallPhase) {
  switch (phase) {
    case 'queued':
      return messages.updates.phaseQueued;
    case 'verifying_release':
      return messages.updates.phaseVerify;
    case 'pulling':
      return messages.updates.phasePull;
    case 'restarting':
      return messages.updates.phaseRestart;
    case 'recovering':
      return messages.updates.phaseRecover;
    default:
      return messages.updates.phaseHealth;
  }
}

function versionPresentation(view: ViewState, automaticUpdate: boolean) {
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

  const status = view.value;
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
          automaticUpdate
            ? messages.updates.executorAvailable
            : messages.updates.executorUnavailable,
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
