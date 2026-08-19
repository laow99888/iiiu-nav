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
  const response = await fetch(force ? '/api/updates/check' : '/api/updates', {
    method: force ? 'POST' : 'GET',
    credentials: 'same-origin',
    headers: { Accept: 'application/json' },
    signal,
  });
  if (!response.ok) {
    throw new Error(`Update status request failed: ${response.status}`);
  }
  const value: unknown = await response.json();
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

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null;
}
