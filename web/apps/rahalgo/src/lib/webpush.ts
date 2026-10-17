"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **إشعاراتُ المتصفّح — الزبونُ والمتجرُ على الويب** (قرارُ المالك ٢٠٢٦-١٠-١٧)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **عاملُ خدمةٍ واحد (`/sw.js`) بتسجيلين**: نطاقُ `/` للزبون ونطاقُ
 * `/store/` للمتجر — **فلكلٍّ اشتراكُه**، وصاحبُ متجرٍ يتسوّق لا يضيع
 * اشتراكُ متجره حين يفعّل إشعاراتِ طلباته كزبون.
 *
 * **والإذنُ يُطلب من ضغطةٍ لا عند الفتح** — المتصفّحاتُ ترفض غيرَه،
 * **وآيفون لا يسمح أصلاً إلّا من أيقونة الشاشة الرئيسيّة وبنظام ١٦٫٤ فأحدث.**
 */

import { api } from "@/lib/api";

export type WebPushApp = "customer" | "merchant";

/**
 * **حالُ الجهاز** — بالترتيب الذي يُعرَض:
 *
 *	unsupported  متصفّحٌ بلا إشعارات
 *	ios-home     آيفون من سفاري — يُضاف للشاشة الرئيسيّة أوّلاً
 *	ios-old      آيفون أقدمُ من ١٦٫٤
 *	denied       منعها صاحبُها — تُفتح من الإعدادات لا من هنا
 *	off          ممكنةٌ ولم تُفعَّل
 *	on           مفعّلةٌ ومحفوظةٌ عند المحرّك
 */
export type WebPushState = "unsupported" | "ios-home" | "ios-old" | "denied" | "off" | "on";

const SCOPE: Record<WebPushApp, string> = { customer: "/", merchant: "/store/" };

type NavIOS = Navigator & { standalone?: boolean };

function iosVersion(): number | null {
  const nav = navigator as NavIOS;
  const ua = nav.userAgent || "";
  const ios = /iPhone|iPad|iPod/i.test(ua) || (nav.platform === "MacIntel" && nav.maxTouchPoints > 1);
  if (!ios) return null;
  const m = ua.match(/OS (\d+)[_.](\d+)/) ?? ua.match(/Version\/(\d+)\.(\d+)/);
  if (!m) return 0;
  return Number(m[1]) + Number(m[2]) / 100;
}

function standalone(): boolean {
  const nav = navigator as NavIOS;
  return nav.standalone === true || window.matchMedia?.("(display-mode: standalone)").matches === true;
}

/** **ما يسمح به الجهاز** — قبل أيّ نداء. */
export function webPushSupport(): Exclude<WebPushState, "denied" | "off" | "on"> | "ok" {
  if (typeof window === "undefined") return "unsupported";
  const ios = iosVersion();
  if (ios !== null) {
    if (ios > 0 && ios < 16.04) return "ios-old";
    if (!standalone()) return "ios-home";
  }
  if (!("serviceWorker" in navigator) || !("PushManager" in window) || !("Notification" in window)) {
    return ios !== null ? "ios-home" : "unsupported";
  }
  return "ok";
}

function keyBytes(b64: string): Uint8Array<ArrayBuffer> {
  const pad = "=".repeat((4 - (b64.length % 4)) % 4);
  const raw = atob((b64 + pad).replace(/-/g, "+").replace(/_/g, "/"));
  const out = new Uint8Array(new ArrayBuffer(raw.length));
  for (let i = 0; i < raw.length; i++) out[i] = raw.charCodeAt(i);
  return out;
}

function sameKey(sub: PushSubscription, key: Uint8Array): boolean {
  const cur = sub.options?.applicationServerKey;
  if (!cur) return true;
  const a = new Uint8Array(cur);
  return a.length === key.length && a.every((v, i) => v === key[i]);
}

async function registration(app: WebPushApp): Promise<ServiceWorkerRegistration> {
  const existing = await navigator.serviceWorker.getRegistration(SCOPE[app]);
  if (existing && new URL(existing.scope).pathname === SCOPE[app]) return existing;
  return navigator.serviceWorker.register("/sw.js", { scope: SCOPE[app] });
}

async function existingSub(app: WebPushApp): Promise<PushSubscription | null> {
  const reg = await navigator.serviceWorker.getRegistration(SCOPE[app]);
  if (!reg || new URL(reg.scope).pathname !== SCOPE[app]) return null;
  return reg.pushManager.getSubscription();
}

async function save(app: WebPushApp, sub: PushSubscription): Promise<void> {
  const json = sub.toJSON();
  await api("/api/v1/me/webpush", {
    method: "POST",
    body: JSON.stringify({ endpoint: json.endpoint, keys: json.keys, app }),
  });
}

/** **الحالُ الآن** — ومن كان مشتركاً يُعاد حفظُه عند المحرّك (مرّةً في الجلسة). */
export async function webPushState(app: WebPushApp): Promise<WebPushState> {
  const support = webPushSupport();
  if (support !== "ok") return support;
  if (Notification.permission === "denied") return "denied";
  if (Notification.permission !== "granted") return "off";
  try {
    const sub = await existingSub(app);
    if (!sub) return "off";
    const once = `rahalgo_webpush_sync_${app}`;
    if (!sessionStorage.getItem(once)) {
      await save(app, sub);
      sessionStorage.setItem(once, "1");
    }
    return "on";
  } catch {
    return "off";
  }
}

/**
 * **يفعّل الإشعارات — يُنادى من ضغطةٍ وحدَها.**
 *
 * **والإذنُ أوّلُ ما يُطلب** قبل أيّ نداء شبكة: سفاري يرفض طلبَه إن لم يكن
 * في الضغطة نفسِها. **ويردّ الخطأَ كما قاله المحرّك** (`webpush_disabled`).
 */
export async function enableWebPush(app: WebPushApp): Promise<WebPushState> {
  const support = webPushSupport();
  if (support !== "ok") return support;
  const perm = await Notification.requestPermission();
  if (perm === "denied") return "denied";
  if (perm !== "granted") return "off";
  const { enabled, public_key } = await api<{ enabled: boolean; public_key: string }>(
    "/api/v1/public/webpush-key",
  );
  if (!enabled || !public_key) throw new WebPushDisabledError();
  const key = keyBytes(public_key);
  const reg = await registration(app);
  await navigator.serviceWorker.ready;
  let sub = await reg.pushManager.getSubscription();
  // **ومفتاحٌ تبدّل على الخادم** يُبطل الاشتراكَ القديم — فيُجدَّد.
  if (sub && !sameKey(sub, key)) {
    await sub.unsubscribe();
    sub = null;
  }
  sub ??= await reg.pushManager.subscribe({ userVisibleOnly: true, applicationServerKey: key });
  await save(app, sub);
  try {
    sessionStorage.setItem(`rahalgo_webpush_sync_${app}`, "1");
  } catch {
    /* تفضيلٌ لا يُحفظ لا يُسقط شيئاً */
  }
  return "on";
}

/** **يوقف إشعاراتِ هذا المتصفّح** — عند المحرّك ثمّ في المتصفّح. */
export async function disableWebPush(app: WebPushApp): Promise<void> {
  const sub = await existingSub(app);
  if (!sub) return;
  await api("/api/v1/me/webpush", {
    method: "DELETE",
    body: JSON.stringify({ endpoint: sub.endpoint }),
  }).catch(() => undefined);
  await sub.unsubscribe().catch(() => false);
}

/**
 * **عند الخروج** — يُلغى الاشتراكُ في المتصفّح.
 *
 * **والمحرّكُ يحذف صفَّه بنفسه مع الجلسة** (`D12`)، فلا يلزم نداءٌ هنا —
 * **ومن دخل بعده على الجهاز نفسِه لا تصله إشعاراتُ غيره.**
 */
export async function dropWebPushOnLogout(): Promise<void> {
  if (typeof window === "undefined" || !("serviceWorker" in navigator)) return;
  try {
    const regs = await navigator.serviceWorker.getRegistrations();
    await Promise.all(
      regs.map(async (r) => {
        const sub = await r.pushManager?.getSubscription();
        await sub?.unsubscribe();
      }),
    );
    sessionStorage.removeItem("rahalgo_webpush_sync_customer");
    sessionStorage.removeItem("rahalgo_webpush_sync_merchant");
  } catch {
    /* لا يُسقط الخروج */
  }
}

export class WebPushDisabledError extends Error {
  constructor() {
    super("webpush_disabled");
  }
}
