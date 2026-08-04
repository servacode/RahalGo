/**
 * صفحةُ قسم — **أصنافُه من كلّ المصادر مختلطةً.**
 *
 * الزبونُ يفتح «شاورما» فيرى شاورما، **لا قائمةَ مطاعم تبيع شاورما.** وأيُّ
 * صنفٍ اختاره نعرف نحن من أين نشتريه.
 */

import Link from "next/link";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import ItemCard, { type BrowseItem } from "@/components/ItemCard";

const m = getMessages(defaultLocale);
const API = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

export const dynamic = "force-dynamic";

export default async function SectionPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  let items: BrowseItem[] = [];
  let title = "";
  try {
    const res = await fetch(`${API}/api/v1/public/sections/${id}/items`, { cache: "no-store" });
    const json = (await res.json()) as { data?: { items?: BrowseItem[] } };
    items = json.data?.items ?? [];
    title = items[0]?.section_name ?? "";
  } catch {
    /* الخادم غير متاح — تُعرض صفحة فارغة بدل الانهيار */
  }

  return (
    <div>
      <Link href="/" className="mb-3 inline-block text-sm text-ink-muted hover:text-primary-dark">
        ← {m.site.backHome}
      </Link>
      <h1 className="mb-4 text-2xl font-bold">{title || m.site.sections.title}</h1>

      {items.length === 0 ? (
        <p className="rounded-card border border-line bg-surface p-10 text-center text-ink-muted">
          {m.site.sections.empty}
        </p>
      ) : (
        /* **بعددِ أعمدةِ الأقسام نفسِه** — من فتح قسماً لا يجد الشبكةَ تغيّرت
           تحته. (قرارُ المالك ٢٠٢٦-٠٨-٠٤: «شكلُ العرض للأصناف يجب أن يكون
           موحّداً».) */
        <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-5">
          {items.map((it) => (
            <ItemCard key={it.id} item={it} />
          ))}
        </div>
      )}
    </div>
  );
}
