/**
 * عميل الـAPI الموحد للوحة — يضيف التوكن تلقائياً ويجدده عند انتهاء صلاحيته.
 * صيغة الخادم الموحدة: النجاح {data} والخطأ {error:{code,message_key}}.
 */

const API_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

/** يحوّل مسار وسائط نسبياً من الخادم (/media/...) إلى رابط كامل */
export function mediaUrl(path: string | null | undefined): string | null {
  if (!path) return null;
  return path.startsWith("http") ? path : `${API_URL}${path}`;
}

export interface ApiErrorBody {
  code: string;
  message_key: string;
  details?: Record<string, unknown>;
}

export class ApiError extends Error {
  constructor(
    public status: number,
    public body: ApiErrorBody,
  ) {
    super(body.code);
  }
}

export interface AuthUser {
  id: string;
  phone: string;
  full_name: string;
  status: string;
  has_password: boolean;
  invite_code: string | null;
  roles: string[];
  created_at: string;
}

export interface TokenPair {
  access_token: string;
  access_expires_at: string;
  refresh_token: string;
}

export interface AuthResult {
  user: AuthUser;
  tokens: TokenPair;
}

const ACCESS_KEY = "rahalgo_access";
const REFRESH_KEY = "rahalgo_refresh";

export const tokenStore = {
  get access() {
    return typeof window === "undefined" ? null : localStorage.getItem(ACCESS_KEY);
  },
  get refresh() {
    return typeof window === "undefined" ? null : localStorage.getItem(REFRESH_KEY);
  },
  set(tokens: TokenPair) {
    localStorage.setItem(ACCESS_KEY, tokens.access_token);
    localStorage.setItem(REFRESH_KEY, tokens.refresh_token);
  },
  clear() {
    localStorage.removeItem(ACCESS_KEY);
    localStorage.removeItem(REFRESH_KEY);
  },
};

async function rawRequest<T>(path: string, init: RequestInit = {}, token?: string | null): Promise<T> {
  const headers = new Headers(init.headers);
  // FormData يضبط ترويسته بنفسه (حد الأجزاء multipart)
  if (!(init.body instanceof FormData)) headers.set("Content-Type", "application/json");
  if (token) headers.set("Authorization", `Bearer ${token}`);

  const res = await fetch(`${API_URL}${path}`, { ...init, headers });
  const json = (await res.json().catch(() => null)) as { data?: T; error?: ApiErrorBody } | null;

  if (!res.ok || !json || json.error) {
    throw new ApiError(res.status, json?.error ?? { code: "internal", message_key: "errors.internal" });
  }
  return json.data as T;
}

let refreshing: Promise<void> | null = null;

async function refreshTokens(): Promise<void> {
  const refresh = tokenStore.refresh;
  if (!refresh) throw new ApiError(401, { code: "unauthorized", message_key: "errors.unauthorized" });
  const result = await rawRequest<AuthResult>("/api/v1/auth/refresh", {
    method: "POST",
    body: JSON.stringify({ refresh_token: refresh }),
  });
  tokenStore.set(result.tokens);
}

/** طلب مصادَق — يجدد التوكن مرة واحدة تلقائياً عند 401 ثم يعيد المحاولة */
export async function api<T>(path: string, init: RequestInit = {}): Promise<T> {
  try {
    return await rawRequest<T>(path, init, tokenStore.access);
  } catch (err) {
    if (err instanceof ApiError && err.status === 401 && tokenStore.refresh) {
      refreshing ??= refreshTokens().finally(() => {
        refreshing = null;
      });
      await refreshing;
      return rawRequest<T>(path, init, tokenStore.access);
    }
    throw err;
  }
}

export const authApi = {
  loginPassword: (phone: string, password: string) =>
    rawRequest<AuthResult>("/api/v1/auth/login", {
      method: "POST",
      body: JSON.stringify({ phone, password }),
    }),
  requestOtp: (phone: string) =>
    rawRequest<{ sent: boolean }>("/api/v1/auth/otp/request", {
      method: "POST",
      body: JSON.stringify({ phone }),
    }),
  verifyOtp: (phone: string, code: string) =>
    rawRequest<AuthResult>("/api/v1/auth/otp/verify", {
      method: "POST",
      body: JSON.stringify({ phone, code }),
    }),
  me: () => api<AuthUser>("/api/v1/auth/me"),
  logout: () => {
    const refresh = tokenStore.refresh;
    tokenStore.clear();
    if (refresh) {
      void rawRequest("/api/v1/auth/logout", {
        method: "POST",
        body: JSON.stringify({ refresh_token: refresh }),
      }).catch(() => undefined);
    }
  },
};
