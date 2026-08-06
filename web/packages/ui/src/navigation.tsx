"use client";

/**
 * **التنقّلُ داخل الصفحة** — تبويبٌ وترقيمٌ وفتاتُ خبز.
 *
 * # ما وجده الفحص
 *
 * **التبويبُ مرتجَلٌ في ستّة ملفّات** — كلٌّ بأصنافه ومقاساته: ملفُّ المتجر،
 * والعروض، والإعدادات، والحسابات، وملفُّ المستخدم، وشاشةُ المحفظة. **وستُّ
 * صياغاتٍ لعنصرٍ واحدٍ تُقرأ ستَّ منصّات.**
 *
 * **والترقيمُ في ستٍّ وخمسين موضعاً بلا مكوّن** — منطقُه مكتوبٌ في كلّ صفحة.
 *
 * # و`TabCards` تبقى لغرضها
 *
 * تلك **بطاقاتٌ تحمل أرقاماً** (رصيدٌ ومجموعٌ ملوَّنٌ بإشارته)، وهذه **شريطُ
 * تبويبٍ عاديّ** — وهو ما ارتُجل ستَّ مرّات. **وواحدٌ لغرضين يصير معقّداً
 * لا مرناً.**
 */

import type { ComponentType, ReactNode } from "react";
import { getMessages, defaultLocale, fmtNum } from "@rahalgo/i18n";
import { IconPrev, IconNext } from "./icons";

const m = getMessages(defaultLocale);

// ══════════════════════════════════════════════════════════════════════
//  Tabs
// ══════════════════════════════════════════════════════════════════════

export interface TabDef<K extends string = string> {
  key: K;
  label: string;
  /** عددٌ بجانب الاسم — **والصفرُ يُعرض ولا يُخفى**: «٠ شكوى» خبرٌ طيّب. */
  count?: number;
  icon?: ComponentType<{ size?: number; className?: string }>;
}

/**
 * Tabs شريطُ تبويبٍ موحَّد.
 *
 * # ولماذا خطٌّ تحت لا صندوقٌ ممتلئ
 *
 * **الصندوقُ الممتلئُ يُقرأ زرّاً**، والتبويبُ ليس فعلاً — هو **موضعٌ أنت
 * فيه**. والخطُّ تحت الاسم يقول «أنت هنا» بلا أن يدعوَ إلى الضغط.
 *
 * **ولا يلتفّ إلى سطرٍ ثانٍ**: ينزلق أفقيّاً على الضيّق — **وتبويبٌ يعلو
 * سطرين يدفع المحتوى ويربك ترتيبَ القراءة.**
 *
 * # وأدوارُ ARIA ليست زينة
 *
 * `role="tablist"` وأخواتُها **تجعل الأسهمَ تنقل بين التبويبات** في قارئات
 * الشاشة، **وبدونها تُقرأ روابطَ متفرّقةً** لا مجموعةً واحدة.
 */
export function Tabs<K extends string>({
  items,
  value,
  onChange,
  className = "",
}: {
  items: readonly TabDef<K>[];
  value: K;
  onChange: (key: K) => void;
  className?: string;
}) {
  return (
    <div
      role="tablist"
      className={`flex gap-1 overflow-x-auto border-b border-line [-ms-overflow-style:none] [scrollbar-width:none] [&::-webkit-scrollbar]:hidden ${className}`}
    >
      {items.map((t) => {
        const on = t.key === value;
        const Icon = t.icon;
        return (
          <button
            key={t.key}
            role="tab"
            type="button"
            aria-selected={on}
            onClick={() => onChange(t.key)}
            /* **والحدُّ السفليُّ محجوزٌ دائماً** (`border-b-2` شفّافاً حين لا
               يكون نشطاً) — **وبدونه يقفز الصفُّ بكسلين** كلّما بُدّل التبويب. */
            className={`-mb-px flex shrink-0 items-center gap-1.5 border-b-2 px-3 py-2.5 text-sm transition-colors ${
              on
                ? "border-accent font-bold text-ink"
                : "border-transparent text-ink-muted hover:text-ink"
            }`}
          >
            {Icon && <Icon size={16} />}
            {t.label}
            {t.count != null && (
              <span
                dir="ltr"
                className={`rounded-badge px-1.5 text-2xs font-bold tabular-nums ${
                  on ? "bg-accent text-on-bright" : "bg-page text-ink-muted"
                }`}
              >
                {fmtNum(t.count)}
              </span>
            )}
          </button>
        );
      })}
    </div>
  );
}

// ══════════════════════════════════════════════════════════════════════
//  Pagination
// ══════════════════════════════════════════════════════════════════════

/**
 * Pagination تنقّلٌ بين الصفحات — **ويقول أين أنت من كم.**
 *
 * # ولماذا «٢ من ٧» لا أرقامٌ مرصوفة
 *
 * **صفوفُ الأرقام (1 2 3 … 47) تصلح لمن يقفز إلى صفحةٍ بعينها** — ولا أحدَ
 * يفعل ذلك في طابور طلبات. **والحاجةُ الحقيقيّةُ: التالي والسابق ومعرفةُ
 * الموضع.**
 *
 * **والعددُ الكلّيُّ يُقال**: من رأى «التالي» بلا رقمٍ لا يعرف أبقيَ صفحةٌ أم
 * عشرون، **فيضغط حتّى يقف الزرُّ.**
 *
 * # والأسهمُ تتبع اتّجاه القراءة
 *
 * **في العربية «التالي» إلى اليسار** — والسهمُ يشير إليه. وهو ما يفعله
 * `IconPrev/Next` أصلاً في هذه العُدّة.
 */
export function Pagination({
  page,
  total,
  perPage,
  onChange,
  busy = false,
}: {
  /** رقمُ الصفحة الحاليّة — **يبدأ من واحدٍ لا من صفر.** */
  page: number;
  /** مجموعُ العناصر لا الصفحات. */
  total: number;
  perPage: number;
  onChange: (page: number) => void;
  busy?: boolean;
}) {
  const pages = Math.max(1, Math.ceil(total / perPage));
  // **ولا يُعرض على صفحةٍ واحدة** — عنصرٌ لا يفعل شيئاً يزاحم ما يفعل.
  if (pages <= 1) return null;

  const btn =
    "flex h-9 items-center gap-1 rounded-control border border-line px-3 text-sm transition-colors hover:border-accent disabled:pointer-events-none disabled:opacity-40";

  return (
    <nav className="flex items-center justify-between gap-3 pt-2" aria-label={m.common.next}>
      <button
        type="button"
        className={btn}
        disabled={page <= 1 || busy}
        onClick={() => onChange(page - 1)}
      >
        <IconPrev size={15} />
        {m.common.back}
      </button>

      {/* **والموضعُ في الوسط** — يُقرأ بلمحةٍ بين الزرّين. */}
      <span dir="ltr" className="text-xs text-ink-muted tabular-nums">
        {fmtNum(page)} / {fmtNum(pages)}
      </span>

      <button
        type="button"
        className={btn}
        disabled={page >= pages || busy}
        onClick={() => onChange(page + 1)}
      >
        {m.common.next}
        <IconNext size={15} />
      </button>
    </nav>
  );
}

// ══════════════════════════════════════════════════════════════════════
//  Breadcrumb
// ══════════════════════════════════════════════════════════════════════

/**
 * Breadcrumb مسارُ العودة — **للوحةٍ بعمق ثلاثة مستويات.**
 *
 * `المتاجر ← مطعم بيت الرقة ← القائمة`
 *
 * **ومن دخل من إشعارٍ إلى صفحةٍ عميقةٍ لا يعرف أين هو** — فيضغط رجوعَ
 * المتصفّح، **ويخرج من المنصة كلَّها.**
 *
 * **والأخيرُ ليس رابطاً**: أنت فيه، **ورابطٌ إلى المكان نفسِه يُضغط فلا يقع
 * شيء** فيُقرأ عطباً.
 */
export function Breadcrumb({
  items,
  Link,
}: {
  items: { label: string; href?: string }[];
  Link: ComponentType<{ href: string; className?: string; children: ReactNode }>;
}) {
  return (
    <nav className="mb-2 flex flex-wrap items-center gap-1 text-xs text-ink-muted">
      {items.map((it, i) => (
        <span key={i} className="flex items-center gap-1">
          {i > 0 && <IconPrev size={12} className="opacity-50" />}
          {it.href && i < items.length - 1 ? (
            <Link href={it.href} className="transition-colors hover:text-ink">
              {it.label}
            </Link>
          ) : (
            <span className={i === items.length - 1 ? "font-medium text-ink" : ""}>{it.label}</span>
          )}
        </span>
      ))}
    </nav>
  );
}
