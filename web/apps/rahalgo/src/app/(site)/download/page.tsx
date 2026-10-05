/**
 * ══════════════════════════════════════════════════════════════════════
 * **‏/download — مركزُ التنزيل الرسميّ** (`DLC`، ٢٠٢٦-٠٩-١٣)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **وهو بابُ التوزيع الرسميُّ الوحيدُ في الويب** — أربعةُ تطبيقاتٍ
 * لأربعة أدوار.
 *
 * # ولا حسابَ يُشترَط
 *
 * **ومن يريد أن يصير زبوناً لا حسابَ له بعد** — **وبابُ تنزيلٍ يطلب
 * دخولاً بابٌ مغلق.**
 *
 * # ولا رابطَ مكتوبٌ هنا
 *
 * **الحالُ تُقرأ من `‎/api/v1/public/releases`** — **والمحرّكُ يحسبها
 * من إعداداتٍ وقرصٍ.** **وصفحةٌ تكتب رابطَها تَعِد بما لا تعرف**: وهو
 * ما وقع في `/app` — **رابطُ متجرٍ في الشيفرة وإعدادُ المتجر فارغ.**
 */

import type { Metadata } from "next";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { readReleases } from "@/lib/releases";
import { AppCard } from "@/components/download/AppCard";

const m = getMessages(defaultLocale);
const D = m.site.download;

export const dynamic = "force-dynamic";

export const metadata: Metadata = {
  title: D.metaTitle,
  description: D.metaDescription,
};

export default async function Page() {
  const apps = await readReleases();
  return (
    <main className="mx-auto max-w-4xl px-4 py-10">
      <h1 className="heading-page">{D.title}</h1>
      <p className="mt-2 text-muted">{D.lead}</p>

      {/* **وشبكةٌ تنطبق على عمودٍ في الجوّال** — والأكثرُ يفتحها منه. */}
      <div className="mx-auto mt-8 grid max-w-xl grid-cols-1 gap-4">
        {/* **والصفحةُ العامّةُ لتطبيق رحّال غو وحدَه** (قرارُ المالك ٢٠٢٦-١٠-٠٥: «ما
            بدّي أيّ شخص يتصفّح الموقع يشوف كلّ التطبيقات، ما بيهمّه غير تطبيق
            الزبون»). **وتطبيقاتُ الفريق بروابطها المباشرة** (`/download/driver` ·
            `/merchant` · `/rep`) — **وهي ما تحمله رسالةُ الترحيب لكلّ حساب.** */}
        {apps
          .filter((r) => r.key === "customer")
          .map((r) => (
            <AppCard key={r.key} release={r} />
          ))}
      </div>
    </main>
  );
}
