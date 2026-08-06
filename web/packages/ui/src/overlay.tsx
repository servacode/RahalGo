"use client";

/**
 * **ما يعلو الصفحة** — ورقةٌ تصعد، وتلميحٌ يشرح، ومفتاحٌ يُبدَّل.
 *
 * # ولماذا الورقةُ لا النافذة
 *
 * نوافذُ المنصة كلُّها مركزيّةٌ (`Modal`). **وعلى شاشةِ جوّالٍ طويلةٍ تقف
 * النافذةُ في الوسط وأزرارُها في منتصفها** — والإبهامُ يصل الثلثَ السفليَّ
 * وحدَه، **فيُمسك الهاتفُ بيدين أو يُمدّ الإبهامُ فيهتزّ الجهاز.**
 *
 * **والورقةُ تصعد من الأسفل** فتقع أزرارُها حيث اليد.
 */

import { useEffect, useId, useState } from "react";
import type { ReactNode } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { IconClose } from "./icons";

const m = getMessages(defaultLocale);

// ══════════════════════════════════════════════════════════════════════
//  Sheet — ورقةٌ على الجوّال، نافذةٌ على الواسع
// ══════════════════════════════════════════════════════════════════════

/**
 * Sheet طبقةٌ واحدةٌ بسلوكين — **ورقةٌ تصعد على الضيّق ونافذةٌ في الوسط على
 * الواسع.**
 *
 * **ومكوّنان لهما لَافترقا**: يُصلَح الإغلاقُ في أحدهما ويبقى الآخر.
 *
 * # ومقبضٌ يُرى
 *
 * الشريطُ الصغيرُ أعلى الورقة **ليس زخرفة**: هو ما يقول إنّها تُسحب لتُغلق،
 * **وورقةٌ بلا مقبضٍ يُبحث عن زرِّ إغلاقها.**
 *
 * # وقفلُ التمرير خلفها
 *
 * **بلاه تُمرَّر الصفحةُ تحت الورقة** حين يصل المستخدمُ إلى آخرها، **فيضيع
 * موضعُه** ويجد نفسَه في مكانٍ آخرَ حين تُغلق.
 */
export function Sheet({
  open,
  onClose,
  title,
  children,
  footer,
}: {
  open: boolean;
  onClose: () => void;
  title: string;
  children: ReactNode;
  /** أزرارُ الحسم — **تلتصق أسفلَ الورقة فلا تُطلب بالتمرير.** */
  footer?: ReactNode;
}) {
  const id = useId();

  useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => e.key === "Escape" && onClose();
    document.addEventListener("keydown", onKey);
    // **والصفحةُ خلفها تتجمّد** — وإلّا مُرِّرت تحتها فضاع موضعُ صاحبها.
    const prev = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    return () => {
      document.removeEventListener("keydown", onKey);
      document.body.style.overflow = prev;
    };
  }, [open, onClose]);

  if (!open) return null;

  return (
    <div
      className="fixed inset-0 z-[65] flex items-end justify-center scrim sm:items-center sm:p-4"
      onClick={onClose}
    >
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby={id}
        onClick={(e) => e.stopPropagation()}
        className="surface-sheet flex max-h-[88vh] w-full flex-col sm:max-h-[90vh] sm:max-w-lg"
      >
        {/* **المقبضُ على الجوّال وحدَه** — على الواسع نافذةٌ لا تُسحب. */}
        <span
          aria-hidden
          className="mx-auto mt-2 h-1 w-10 shrink-0 rounded-badge bg-line-soft sm:hidden"
        />

        <div className="flex shrink-0 items-start gap-2 px-5 pt-3 pb-2">
          <h2 id={id} className="min-w-0 flex-1 text-lg font-bold">
            {title}
          </h2>
          <button
            type="button"
            onClick={onClose}
            aria-label={m.common.close}
            className="-me-1 shrink-0 rounded-control p-1.5 text-ink-muted transition-colors hover:text-ink"
          >
            <IconClose size={18} />
          </button>
        </div>

        {/* **والمحتوى وحدَه يُمرَّر** — والعنوانُ وأزرارُ الحسم ثابتان. */}
        <div className="min-h-0 flex-1 overflow-y-auto px-5 pb-4">{children}</div>

        {footer && (
          <div className="shrink-0 border-t border-line-soft px-5 py-3 [padding-bottom:max(0.75rem,env(safe-area-inset-bottom))]">
            {footer}
          </div>
        )}
      </div>
    </div>
  );
}

// ══════════════════════════════════════════════════════════════════════
//  Tooltip
// ══════════════════════════════════════════════════════════════════════

/**
 * Tooltip شرحٌ قصيرٌ عند الوقوف — **ولمن لا فأرةَ له يُضغط.**
 *
 * # ولماذا وُجد
 *
 * **النصُّ الطويلُ يُقصّ في المنصة بلا بديل** (`truncate` في عشرات المواضع):
 * اسمُ متجرٍ طويلٌ يصير «مطعم بيت الر…» **ولا سبيلَ لقراءة الباقي.**
 *
 * **و`title=""` الأصليّةُ لا تظهر على اللمس أصلاً** — وأكثرُ مستخدمي المنصة
 * على جوّال.
 *
 * # ولا يُستعمل لما يجب أن يُقرأ
 *
 * **ما لا يُرى إلّا بالوقوف عليه لا يُقرأ**: التلميحُ للتفصيل الزائد لا
 * للمعلومة اللازمة.
 */
export function Tooltip({
  label,
  children,
  side = "top",
}: {
  label: string;
  children: ReactNode;
  side?: "top" | "bottom";
}) {
  const [on, setOn] = useState(false);
  const id = useId();
  const pos = side === "top" ? "bottom-full mb-1.5" : "top-full mt-1.5";

  return (
    <span
      className="relative inline-flex"
      onMouseEnter={() => setOn(true)}
      onMouseLeave={() => setOn(false)}
      onFocus={() => setOn(true)}
      onBlur={() => setOn(false)}
    >
      <span aria-describedby={on ? id : undefined} className="inline-flex min-w-0">
        {children}
      </span>
      {on && (
        <span
          role="tooltip"
          id={id}
          className={`elev-3 pointer-events-none absolute start-1/2 z-50 w-max max-w-[16rem] -translate-x-1/2 rounded-control border border-line bg-raised px-2.5 py-1.5 text-xs text-ink ${pos}`}
        >
          {label}
        </span>
      )}
    </span>
  );
}

// ══════════════════════════════════════════════════════════════════════
//  Switch
// ══════════════════════════════════════════════════════════════════════

/**
 * Switch مفتاحُ تبديل — **لما يقع فوراً.**
 *
 * # وما الفرقُ عن `Checkbox`
 *
 * **المربّعُ اختيارٌ يُحفظ بزرّ** («أوافق على الشروط» ثمّ «تسجيل»)،
 * **والمفتاحُ يقع لحظةَ يُلمس** («تفعيل الإشعارات»).
 *
 * **وخلطُهما يجعل المستخدمَ يبحث عن زرِّ حفظٍ لا وجودَ له**، أو يظنّ أنّ ما
 * بدّله لم يُحفظ.
 *
 * # ونصٌّ يقول الحال
 *
 * **«مُفعَّل/مُطفأ» بجانب المفتاح**: لونٌ وحدَه لا يكفي لمن لا يميّز الألوان،
 * **وموضعُ المقبض على شاشةٍ صغيرةٍ يُقرأ بصعوبة.**
 */
export function Switch({
  checked,
  onChange,
  label,
  hint,
  disabled = false,
  id,
}: {
  checked: boolean;
  onChange: (next: boolean) => void;
  label: string;
  /** سطرٌ يشرح ما يقع — **ومفتاحٌ بلا شرحٍ يُبدَّل بالتجربة.** */
  hint?: string;
  disabled?: boolean;
  id?: string;
}) {
  const auto = useId();
  const btnID = id ?? auto;
  return (
    <div className="flex items-start justify-between gap-3">
      <label htmlFor={btnID} className="min-w-0 flex-1 cursor-pointer">
        <span className="block text-sm font-medium">{label}</span>
        {hint && <span className="mt-0.5 block text-xs text-ink-muted">{hint}</span>}
      </label>
      <button
        id={btnID}
        type="button"
        role="switch"
        aria-checked={checked}
        disabled={disabled}
        onClick={() => onChange(!checked)}
        /* **ومقاسُ اللمس ٤٤ بكسلاً** — والمفتاحُ نفسُه أصغرُ، فالمساحةُ حوله
           تُكمّله. */
        className={`relative inline-flex h-6 w-11 shrink-0 items-center rounded-badge border transition-colors disabled:opacity-50 ${
          checked ? "border-success-edge bg-success-fill" : "border-line bg-page"
        }`}
      >
        <span
          className={`absolute h-4.5 w-4.5 rounded-badge transition-[inset-inline-start] duration-200 ${
            checked ? "start-[1.5rem] bg-success" : "start-0.5 bg-ink-muted"
          }`}
        />
      </button>
    </div>
  );
}
