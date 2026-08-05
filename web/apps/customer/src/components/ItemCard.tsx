"use client";

/**
 * بطاقةُ صنفٍ في التصفّح — **بلا مصدرها.**
 *
 * الزبونُ يشتري «من رحّال» لا «من مطعم فلان»: **الاسمُ والصورةُ والسعرُ وحدَها،
 * ولا شيءَ يدلّ على المطبخ.**
 *
 * **وسببُ الغياب يُقال بموعده**: «متاح من ١٠ صباحاً» موعدٌ يُعاد إليه، و«غير
 * متاح» طريقٌ مسدود.
 *
 * # وشكلُها شكلُ القسم — والصنفِ في اللوحة
 *
 * كانت **سطراً أفقيّاً بمصغَّرةٍ ٦٤ بكسل** بينما القسمُ فوقها بطاقةٌ بصورةٍ
 * تملأ عرضَها. **فيُقرأ القسمُ سلعةً والصنفُ سطرَ جدول** — وهو مقلوب: الصنفُ
 * هو ما يُشترى.
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-٠٤: «شكلُ العرض بالموقع للأصناف يجب أن يكون موحّداً».)
 *
 * **والشكلُ الواحدُ ليس ذوقاً**: من يمسح السوقَ بالعين ثمّ يفتح قسماً **لا
 * يُعيد تعلُّمَ أين يقع الاسمُ وأين السعرُ وأين الحال**. وشاشةُ اللوحة تعرض
 * البطاقةَ نفسَها، **فمراجعةُ ما يراه الزبونُ لا تصير تخميناً.**
 */

import { useCallback, useState } from "react";
import { Modal, LoadingState } from "@rahalgo/ui";
import ItemClient, { type Group } from "@/app/i/[id]/ItemClient";
import { api } from "@/lib/api";
import { getMessages, defaultLocale, fmtNum, fmtTime } from "@rahalgo/i18n";
import { Badge } from "@rahalgo/ui";
import { mediaUrl } from "@/lib/api";

const m = getMessages(defaultLocale);

export interface BrowseItem {
  id: string;
  name: string;
  description: string;
  price: number;
  /**
   * **الأصلُ للبطاقة، والمصغَّرةُ بديلُها.**
   *
   * المصغَّرةُ حدُّها ٤٠٠ بكسل، **وبطاقةٌ تمطّها تبهت.** والمصغَّرةُ تبقى لما
   * رُفع قبل هذا التغيير ولنافذةِ الصنف.
   */
  image_url?: string | null;
  image_thumb_url: string | null;
  available: boolean;
  source_closed: boolean;
  source_opens_at: string | null;
  section_id: string;
  section_name: string;
}

export default function ItemCard({ item }: { item: BrowseItem }) {
  /**
   * **الصنفُ يُفتح في نافذةٍ لا في صفحة.**
   *
   * الزبونُ في قسمٍ يتصفّح عشرةَ أصناف: يفتح واحداً، يقرأ خياراتِه، **يرجع
   * ليفتح غيرَه** — وكلُّ رجعةٍ تعيده إلى أعلى القائمة فيبحث عن موضعه.
   * **والنافذةُ تُبقيه حيث هو.**
   *
   * (قرارُ المالك ٢٠٢٦-٠٨-٠٣: «بعد أن يدخل إلى قسم معيّن ويضغط على نوع،
   * هنا تكون نافذة منبثقة».)
   */
  const [openItem, setOpenItem] = useState<{ item: BrowseItem; modifiers: Group[] } | null>(null);
  const [loading, setLoading] = useState(false);

  const open = useCallback(async () => {
    // **الخياراتُ تُجلب عند الفتح** — بطاقةُ التصفّح لا تحملها، وجلبُها لكلّ
    // بطاقةٍ في القائمة عشرةُ نداءاتٍ لينظر في واحد.
    setLoading(true);
    setOpenItem(null);
    try {
      const d = await api<{ item?: BrowseItem; modifiers?: Group[] }>(
        `/api/v1/public/items/${item.id}`,
      );
      if (d.item) setOpenItem({ item: d.item, modifiers: d.modifiers ?? [] });
    } catch {
      /* تعذّر الجلب — تُغلق النافذة ولا تُفتح فارغة */
    } finally {
      setLoading(false);
    }
  }, [item.id]);

  const img = mediaUrl(item.image_url ?? item.image_thumb_url);
  const off = !item.available || item.source_closed;

  return (
    <>
      <button
        type="button"
        onClick={open}
        className={`flex flex-col overflow-hidden rounded-card border border-line bg-surface text-start transition-shadow hover:elev-2 ${
          off ? "opacity-60" : ""
        }`}
      >
        {/* **الصورةُ أوّلاً وتملأ العرض** — كبطاقة القسم فوقها تماماً. */}
        <span className="relative flex aspect-[4/3] items-center justify-center bg-page">
          {img ? (
            // eslint-disable-next-line @next/next/no-img-element
            <img src={img} alt="" loading="lazy" className="h-full w-full object-cover" />
          ) : (
            /* **وحرفُ الاسم لا رمزٌ رماديّ** — الرمزُ الواحدُ لعشرة أصنافٍ
               يجعلها شيئاً واحداً، **والحرفُ يفرّق بينها ويبقى لها.** */
            <span className="text-3xl font-bold text-primary-dark">{item.name.charAt(0)}</span>
          )}
          {/* **«نفد» و«نائم» خبران مختلفان** — الأوّلُ لا موعدَ له والثاني له
              موعد. **وموضعُهما فوق الصورة** كشارة القسم: تُقرأ قبل الاسم. */}
          {!item.available ? (
            <span className="absolute end-1.5 top-1.5">
              <Badge variant="warning">{m.site.menu.unavailable}</Badge>
            </span>
          ) : item.source_closed ? (
            <span className="absolute end-1.5 top-1.5">
              <Badge variant="neutral">
                {item.source_opens_at
                  ? m.site.menu.availableFrom.replace("{t}", fmtTime(item.source_opens_at))
                  : m.site.menu.unavailable}
              </Badge>
            </span>
          ) : null}
        </span>

        {/* **الاسمُ يميناً والسعرُ يساراً** — كالقسم: اسمُه يميناً وعددُه يساراً. */}
        <span className="flex min-w-0 flex-1 flex-col gap-1 p-3">
          <span className="truncate font-bold">{item.name}</span>
          <span className="flex items-baseline justify-between gap-2">
            <span className="truncate text-xs text-ink-muted">{item.description}</span>
            <span className="shrink-0 font-bold text-primary-dark">
              {fmtNum(item.price)} {m.common.currency}
            </span>
          </span>
        </span>
      </button>

      <Modal
        open={loading || !!openItem}
        onClose={() => setOpenItem(null)}
        title={openItem?.item.name ?? item.name}
        size="lg"
      >
        {loading || !openItem ? (
          <LoadingState />
        ) : (
          <ItemClient item={openItem.item} modifiers={openItem.modifiers} />
        )}
      </Modal>
    </>
  );
}
