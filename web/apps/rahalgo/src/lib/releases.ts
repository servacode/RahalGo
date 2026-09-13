/**
 * ══════════════════════════════════════════════════════════════════════
 * **سجلُّ التوزيع كما يقرؤه الموقع** (`DLC`، ٢٠٢٦-٠٩-١٣)
 * ══════════════════════════════════════════════════════════════════════
 *
 * # ولا رابطَ في صفحة
 *
 * **والحالُ يحسبها المحرّك** (`internal/release`) — **وهذه تقرؤها.**
 * **وأربعُ صفحاتٍ تكتب كلٌّ منها رابطَها تفترق في الرابعة** — **وقد
 * وقع ذلك فعلاً**: **صفحةُ `/app` تحمل رابطَ متجرٍ مكتوباً في شيفرتها
 * و`platform.app_url` فارغٌ في الإنتاج.**
 *
 * # والفشلُ مغلقٌ
 *
 * **ومحرّكٌ لا يُجيب يعني «غيرُ متوفّرٍ بعد»** — **لا رابطاً قديماً ولا
 * زرّاً يعطي أربعمئةً وأربعة.** **وصفحةٌ تسقط بعطبٍ لأنّ المحرّكَ
 * تعثّر أسوأُ من صفحةٍ تقول «لم يُفتح بعد».**
 */

import { readServerConfig, serverApiBase } from "@/lib/config";

/** مفاتيحُ التطبيقات الأربعة — **مرآةُ `release.Apps` في المحرّك.** */
export const APP_KEYS = ["customer", "driver", "merchant", "rep"] as const;
export type AppKey = (typeof APP_KEYS)[number];

/** حالُ التوزيع — **مرآةُ `release.Status`.** */
export type ReleaseStatus =
  | "unavailable"
  | "play"
  | "direct"
  | "play_and_direct";

export interface Release {
  key: AppKey;
  package_id: string;
  channel: "play" | "direct";
  status: ReleaseStatus;
  play_url: string;
  download_url: string;
  version: string;
  size_bytes: number;
  sha256: string;
}

/** closed **حالٌ مغلقةٌ لتطبيقٍ** — حين لا يُقرأ السجلّ. */
function closed(key: AppKey): Release {
  return {
    key,
    package_id: "",
    channel: key === "customer" ? "play" : "direct",
    status: "unavailable",
    play_url: "",
    download_url: "",
    version: "",
    size_bytes: 0,
    sha256: "",
  };
}

/**
 * readReleases **يقرأ حالَ الأربعة من المحرّك.**
 *
 * **وبلا تخزين**: **الأثرُ الواحدُ يخدم بيئتين** — وهو نصُّ قاعدةِ
 * `sitemap` نفسِها. **وبمهلةٍ صريحةٍ** كي لا يورّث تعثّرُ المحرّك
 * عشرَ ثوانٍ لكلّ زائر.
 */
export async function readReleases(): Promise<Release[]> {
  const fallback = APP_KEYS.map(closed);
  try {
    const res = await fetch(`${serverApiBase()}/api/v1/public/releases`, {
      cache: "no-store",
      signal: AbortSignal.timeout(3000),
    });
    if (!res.ok) return fallback;
    const json = (await res.json()) as { data?: { apps?: Release[] } };
    const apps = json.data?.apps;
    if (!Array.isArray(apps) || apps.length === 0) return fallback;
    // **والترتيبُ من المحرّك** — **وصفحةٌ تعيد الترتيبَ تخترع أولويّةً.**
    return apps;
  } catch {
    return fallback;
  }
}

/** readRelease حالُ تطبيقٍ واحدٍ بمفتاحه. */
export async function readRelease(key: AppKey): Promise<Release> {
  const all = await readReleases();
  return all.find((a) => a.key === key) ?? closed(key);
}

/**
 * downloadHref **رابطُ التنزيل كاملاً** — **ومسارُ المحرّك لا مسارُ
 * الموقع.**
 *
 * **والمحرّكُ على أصلٍ آخر** (`api.rahalgo.com`) — **ورابطٌ نسبيٌّ في
 * صفحةٍ يقود إلى `rahalgo.com/api/...` وهي لا توجد.**
 */
export function downloadHref(r: Release): string {
  if (!r.download_url) return "";
  return `${readServerConfig().apiUrl}${r.download_url}`;
}

/** megabytes **حجمٌ يُقرأ** — ميغابايت بمنزلةٍ واحدة. */
export function megabytes(bytes: number): string {
  if (!bytes || bytes <= 0) return "";
  return (bytes / (1024 * 1024)).toFixed(1);
}
