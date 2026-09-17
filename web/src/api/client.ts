// 统一的同源 API 客户端：集中处理 fetch 凭据与请求头、错误响应解析、
// 错误类型与文件下载锚点，供各 *-api.ts 适配器复用，避免逐处手写。

/** 判定未知值为普通对象，供各领域的响应校验器复用。 */
export function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null;
}

/** 携带 HTTP 状态码与服务端错误码的 API 错误。 */
export class ApiError extends Error {
  readonly status: number;
  readonly code: string | null;
  readonly retryAfter: number | null;

  constructor(
    message: string,
    options: {
      code?: string | null;
      retryAfter?: number | null;
      status: number;
    },
  ) {
    super(message);
    this.status = options.status;
    this.code = options.code ?? null;
    this.retryAfter = options.retryAfter ?? null;
  }
}

/** 读取错误响应体里的 { error: "..." } 错误码；响应不是 JSON 时返回 null。 */
export async function readErrorCode(
  response: Response,
): Promise<string | null> {
  try {
    const value: unknown = await response.json();
    return isRecord(value) && typeof value.error === 'string'
      ? value.error
      : null;
  } catch {
    return null;
  }
}

/** 发起同源 JSON API 请求：统一携带 Cookie 与 Accept 头；JSON 请求体自动补 Content-Type。 */
export async function request(
  path: string,
  init?: RequestInit,
): Promise<Response> {
  const body = init?.body;
  const headers: Record<string, string> = {
    Accept: 'application/json',
    ...(init?.headers as Record<string, string> | undefined),
  };
  if (body && !(body instanceof FormData)) {
    headers['Content-Type'] = 'application/json';
  }
  return fetch(path, {
    ...init,
    credentials: 'same-origin',
    headers,
  });
}

/** request 的 JSON 便捷封装：失败时抛出带状态码与错误码的 ApiError，成功时返回解析后的未知值。 */
export async function requestJSON(
  path: string,
  init?: RequestInit,
  errorPrefix?: string,
): Promise<unknown> {
  const response = await request(path, init);
  if (!response.ok) {
    const code = await readErrorCode(response);
    throw new ApiError(
      errorPrefix
        ? `${errorPrefix}: ${response.status}`
        : `Request failed: ${response.status}`,
      { code, status: response.status },
    );
  }
  return response.json();
}

/** 通过临时锚点触发浏览器下载；url 可为对象 URL，由调用方负责释放。 */
export function downloadFile(url: string, filename: string) {
  const anchor = document.createElement('a');
  anchor.href = url;
  anchor.download = filename;
  anchor.click();
}
