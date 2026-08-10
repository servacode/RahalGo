/**
 * صفحةُ قسم — **أصنافُه من كلّ المصادر مختلطةً.**
 *
 * الزبونُ يفتح «شاورما» فيرى شاورما، **لا قائمةَ مطاعم تبيع شاورما.** وأيُّ
 * صنفٍ اختاره نعرف نحن من أين نشتريه.
 */

import Link from "next/link";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { Alert, EmptyState, IconStore } from "@rahalgo/ui";
import ItemGrid from "@/components/ItemGrid";
import { type BrowseItem } from "@/components/ItemCard";

const m = getMessages(defaultLocale);
const API = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

export const dynamic = "force-dynamic";

export default async function SectionPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  let items: BrowseItem[] = [];
  let title = "";
  /**
   * **«لم أصل» غيرُ «وصلتُ فلم أجد».**
   *
   * كانت الصفحةُ تقول «لا أصنافَ في هذا القسم» **في الحالين** — فانقطاعُ
   * الشبكة يُقرأ قسماً فارغاً، **فيظنّ الزبونُ أنّ المنصةَ خاوية** ولا يعود.
   *
   * **وهي العائلةُ نفسُها التي وقعت في الرئيسيّة** (٢٠٢٦-٠٨-٠٣) فذهب المالكُ
   * يُعيد تشغيل المشروع كلَّه والخادمُ يردّ مئتين. **وأُصلحت هناك ونُسيت هنا.**
   */
  let reached = false;
  try {
    const res = await fetch(`${API}/api/v1/public/sections/${id}/items`, { cache: "no-store" });
    reached = res.ok;
    const json = (await res.json()) as { data?: { items?: BrowseItem[] } };
    items = json.data?.items ?? [];
    title = items[0]?.section_name ?? "";
  } catch {
    /* الخادم غير متاح — **وهذه وحدَها رسالةُ الانقطاع** */
  }

  return (
    <div>
      <Link href="/" className="mb-3 inline-block text-sm text-ink-muted hover:text-primary-dark">
        ← {m.site.backHome}
      </Link>
      <h1 className="heading-page mb-4">{title || m.site.sections.title}</h1>

      {!reached ? (
        <Alert tone="warning" title={m.errors.offline}>
          {m.errors.offlineHint}
        </Alert>
      ) : items.length === 0 ? (
        <EmptyState icon={<IconStore size={28} />} title={m.site.sections.empty} />
      ) : (
        /* **بعددِ أعمدةِ الأقسام نفسِه** — من فتح قسماً لا يجد الشبكةَ تغيّرت
           تحته. (قرارُ المالك ٢٠٢٦-٠٨-٠٤: «شكلُ العرض للأصناف يجب أن يكون
           موحّداً».) */
        <ItemGrid items={items} next={`/s/${id}`} />
      )}
    </div>
  );
}
