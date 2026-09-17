import { isRecord, requestJSON } from '../api/client';

export type UpdateState =
  | 'development'
  | 'up_to_date'
  | 'update_available'
  | 'unavailable'
  | 'rate_limited'
  | 'invalid_release';

export type UpdateStatus = {
  automaticUpdate: boolean;
  checkedAt?: string;
  currentVersion: string;
  latestVersion?: string;
  manifestUrl?: string;
  publishedAt?: string;
  releaseUrl?: string;
  state: UpdateState;
};

export async function fetchUpdateStatus(
  force = false,
  signal?: AbortSignal,
): Promise<UpdateStatus> {
  const value: unknown = await requestJSON(
    force ? '/api/updates/check' : '/api/updates',
    { method: force ? 'POST' : 'GET', signal },
    'Update status request failed',
  );
  if (!isUpdateStatus(value)) {
    throw new Error('Update status response is invalid');
  }
  return value;
}

function isUpdateStatus(value: unknown): value is UpdateStatus {
  if (
    !isRecord(value) ||
    !isUpdateState(value.state) ||
    typeof value.currentVersion !== 'string' ||
    typeof value.automaticUpdate !== 'boolean' ||
    !optionalDate(value.checkedAt)
  ) {
    return false;
  }
  if (value.state === 'development') {
    return true;
  }
  if (value.state === 'up_to_date' || value.state === 'update_available') {
    return (
      stableVersion(value.latestVersion) &&
      optionalDate(value.publishedAt, true) &&
      officialURL(value.releaseUrl, '/releases/tag/') &&
      officialURL(value.manifestUrl, '/releases/download/')
    );
  }
  return true;
}

function isUpdateState(value: unknown): value is UpdateState {
  return (
    value === 'development' ||
    value === 'up_to_date' ||
    value === 'update_available' ||
    value === 'unavailable' ||
    value === 'rate_limited' ||
    value === 'invalid_release'
  );
}

function stableVersion(value: unknown): value is string {
  return (
    typeof value === 'string' &&
    /^v(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)$/.test(value)
  );
}

function optionalDate(value: unknown, required = false) {
  if (value === undefined) return !required;
  return typeof value === 'string' && !Number.isNaN(Date.parse(value));
}

function officialURL(value: unknown, marker: string): value is string {
  if (typeof value !== 'string') return false;
  try {
    const parsed = new URL(value);
    return (
      parsed.protocol === 'https:' &&
      parsed.hostname === 'github.com' &&
      parsed.pathname.startsWith(`/laow99888/iiiu-nav${marker}`)
    );
  } catch {
    return false;
  }
}

export type InstallState = 'idle' | 'running' | 'succeeded' | 'failed';

export type InstallPhase =
  | 'queued'
  | 'verifying_release'
  | 'pulling'
  | 'restarting'
  | 'verifying_health'
  | 'recovering';

export type InstallStatus = {
  message?: string;
  phase: InstallPhase;
  state: InstallState;
  targetVersion?: string;
};

// 一键更新需要复验当前密码；成功返回只代表执行器已接受请求，
// 进度须经 fetchInstallStatus 轮询观察（应用会短暂离线）。
export async function installUpdate(
  password: string,
  version: string,
): Promise<void> {
  await requestJSON(
    '/api/updates/install',
    { method: 'POST', body: JSON.stringify({ password, version }) },
    'Update install request failed',
  );
}

export async function fetchInstallStatus(): Promise<InstallStatus> {
  const value: unknown = await requestJSON(
    '/api/updates/install/status',
    { method: 'GET' },
    'Update install status request failed',
  );
  if (
    !isRecord(value) ||
    !isInstallState(value.state) ||
    !isInstallPhase(value.phase)
  ) {
    throw new Error('Update install status response is invalid');
  }
  return {
    state: value.state,
    phase: value.phase,
    targetVersion:
      typeof value.targetVersion === 'string' ? value.targetVersion : undefined,
    message: typeof value.message === 'string' ? value.message : undefined,
  };
}

function isInstallState(value: unknown): value is InstallState {
  return (
    value === 'idle' ||
    value === 'running' ||
    value === 'succeeded' ||
    value === 'failed'
  );
}

function isInstallPhase(value: unknown): value is InstallPhase {
  return (
    value === 'queued' ||
    value === 'verifying_release' ||
    value === 'pulling' ||
    value === 'restarting' ||
    value === 'verifying_health' ||
    value === 'recovering'
  );
}
