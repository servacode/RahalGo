/**
 * ══════════════════════════════════════════════════════════════════════
 * **الفاتورةُ سطرٌ لكلّ عنصر — بشكلٍ واحدٍ في كلّ الشاشات** (قرارُ المالك ٢٠٢٦-١٠-٣٠)
 * ══════════════════════════════════════════════════════════════════════
 *
 * «شاورما دبل × 3 ........ 90 ل.س» · «بندورة × 1 كيلو ........ 600 ل.س» ·
 * «بوط أسود × 1 نمرة 40 ........ 9,100 ل.س»: **النصُّ في البداية ومجموعُ السطر
 * في النهاية** — لا سطرَ خياراتٍ ولا «للواحد» ولا عمودَ عدد. **ولا يُكتب اسمُ
 * مجموعة الخيار** («وزن ثقيل»، «الحجم»). **والأصنافُ أوّلاً ثمّ المشروباتُ (`drink_for`) كلُّها تحتها، والمشروبُ المكرّرُ سطرٌ واحد** (`invoiceOrder`).
 *
 * **والقاعدةُ نفسُها في تطبيقات أندرويد** (`mobile/ui/.../Invoice.kt`، `invoiceLineText`).
 */

"use client";

import { useState } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { ImageViewer } from "./ImageViewer";
import { Money } from "./money";
import { invoiceLineText, invoiceOptionsLabel, invoiceOrder, type InvoiceLineItem } from "./invoiceText";
import { apiBase } from "./runtimeconfig";

const m = getMessages(defaultLocale);

export { invoiceLineText, invoiceOptionsLabel, invoiceOrder, type InvoiceLineItem };

/** **عنوانٌ كاملٌ لصورةٍ يرسلها المحرّكُ نسبيّةً** (`/media/...`) — كما يفعل `mediaUrl` في الحزمة المقابلة. */
function lineMedia(path: string | null | undefined): string | null {
  if (!path) return null;
  return path.startsWith("http") ? path : `${apiBase()}${path}`;
}

/**
 * **مربّعُ صورة الصنف على سطر الطلب** (قرارُ المالك ٢٠٢٦-١١-١٢) — مصغّرةٌ صغيرة،
 * **ولمسةٌ عليها تفتحها بملء الشاشة** (`ImageViewer`). واحدٌ للزبون والمتجر والمكتب.
 */
export function OrderLineThumb({ image, thumb, name }: { image?: string | null; thumb?: string | null; name: string }) {
  const [open, setOpen] = useState(false);
  const full = lineMedia(image ?? thumb);
  const small = lineMedia(thumb ?? image);
  if (!full || !small) return null;
  return (
    <>
      <button
        type="button"
        onClick={() => setOpen(true)}
        aria-label={`${m.common.itemPhotoOpen} · ${name}`}
        data-testid="order-line-photo"
        className="size-12 shrink-0 overflow-hidden rounded-rg-sm bg-field"
      >
        {/* eslint-disable-next-line @next/next/no-img-element */}
        <img src={small} alt={name} loading="lazy" className="size-full object-cover" />
      </button>
      {open && <ImageViewer src={full} alt={name} onClose={() => setOpen(false)} />}
    </>
  );
}

/**
 * **أسطرُ الأصناف** — كلٌّ في سطرٍ: النصُّ في البداية ومجموعُه (سعرُ الوحدة × العدد)
 * في النهاية. **والمجاميعُ تحت الخطّ تبقى عند كلّ شاشةٍ** بما يحقّ لقارئها.
 */
export function InvoiceLines({
  items,
  className = "",
}: {
  items: InvoiceLineItem[];
  className?: string;
}) {
  return (
    <ul className={`space-y-1.5 text-sm ${className}`}>
      {invoiceOrder(items).map((it, i) => (
        <li key={it.id || i} className="flex items-start gap-2">
          <OrderLineThumb image={it.image_url} thumb={it.image_thumb_url} name={it.name} />
          <div className="min-w-0 flex-1">
          <div className="flex items-baseline justify-between gap-3">
            <span className={`min-w-0 break-words font-medium ${it.returned ? "text-ink-muted line-through" : ""}`}>
              {invoiceLineText(it.name, it.qty, it.options)}
            </span>
            <Money value={it.unit_price * it.qty} small className="shrink-0 tabular-nums" />
          </div>
          {/* **ما رجّعه الزبونُ بعد الاستلام يُقال بكلمته** (٢٠٢٦-١١-٠٣) — لا يُقرأ صنفاً مجّانيّاً. */}
          {it.returned && <p className="text-xs font-bold text-danger">{m.common.returnedByCustomer}</p>}
          {/* **وما اختاره بمجموعته** — «المقاس: 42 · اللون: أسود» (لا يغني عنه نصُّ السطر المختصر). */}
          {!it.drink_for && invoiceOptionsLabel(it.options) && (
            <p className="text-xs text-ink">{invoiceOptionsLabel(it.options)}</p>
          )}
          {it.note && <p className="text-xs text-ink-muted">{it.note}</p>}
          </div>
        </li>
      ))}
    </ul>
  );
}
