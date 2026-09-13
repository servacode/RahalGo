import { apiBase } from "@rahalgo/ui";
/**
 * عميل الـAPI المركزي — **نسخة واحدة** لكل التطبيقات (كان مكرّراً 4 مرات بنسبة 99%).
 * عميل الـAPI الموحد للوحة — يضيف التوكن تلقائياً ويجدده عند انتهاء صلاحيته.
 * صيغة الخادم الموحدة: النجاح {data} والخطأ {error:{code,message_key}}.
 */

/**
 * ══════════════════════════════════════════════════════════════════════
 * **وعنوانُ المحرّك يُقرأ وقتَ التشغيل** (دورةُ ٧١و)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **وكان ثابتاً من `NEXT_PUBLIC_API_URL`** — **فيُخبَز في الحزمة**،
 * **فصورةُ التجهيز لا تعرف عنوانَ الإنتاج ولو كان الالتزامُ واحداً.**
 *
 * **ودالّةٌ لا ثابت**: **ثابتٌ على مستوى الملفّ يُحسَب عند التحميل** —
 * **وقد يسبق وصولَ `/config.js`.**
 *
 * **ولا ارتدادَ صامتاً** — **ومن ارتدّ إلى `localhost` أخفى عطباً في
 * النشر حتّى يراه زبون.**
 */
const API_URL = (): string => apiBase();

/**
 * عميل الـAPI المركزي — **نسخة واحدة** لكل التطبيقات (كان مكرّراً 4 مرات بنسبة 99%). يحوّل مسار وسائط نسبياً من الخادم (/media/...) إلى رابط كامل */
export function mediaUrl(path: string | null | undefined): string | null {
  if (!path) return null;
  return path.startsWith("http") ? path : `${API_URL()}${path}`;
}

export interface ApiErrorBody {
  code: string;
  message_key: string;
  details?: Record<string, unknown>;
}

export class ApiError extends Error {
  /** **ما يطلب الخادمُ تأكيدَه** — حاضرٌ في `step_up_required` وحدَه. */
  stepUp?: StepUpNeed;
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
  /** **وحالُه كسائق** — ورديّتُه ونقدُه وما في يده وما سلّم اليوم. */
  on_shift?: boolean;
  driver_cash?: number;
  open_orders?: number;
  delivered_today?: number;
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

/**
 * ══════════════════════════════════════════════════════════════════════
 * **وإشارةُ «بدّل كلمتَك» تُلتقَط مرّةً في المركز** (`WEBA`، ٢٠٢٦-٠٩-١٣)
 * ══════════════════════════════════════════════════════════════════════
 *
 * # لماذا لزمت
 *
 * **المحرّكُ يمنع من كلمتُه مؤقّتةٌ بـ`403 password_change_required`**
 * (`TMP`) — **وبوّابةُ التبديل في الويب تقرأ علَمَ `must_change_password`
 * من `‎/auth/me`.**
 *
 * **والعلَمُ يُقنَّع في الردّ حين يكون `security.force_password_change`
 * مُطفأً** (قرارُ المالك ٢٠٢٦-٠٨-٠٩، **وهو مُطفأٌ في الإنتاج**) — **فلا
 * تظهر البوّابةُ، ويرى صاحبُ الحساب منعاً بلا طريق.**
 *
 * # ولمَ إشارةٌ لا تحويلُ مسار
 *
 * **التحويلُ من داخل عميل الـAPI يسرق الملاحةَ من الصفحات** — **وطلبٌ
 * في الخلفيّة يقذف المستخدمَ من مكانه.** **والإشارةُ تُعلن الواقعةَ
 * وتتركُ العرضَ لمن يملكه** (`PasswordGate`).
 *
 * **ولا تُنقَض سياسةُ المحرّك**: **هي تقرأ ردَّه ولا تستنتج شيئاً** —
 * **والمنعُ يبقى في المحرّك ولو أخفت الواجهةُ بوّابتَها.**
 */
let passwordChangeRequired = false;
type PwListener = () => void;
const pwListeners = new Set<PwListener>();

function notePasswordChangeRequired(err: ApiError): void {
  if (err.status !== 403 || err.body.code !== "password_change_required") return;
  passwordChangeRequired = true;
  pwListeners.forEach((fn) => fn());
}

/** **أقال المحرّكُ «بدّلْ كلمتَك» في هذه الجلسة؟** */
export function isPasswordChangeRequired(): boolean {
  return passwordChangeRequired;
}

/** **يُنسى بعد التبديل** — فتُفتَح اللوحةُ بلا إعادة تحميل. */
export function clearPasswordChangeRequired(): void {
  passwordChangeRequired = false;
  pwListeners.forEach((fn) => fn());
}

/** يُشترَك ليُعاد الرسمُ عند ورود الإشارة — ويعيد فاسخَ الاشتراك. */
export function onPasswordChangeRequired(fn: PwListener): () => void {
  pwListeners.add(fn);
  return () => pwListeners.delete(fn);
}

async function rawRequest<T>(path: string, init: RequestInit = {}, token?: string | null): Promise<T> {
  const headers = new Headers(init.headers);
  // FormData يضبط ترويسته بنفسه (حد الأجزاء multipart)
  if (!(init.body instanceof FormData)) headers.set("Content-Type", "application/json");
  if (token) headers.set("Authorization", `Bearer ${token}`);

  const res = await fetch(`${API_URL()}${path}`, { ...init, headers });
  const json = (await res.json().catch(() => null)) as
    | { data?: T; error?: ApiErrorBody; step_up?: StepUpNeed }
    | null;

  if (!res.ok || !json || json.error) {
    const err = new ApiError(res.status, json?.error ?? { code: "internal", message_key: "errors.internal" });
    if (json?.step_up) err.stepUp = json.step_up;
    notePasswordChangeRequired(err);
    throw err;
  }
  return json.data as T;
}

/**
 * **ما يقوله الخادمُ حين يطلب تأكيداً** — `ADG-3`.
 *
 * **والنصُّ يُبنى من هذا لا من المسار** — **فاللوحةُ لا تخترع اسمَ
 * فعلٍ تعرضه على من يوشك أن يؤكّده.**
 */
export type StepUpNeed = { action: string; target_type: string; target_id: string };

/**
 * **من يسأل الكلمةَ حين يطلبها الخادم** — تسجّله اللوحةُ مرّةً.
 *
 * **ولا يوجد نموذجُ كلمةٍ في كلّ صفحة** — **نافذةٌ واحدةٌ تعرض الفعلَ
 * بنصّه ثمّ تسأل.**
 */
let stepUpAsker: ((need: StepUpNeed, req: StepUpRequest) => Promise<string | null>) | null = null;

/** طلبٌ يوشك أن يُنفَّذ — يُعرَض على من يؤكّد. */
export type StepUpRequest = { method: string; path: string; body: unknown };

export function setStepUpAsker(fn: typeof stepUpAsker) {
  stepUpAsker = fn;
}

/**
 * **يطلب إثباتَ تأكيدٍ لنداءٍ بعينه** — **ولا تُحفَظ الكلمةُ ولا تُعاد.**
 *
 * **والكلمةُ تُرسَل إلى بابِ التأكيد وحدَه** — **ولا تُرسَل ثانيةً مع
 * الفعل نفسِه**، ولا تُكتب في مخزنٍ ولا في حالةٍ تبقى.
 */
export async function requestStepUp(
  req: StepUpRequest,
  password: string,
): Promise<string> {
  const grant = await api<{ id: string }>("/api/v1/admin/step-up", {
    method: "POST",
    body: JSON.stringify({ method: req.method, path: req.path, body: req.body, password }),
  });
  return grant.id;
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
    // ══════════════════════════════════════════════════════════════
    // **وفعلٌ شديدٌ يُسأل عنه مرّةً ثمّ يُعاد** — `ADG-3`
    // ══════════════════════════════════════════════════════════════
    //
    // **والحدُّ في الخادم لا هنا** — **وهذا يسر استعمالٍ لا أمن**:
    // نداءٌ مباشرٌ بلا إثباتٍ يُردّ سواءٌ مرّ من هنا أو لم يمرّ.
    if (
      err instanceof ApiError &&
      err.status === 403 &&
      err.body.code === "step_up_required" &&
      err.stepUp &&
      stepUpAsker &&
      !init.headers
    ) {
      const grant = await stepUpAsker(err.stepUp, {
        method: (init.method ?? "GET").toUpperCase(),
        path,
        body: init.body ? JSON.parse(String(init.body)) : undefined,
      });
      // **وإلغاءُ التأكيد يُبقي الخطأ كما هو** — ولا يُبتلَع صامتاً.
      if (!grant) throw err;
      return rawRequest<T>(path, { ...init, headers: { "X-Step-Up": grant } }, tokenStore.access);
    }
    throw err;
  }
}

/**
 * **تنزيلُ ملفٍّ مصادَقٍ — بالتجديد نفسِه.**
 *
 * (كشفه فحصُ المالك ٢٠٢٦-٠٨-١٦ في التقارير.)
 *
 * # المسألة
 *
 * **`api` تفكّ ظرفَ JSON وتردّ كائناً** — ولا تصلح لملفٍّ نصّيّ. **فكُتب
 * في شاشة التقارير `fetch` خامٌّ بيده** — **وتجاوز التجديدَ عند ٤٠١.**
 *
 * **فالنتيجةُ حالةٌ تقع كثيراً**: تُفتح الشاشةُ وتُقرأ الأرقامُ دقائق،
 * ثمّ يُضغط «تصدير» **فيردّ «حدث خطأ ما» والصفحةُ حولك تعمل** — لأنّ
 * نداءاتِها جدّدت التوكنَ وهذا لم يفعل. **فيُظنّ التصديرُ معطّلاً وهو
 * معطّلٌ بانتهاء توكن.**
 *
 * # ولماذا هنا لا هناك
 *
 * **التجديدُ منطقٌ واحدٌ في موضعٍ واحد** — ونسخةٌ ثانيةٌ منه في شاشةٍ
 * تفترق يوماً: **يُضاف حارسٌ في إحداهما ويُنسى في الأخرى.**
 *
 * **ويردّ الاستجابةَ خاماً** — فمن أراد ملفّاً أخذ `blob`، ومن أراد نصّاً
 * أخذ `text`. **ولا يُفترض شكلٌ على من يطلب.**
 */
export async function apiFile(path: string, init: RequestInit = {}): Promise<Response> {
  const call = (token?: string | null) =>
    fetch(`${API_URL()}${path}`, {
      ...init,
      headers: {
        ...(init.headers ?? {}),
        ...(token ? { Authorization: `Bearer ${token}` } : {}),
      },
    });
  let res = await call(tokenStore.access);
  // **ومحاولةٌ واحدةٌ بعد التجديد** — **وحلقةٌ تعيد بلا حدٍّ تخنق الخادمَ
  // بتوكنٍ ميّت.**
  if (res.status === 401 && tokenStore.refresh) {
    refreshing ??= refreshTokens().finally(() => {
      refreshing = null;
    });
    await refreshing;
    res = await call(tokenStore.access);
  }
  if (!res.ok) {
    // **وسببُ الخادم يُقرأ ويُرمى إلى من يستطيع ترجمتَه** — **ورسالةٌ
    // واحدةٌ لكلّ العلل تُسكت ما يُفيد.**
    let body: ApiErrorBody | undefined;
    try {
      body = ((await res.json()) as { error?: ApiErrorBody }).error;
    } catch {
      body = undefined;
    }
    const err = new ApiError(res.status,
      body ?? { code: "internal", message_key: "errors.internal" });
    notePasswordChangeRequired(err);
    throw err;
  }
  return res;
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
  /**
   * **قدراتُ صاحب الجلسة** — لِبابِ القائمة لا للتخويل.
   *
   * **والحدُّ في المحرّك** (جدولُ السياسة): من نادى باباً لا يملكه رُدّ
   * **ولو أظهرت الواجهةُ زرَّه.** **وهذا يمنع العكس**: بابٌ يملكه
   * صاحبُه ولا يراه لأنّ اسمَ دورِه ليس مكتوباً في الواجهة.
   */
  capabilities: () =>
    api<{ capabilities: string[] }>("/api/v1/auth/capabilities").then(
      (r) => r.capabilities ?? [],
    ),
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
