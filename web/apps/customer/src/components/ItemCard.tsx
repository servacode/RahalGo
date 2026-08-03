"use client";

/**
 * بطاقةُ صنفٍ في التصفّح — **بلا مصدرها.**
 *
 * الزبونُ يشتري «من رحّال» لا «من مطعم فلان»: **الاسمُ والصورةُ والسعرُ وحدَها،
 * ولا شيءَ يدلّ على المطبخ.**
 *
 * **وسببُ الغياب يُقال بموعده**: «متاح من ١٠ صباحاً» موعدٌ يُعاد إليه، و«غير
 * متاح» طريقٌ مسدود.
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

  const img = mediaUrl(item.image_thumb_url);
  const off = !item.available || item.source_closed;

  return (
    <>
    <button
      type="button"
      onClick={open}
      className={`flex items-center gap-3 rounded-card border border-line bg-surface p-3 transition-shadow hover:shadow-md ${
        off ? "opacity-60" : ""
      }`}
    >
      {img ? (
        // eslint-disable-next-line @next/next/no-img-element
        <img src={img} alt="" className="h-16 w-16 rounded-control object-cover" />
      ) : (
        <span className="flex h-16 w-16 items-center justify-center rounded-control bg-primary-light text-lg font-bold text-primary-dark">
          {item.name.charAt(0)}
        </span>
      )}
      <div className="min-w-0 flex-1">
        <p className="truncate font-bold">{item.name}</p>
        {item.description && (
          <p className="line-clamp-1 text-xs text-ink-muted">{item.description}</p>
        )}
        <p className="mt-1 font-bold text-primary-dark">
          {fmtNum(item.price)} {m.common.currency}
        </p>
      </div>
      {/* **«نفد» و«نائم» خبران مختلفان** — الأوّلُ لا موعدَ له والثاني له موعد. */}
      {!item.available ? (
        <Badge variant="warning">{m.site.menu.unavailable}</Badge>
      ) : item.source_closed ? (
        <Badge variant="neutral">
          {item.source_opens_at
            ? m.site.menu.availableFrom.replace("{t}", fmtTime(item.source_opens_at))
            : m.site.menu.unavailable}
        </Badge>
      ) : null}
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
