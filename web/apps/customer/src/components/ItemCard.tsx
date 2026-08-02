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

import Link from "next/link";
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
  const img = mediaUrl(item.image_thumb_url);
  const off = !item.available || item.source_closed;

  return (
    <Link
      href={`/i/${item.id}`}
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
    </Link>
  );
}
