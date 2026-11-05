"use client";

/**
 * **مستلموه السابقون تحت الحقل** (قرارُ المالك ٢٠٢٦-١١-٠٥) — مرآةُ `RecipientSuggestions` في
 * `DeliveryScreen.kt`: الاسمُ ثمّ «الرقم — العنوان»، **وعلامةُ موقعٍ إن عُرفت نقطتُه**، و✕
 * «شيلو من القائمة». واللمسةُ تملأ النموذج.
 */

import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { IconClose, IconLocation, IconUser } from "@rahalgo/ui";
import { fill } from "@/lib/customer/texts";
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
    <div className="surface-inset -mt-1 overflow-hidden" data-testid="delivery-recipients">
      <p className="px-3 pt-2 text-xs text-ink-muted">{D.savedTitle}</p>
      <ul className="flex flex-col">
        {items.map((r) => (
          <li key={r.phone} className="flex items-center gap-1">
            <button
              type="button"
              disabled={disabled}
              onMouseDown={keepFocus}
              onClick={() => onPick(r)}
              className="flex min-w-0 flex-1 items-center gap-2 px-3 py-2 text-start transition-colors hover:bg-row-hover disabled:opacity-60"
            >
              <IconUser size={18} className="shrink-0 text-accent-text" />
              <span className="min-w-0 flex-1">
                <span className="block truncate text-sm font-bold text-ink">{r.name}</span>
                <span className="block truncate text-xs text-ink-muted">
                  {fill(D.savedLine, { phone: localPhone(r.phone), address: r.address_text })}
                </span>
              </span>
              {r.lat != null && r.lng != null && (
                <IconLocation size={16} className="shrink-0 text-success" aria-label={D.savedHasPoint} />
              )}
            </button>
            <button
              type="button"
              disabled={disabled}
              onMouseDown={keepFocus}
              onClick={() => onHide(r)}
              aria-label={D.savedHide}
              title={D.savedHide}
              className="flex h-10 w-10 shrink-0 items-center justify-center rounded-full text-ink-muted transition-colors hover:bg-row-hover disabled:opacity-60"
            >
              <IconClose size={16} />
            </button>
          </li>
        ))}
      </ul>
    </div>
  );
}
