"use client";

/**
 * **المفتاحُ المقسوم الموحَّد** — خياران أو ثلاثة بعرضٍ متساوٍ على أرضٍ واحدة (إعادةُ التصميم ٢٠٢٦-١١-١١).
 *
 * **مرآةُ `SegmentedToggle` في `mobile/ui`**: أرضُ الحقل، والمختارُ قرصٌ بلون العلامة، وكلُّ قسمٍ
 * بارتفاع لمسٍ مريح (≥44px). **وهو `radiogroup`** فيُقرأ خياراً واحداً من عدّة.
 *
 * نُقل من `components/customer/kit.tsx` (`Segmented`) بالواجهة نفسِها — فيبدّل إليه من شاء بلا تغييرِ نداء.
 */
export function SegmentedToggle<T extends string>({
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
