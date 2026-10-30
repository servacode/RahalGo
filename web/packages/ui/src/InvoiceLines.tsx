/**
 * ══════════════════════════════════════════════════════════════════════
 * **الفاتورةُ سطرٌ لكلّ عنصر — بشكلٍ واحدٍ في كلّ الشاشات** (قرارُ المالك ٢٠٢٦-١٠-٣٠)
 * ══════════════════════════════════════════════════════════════════════
 *
 * «شاورما دبل × 3 ........ 90 ل.س» · «بندورة × 1 كيلو ........ 600 ل.س» ·
 * «بوط أسود × 1 نمرة 40 ........ 9,100 ل.س»: **النصُّ في البداية ومجموعُ السطر
 * في النهاية** — لا سطرَ خياراتٍ ولا «للواحد» ولا عمودَ عدد. **ولا يُكتب اسمُ
 * مجموعة الخيار** («وزن ثقيل»، «الحجم»). **والمشروبُ (`drink_for`) سطرٌ بعد صنفه.**
 *
 * **والقاعدةُ نفسُها في تطبيقات أندرويد** (`mobile/ui/.../Invoice.kt`، `invoiceLineText`).
 */

import { Money } from "./money";
import { invoiceLineText, invoiceOrder, type InvoiceLineItem } from "./invoiceText";

export { invoiceLineText, invoiceOrder, type InvoiceLineItem };

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
        <li key={it.id || i}>
          <div className="flex items-baseline justify-between gap-3">
            <span className="min-w-0 break-words font-medium">
              {invoiceLineText(it.name, it.qty, it.options)}
            </span>
            <Money value={it.unit_price * it.qty} small className="shrink-0 tabular-nums" />
          </div>
          {it.note && <p className="text-xs text-ink-muted">{it.note}</p>}
        </li>
      ))}
    </ul>
  );
}
