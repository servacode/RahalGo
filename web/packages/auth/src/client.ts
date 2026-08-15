/**
 * عميل الـAPI المركزي — **نسخة واحدة** لكل التطبيقات (كان مكرّراً 4 مرات بنسبة 99%).
 * عميل الـAPI الموحد للوحة — يضيف التوكن تلقائياً ويجدده عند انتهاء صلاحيته.
 * صيغة الخادم الموحدة: النجاح {data} والخطأ {error:{code,message_key}}.
 */

const API_URL = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

/**
 * عميل الـAPI المركزي — **نسخة واحدة** لكل التطبيقات (كان مكرّراً 4 مرات بنسبة 99%). يحوّل مسار وسائط نسبياً من الخادم (/media/...) إلى رابط كامل */
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
  /** كلمة المرور وضعها طرف ثالث — تُجبر الواجهة على تبديلها قبل أي شاشة */
  must_change_password: boolean;
  avatar_thumb_url: string | null;
  last_seen_at: string | null;
  created_at: string;

  /**
   * **وأرقامُه كزبون — في جدول الحسابات لا في تبويبٍ ثانٍ.**
   *
   * (قرارُ المالك ٢٠٢٦-٠٨-١٥: «تبويبٌ منفصلٌ باسم الزبائن لا يلزم
   *  أساساً — كلُّ شيءٍ نريده موجودٌ بكلّ الحسابات».)
   *
   * **وتصل من نقطة الحسابات وحدَها** — فهي اختياريّةٌ في النوع:
   * **شاشةُ الدخول تقرأ `AuthUser` نفسَه ولا تعرف هذه.**
   */
  balance?: number;
  orders_count?: number;
  orders_spent?: number;
  last_order_at?: string | null;
  /** **وأرقامُه كمندوب** — ورمزُ دعوته في `invite_code` أعلاه. */
  rep_stores?: number;
  commissions?: number;
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

/**
 * خزنة التوكنات. "تذكّرني" ليست زينة: عند تفعيلها تُحفظ الجلسة في localStorage
 * فتبقى بعد إغلاق المتصفح، وعند تركها تُحفظ في sessionStorage فتموت مع اللسان —
 * وهذا ما يهمّ فعلاً على جهاز مشترك.
 */
export const tokenStore = {
  get access() {
    if (typeof window === "undefined") return null;
    return localStorage.getItem(ACCESS_KEY) ?? sessionStorage.getItem(ACCESS_KEY);
  },
  get refresh() {
    if (typeof window === "undefined") return null;
    return localStorage.getItem(REFRESH_KEY) ?? sessionStorage.getItem(REFRESH_KEY);
  },
  /** remember=false يقصر الجلسة على لسان المتصفح الحالي. */
  set(tokens: TokenPair, remember = true) {
    this.clear();
    const store = remember ? localStorage : sessionStorage;
    store.setItem(ACCESS_KEY, tokens.access_token);
    store.setItem(REFRESH_KEY, tokens.refresh_token);
  },
  clear() {
    for (const store of [localStorage, sessionStorage]) {
      store.removeItem(ACCESS_KEY);
      store.removeItem(REFRESH_KEY);
    }
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

/**
 * عميل الـAPI المركزي — **نسخة واحدة** لكل التطبيقات (كان مكرّراً 4 مرات بنسبة 99%). طلب مصادَق — يجدد التوكن مرة واحدة تلقائياً عند 401 ثم يعيد المحاولة */
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
  /** استعادة كلمة المرور: إرسال الرمز ثم ضبط كلمة جديدة (تفتح جلسة مباشرة). */
  requestReset: (phone: string) =>
    rawRequest<{ sent: boolean }>("/api/v1/auth/password/reset/request", {
      method: "POST",
      body: JSON.stringify({ phone }),
    }),
  confirmReset: (phone: string, code: string, password: string) =>
    rawRequest<AuthResult>("/api/v1/auth/password/reset/confirm", {
      method: "POST",
      body: JSON.stringify({ phone, code, password }),
    }),
  /** **يتحقّق من رمز الاستعادة ولا يستهلكه** — قبل نموذج الكلمة الجديدة. */
  verifyReset: (phone: string, code: string) =>
    rawRequest<{ verified: boolean }>("/api/v1/auth/password/reset/verify", {
      method: "POST",
      body: JSON.stringify({ phone, code }),
    }),
  /**
   * ══════════════════════════════════════════════════════════════════
   * **رمزُ الأدمن — الخطوةُ الثانية**
   * ══════════════════════════════════════════════════════════════════
   *
   * (قرارُ المالك ٢٠٢٦-٠٨-١٠.)
   *
   * **ولا رقمَ هاتفٍ في الطلب** — التحدّي يعرف صاحبَه. **ولو مُرّر الرقمُ
   * لَأمكن تخطّي الكلمة**: يرسل المهاجمُ رقمَ المالك ورمزاً يخمّنه.
   */
  verifyPin: (challenge: string, pin: string) =>
    rawRequest<AuthResult>("/api/v1/auth/pin", {
      method: "POST",
      body: JSON.stringify({ challenge, pin }),
    }),
  /** **أوّلُ ضبطٍ للرمز** — بالتحدّي نفسِه، ويُصدر الجلسة. */
  setupPin: (challenge: string, pin: string) =>
    rawRequest<AuthResult>("/api/v1/auth/pin/setup", {
      method: "POST",
      body: JSON.stringify({ challenge, pin }),
    }),

  /** إنشاء حساب زبون — لا يُنشئ أي دور آخر. */
  requestSignup: (phone: string) =>
    rawRequest<{ sent: boolean }>("/api/v1/auth/signup/request", {
      method: "POST",
      body: JSON.stringify({ phone }),
    }),
  /**
   * **يتحقّق من الرمز ولا يستهلكه** — خطوةٌ بين إرساله وبين ملء البيانات.
   *
   * (قرارُ المالك ٢٠٢٦-٠٨-٠٦: «لا تظهر المعلومات إلّا بعد التحقّق من الرمز».)
   */
  verifySignup: (phone: string, code: string) =>
    rawRequest<{ verified: boolean }>("/api/v1/auth/signup/verify", {
      method: "POST",
      body: JSON.stringify({ phone, code }),
    }),
  /** @param ref رمزُ من دعاه — **اختياريّ**، ومن سجّل بلا دعوةٍ حسابُه كامل. */
  confirmSignup: (
    phone: string,
    code: string,
    full_name: string,
    password: string,
    ref?: string,
  ) =>
    rawRequest<AuthResult>("/api/v1/auth/signup/confirm", {
      method: "POST",
      body: JSON.stringify({ phone, code, full_name, password, ref: ref ?? "" }),
    }),
  sso: (code: string) =>
    rawRequest<AuthResult>("/api/v1/auth/sso", {
      method: "POST",
      body: JSON.stringify({ code }),
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
