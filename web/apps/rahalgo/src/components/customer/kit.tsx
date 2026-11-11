"use client";

/**
 * **قطعُ شكلٍ محلّيّةٌ لزبون الويب** — مرآةُ ما أُضيف لتطبيق الزبون في إعادة التصميم (٢٠٢٦-١١-١١).
 *
 * (TODO(shared): لو احتاجتها لوحةٌ أخرى تُنقل إلى `@rahalgo/ui` — الآن هي هنا لأنّ الحزمةَ المشتركة
 *  بيد غيري في هذه الدفعة.)
 *
 * **شكلٌ فقط** — لا منطقَ ولا نداء ولا مال. والأسماءُ مرايا لأسماء `mobile/ui`:
 * `SegmentedControl` · `RadioCard` · `StickyActionBar` · بطاقةٌ بإطارٍ بلا ظلّ · نبرةُ حالِ الطلب.
 */

import type { ReactNode } from "react";
import type { KitTone as StatusTone } from "@rahalgo/ui";

/** **بطاقةٌ بحدٍّ لا بظلّ** — مرآةُ `RahalCard(outlined)`. وحشوتُها من التوكنز. */
export const OUTLINE_CARD = "rounded-rg-lg border border-line bg-surface p-rg-lg";

/**
 * **نبرةُ حال الطلب** — مرآةُ `statusTone` في `Status.kt`:
 * انتظارٌ وتوزيعٌ = تحذير · العملُ عليه = معلومة · التسليمُ = نجاح · ما سقط = خطر · وإلّا محايد.
 */
const INFO_STATES = new Set(["accepted", "confirmed", "preparing", "assigned", "at_pickup", "picked_up", "on_the_way", "at_dropoff"]);

export function orderStatusTone(status: string): StatusTone {
  if (status === "delivered") return "success";
  if (status === "cancelled" || status === "rejected" || status === "failed") return "danger";
  if (status === "pending" || status === "dispatching") return "warning";
  if (INFO_STATES.has(status)) return "info";
  return "neutral";
}

/**
 * **مفتاحُ نصفَين متساويَين** — خلفيةُ الحقل الرماديّة وقرصٌ بلون العلامة للمختار (≥40px).
 * مرآةُ `SegmentedControl`. **وهو `radiogroup`** فيُقرأ خياراً واحداً من اثنين.
 */
export function Segmented<T extends string>({
  items,
  value,
  onChange,
  label,
}: {
  items: { id: T; label: string }[];
  value: T;
  onChange: (id: T) => void;
  label?: string;
}) {
  return (
    <div role="radiogroup" aria-label={label} className="flex w-full gap-rg-xxs rounded-rg-md bg-field p-rg-xxs">
      {items.map((it) => {
        const on = it.id === value;
        return (
          <button
            key={it.id}
            type="button"
            role="radio"
            aria-checked={on}
            onClick={() => onChange(it.id)}
            className={`min-h-11 min-w-0 flex-1 rounded-rg-sm px-rg-sm text-button transition-colors ${
              on ? "bg-primary text-on-bright" : "text-ink-muted hover:bg-row-hover"
            }`}
          >
            {it.label}
          </button>
        );
      })}
    </div>
  );
}

/**
 * **صفُّ اختيارٍ بإطار** — راديو حقيقيّ، والمختارُ بحدٍّ ٢px بلون العلامة وتظليلٍ خفيف.
 * **هو `.shop-opt` نفسُه** (خياراتُ دفع السلّة) فلا يختلف شكلُ اختيارٍ عن آخر.
 */
export function RadioCard({
  selected,
  onSelect,
  title,
  hint,
  icon,
  disabled,
  id,
}: {
  selected: boolean;
  onSelect: () => void;
  title: ReactNode;
  hint?: ReactNode;
  icon?: ReactNode;
  disabled?: boolean;
  id?: string;
}) {
  return (
    <button
      type="button"
      id={id}
      role="radio"
      aria-checked={selected}
      data-on={selected}
      disabled={disabled}
      onClick={onSelect}
      className="shop-opt min-h-12 w-full disabled:opacity-60"
    >
      {icon && <span className="shop-opt-ic">{icon}</span>}
      <span className="min-w-0 flex-1">
        <b className="block text-body-strong">{title}</b>
        {hint && <span className="block text-caption text-ink-muted">{hint}</span>}
      </span>
      <i aria-hidden className="shop-opt-dot" />
    </button>
  );
}

/**
 * **شريطُ الفعل السفليّ الثابت** — بلون خلفية الصفحة وظلٍّ مرفوعٍ إلى أعلى (مرآةُ `BottomActionBar`).
 * `aboveTabs`: على الجوّال فوق شريط التبويب السفليّ.
 */
export function StickyActionBar({
  children,
  aboveTabs = false,
  className = "",
}: {
  children: ReactNode;
  aboveTabs?: boolean;
  className?: string;
}) {
  return (
    <div
      className={`sticky z-30 -mx-rg-gutter flex lg:mx-0 lg:rounded-rg-lg lg:border flex-col gap-rg-sm border-t border-line-soft bg-shell px-rg-gutter py-rg-md rg-raised-bar ${
        aboveTabs ? "bottom-[calc(3.9rem+env(safe-area-inset-bottom))] lg:bottom-0" : "bottom-0"
      } ${className}`}
    >
      {children}
    </div>
  );
}
