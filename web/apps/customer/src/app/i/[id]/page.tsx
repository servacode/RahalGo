/** صفحةُ الصنف — تُقدَّم من الخادم كي يكون رابطُها قابلاً للمشاركة والفهرسة. */

import { notFound } from "next/navigation";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { Alert } from "@rahalgo/ui";
import ItemClient, { type Group } from "./ItemClient";
import type { BrowseItem } from "@/components/ItemCard";

const API = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

export const dynamic = "force-dynamic";

const m = getMessages(defaultLocale);

export default async function ItemPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  /**
   * **«لم أصل» غيرُ «غير موجود» — وهذه أسوأُ صياغاتها.**
   *
   * كان انقطاعُ الشبكة يستدعي `notFound()` — **فتقول الشاشةُ للزبون إنّ
   * الصنفَ غيرُ موجود** وهو في القائمة. **فيظنّ أنّ المتجرَ حذفه** ويبحث عن
   * غيره، ولا يعود إليه.
   *
   * **والفرقُ في العاقبة**: «لا إنترنت» يُعاد المحاولةُ بعدها، **و«غير
   * موجود» بابٌ مسدود.**
   *
   * **و`notFound()` كان داخل `try`** — وهو يرمي، **فيلتقطه `catch` ويناديه
   * ثانيةً**: صنفٌ محذوفٌ حقّاً يمرّ بالمسار نفسِه الذي يمرّ به الانقطاع،
   * **فلا يُفرَّق بينهما أصلاً.**
   */
  let data: { item?: BrowseItem; modifiers?: Group[] } | undefined;
  let reached = false;
  try {
    const res = await fetch(`${API}/api/v1/public/items/${id}`, { cache: "no-store" });
    reached = res.ok;
    data = ((await res.json()) as { data?: typeof data }).data;
  } catch {
    /* الخادمُ غير متاح — **وهذه وحدَها رسالةُ الانقطاع** */
  }

  // **والصنفُ المحذوفُ حقّاً وحدَه يُقال عنه «غير موجود».**
  if (reached && !data?.item) notFound();
  if (!data?.item) {
    return (
      <div className="py-10">
        <Alert tone="warning" title={m.errors.offline}>
          {m.errors.offlineHint}
        </Alert>
      </div>
    );
  }
  return <ItemClient item={data.item} modifiers={data.modifiers ?? []} />;
}
