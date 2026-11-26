"use client";

/**
 * **مستلموه السابقون تحت الحقل** (قرارُ المالك ٢٠٢٦-١١-٠٥) — مرآةُ `RecipientSuggestions` في
 * `DeliveryScreen.kt`: الاسمُ ثمّ «الرقم — العنوان»، **وعلامةُ موقعٍ إن عُرفت نقطتُه**، و✕
 * «شيلو من القائمة». واللمسةُ تملأ النموذج.
 */

import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { IconClose, IconLocation } from "@rahalgo/ui";
import { localPhone } from "@/lib/store/recipientSuggest";
import type { DeliveryRecipient } from "@/lib/store/types";

const D = getMessages(defaultLocale).store.delivery;

/**
 * **الضغطُ بالفأرة لا يُخرج المؤشّرَ من الحقل** — وإلّا طُويت القائمةُ قبل أن تصل الضغطة
 * (سفاري لا يعطي الزرَّ تركيزاً، فيُقرأ خروجاً من الحقل). **ولوحةُ المفاتيح تمرّ بـ`Tab`.**
 */
const keepFocus = (e: { preventDefault: () => void }) => e.preventDefault();

export function RecipientSuggestions({
  items,
  disabled,
  onPick,
  onHide,
}: {
  items: DeliveryRecipient[];
  disabled: boolean;
  onPick: (r: DeliveryRecipient) => void;
  onHide: (r: DeliveryRecipient) => void;
}) {
  if (items.length === 0) return null;
  return (
    <div data-testid="delivery-recipients">
      <p className="mb-rg-xs text-caption font-bold text-ink-muted">{D.savedTitle}</p>
      {/* **بطاقاتٌ تُمرَّر أفقيّاً** (التصميمُ المعتمد ٢٠٢٦-١١-٢٧) — اللمسةُ تملأ النموذج، و✕ تُخفي المستلم. */}
      <ul className="-mx-rg-xs flex gap-rg-sm overflow-x-auto px-rg-xs pb-rg-xs [scrollbar-width:none]">
        {items.map((r) => (
          <li key={r.phone} className="relative shrink-0">
            <button
              type="button"
              disabled={disabled}
              onMouseDown={keepFocus}
              onClick={() => onPick(r)}
              className="flex min-h-14 w-44 flex-col items-start gap-0.5 rounded-card border border-line bg-paper py-rg-sm ps-rg-md pe-10 text-start transition-colors hover:bg-row-hover focus-visible:border-primary disabled:opacity-60"
            >
              <span className="block max-w-full truncate text-body font-bold text-ink">{r.name}</span>
              <span className="block max-w-full truncate text-caption text-ink-muted" dir="ltr">
                {localPhone(r.phone)}
              </span>
              {r.lat != null && r.lng != null && (
                <span className="flex items-center gap-1 text-caption font-bold text-success">
                  <IconLocation size={12} aria-hidden />
                  {D.savedHasPoint}
                </span>
              )}
            </button>
            <button
              type="button"
              disabled={disabled}
              onMouseDown={keepFocus}
              onClick={() => onHide(r)}
              aria-label={D.savedHide}
              title={D.savedHide}
              className="absolute end-1 top-1 flex h-8 w-8 items-center justify-center rounded-full text-ink-muted transition-colors hover:bg-row-hover disabled:opacity-60"
            >
              <IconClose size={14} />
            </button>
          </li>
        ))}
      </ul>
    </div>
  );
}
