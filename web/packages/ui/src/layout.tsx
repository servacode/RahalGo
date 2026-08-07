"use client";

/**
 * العناصر البصرية المركزية — الشكل الموحّد لكل صفحة في المشروع.
 * ممنوع أن ترتجل أي صفحة عنواناً أو كرتاً أو حالة فراغ خاصة بها: كل شيء من هنا،
 * فيبقى المشروع بروح واحدة ويكفي تعديل واحد ليطال كل اللوحات (GROUND-RULES §1.2).
 */

import type { ComponentType, ReactNode } from "react";
import { getMessages, defaultLocale, fmtNum, fmtDate, fmtTime } from "@rahalgo/i18n";
import { CountBadge } from "./components";
import { SkeletonList, SkeletonStats } from "./feedback";
import { IconStar } from "./icons";
import { BrandMark } from "./platform";

const m = getMessages(defaultLocale);

type IconType = ComponentType<{ size?: number; className?: string }>;

// ---------- حاوية الصفحة ----------

const widths = {
  narrow: "mx-auto w-full max-w-lg", // نماذج ضيقة (الحساب، الإعدادات)
  medium: "mx-auto w-full max-w-2xl", // محتوى متوسط
  wide: "mx-auto w-full max-w-6xl", // جداول وقوائم
  full: "w-full", // يملأ العرض
} as const;

/** حاوية الصفحة الموحّدة — عرض وإيقاع رأسي واحد بدل الارتجال في كل صفحة. */
export function PageContainer({
  width = "full",
  className = "",
  children,
}: {
  width?: keyof typeof widths;
  className?: string;
  children: ReactNode;
}) {
  return <div className={`${widths[width]} space-y-5 ${className}`}>{children}</div>;
}

// ---------- عنوان الصفحة ----------

/** عنوان الصفحة الموحّد — حجم وأيقونة وتباعد واحد في كل اللوحات. */
export function PageHeader({
  icon: Icon,
  title,
  subtitle,
  actions,
}: {
  icon?: IconType;
  title: string;
  subtitle?: string;
  actions?: ReactNode;
}) {
  return (
    /* ══════════════════════════════════════════════════════════════════
       **وشريطُ العنوان زجاجٌ — لأنّ ما تحته يتبدّل**
       ══════════════════════════════════════════════════════════════════

       (كشفه جردُ السائق ٢٠٢٦-٠٨-٠٦ بعد حذف كرت المحتوى بقرار المالك.)

       **كان يستظلّ باللوح الجامع** — فلمّا ذهب صار يقف على الخلفيّة
       المتدرّجة. **وقِيس**: «التقييمات» ٢٫٣٤ ووصفُها ٢٫١١، و«ما نفّذتَه»
       ٣٫٨٢ — **كلُّها دون حدّها.**

       **وعنوانٌ يُقرأ في صفحةٍ ولا يُقرأ في أخرى أسوأُ من عنوانٍ ثابتِ
       اللون** — لأنّ صاحبَه يظنّ العيبَ في عينه.

       **وشريطٌ لا صندوق**: الحشوةُ ضيّقةٌ والزاويةُ واحدة — **فهو أثاثُ
       العنوان لا كرتٌ جامعٌ عاد من الباب الخلفيّ.** */
    <div className="surface-lit flex flex-wrap items-center justify-between gap-3 surface px-4 py-3">
      <div className="min-w-0">
        <h1 className="heading-section flex items-center gap-2">
          {Icon && <Icon size={20} className="text-primary" />}
          {title}
        </h1>
        {subtitle && <p className="mt-1 text-sm text-ink-muted">{subtitle}</p>}
      </div>
      {actions && <div className="flex flex-wrap items-center gap-2">{actions}</div>}
    </div>
  );
}

// ---------- الكرت ----------

const pads = { sm: "p-3", md: "p-4", lg: "p-5" } as const;

/** الكرت الموحّد — الحاوية البصرية الوحيدة المسموحة للمحتوى. */
/** نبراتُ السطح — **الحدُّ يقول المعنى والسطحُ يبقى واحداً.** */
const tones = {
  default: "border-line",
  danger: "border-danger-edge",
  success: "border-success-edge",
  accent: "border-accent-edge",
} as const;

export function Card({
  title,
  icon: Icon,
  padding = "md",
  tone = "default",
  as: Tag = "section",
  className = "",
  actions,
  children,
}: {
  title?: string;
  icon?: IconType;
  padding?: keyof typeof pads;
  /**
   * **نبرةُ الحدّ لا لونُ السطح.**
   *
   * (جردُ ٢٠٢٦-٠٨-٠٦: سبعةٌ وسبعون سطحاً مبنيّاً باليد، **وأكثرُها بُني
   *  لأنّه أراد حدّاً بلونٍ آخر** — كقسم حذف الحساب.)
   *
   * **والسطحُ يبقى زجاجاً في كلّ النبرات**: لونٌ خافتٌ خلف النصّ يُسقط
   * تباينَه، **والحدُّ يقول «خطر» بلا أن يمسّ ما يُقرأ.**
   */
  tone?: keyof typeof tones;
  /**
   * **وعنصرُ HTML يُختار** — `section` افتراضاً، **و`div` لِما ليس قسماً
   * دلاليّاً.** (وهذا ما دفع ملفّاتٍ إلى بناء سطحها بيدها.)
   */
  as?: "section" | "div" | "article" | "aside";
  className?: string;
  actions?: ReactNode;
  children: ReactNode;
}) {
  return (
    /* **والبطاقةُ تتلقّى ضوءاً وتستجيب.**

       `surface-lit` تعطيها عمقاً — ضوءٌ من أعلى وثقلٌ في القاع وخيطٌ لامعٌ
       على الحرف. **والاستجابةُ عند المرور** تجعلها تُحسّ حيّةً: ترتفع قليلاً
       بظلٍّ أعمق. **وسطحٌ لا يردّ على يدٍ تمرّ عليه سطحٌ ميّت.** */
    <Tag
      className={`surface-lit rounded-card border ${tones[tone]} bg-surface transition-shadow duration-200 hover:shadow-e3 ${pads[padding]} ${className}`}
    >
      {(title || actions) && (
        <div className="mb-3 flex items-center justify-between gap-2">
          {title && (
            <h2 className="flex items-center gap-2 text-sm font-bold">
              {Icon && <Icon size={16} className="text-primary" />}
              {title}
            </h2>
          )}
          {actions}
        </div>
      )}
      {children}
    </Tag>
  );
}

// ---------- حالات الفراغ والتحميل ----------

/** حالة "لا بيانات" الموحّدة. */
export function EmptyState({
  icon: Icon,
  title,
  tone = "muted",
  action,
}: {
  icon?: IconType;
  title: string;
  tone?: "muted" | "success";
  action?: ReactNode;
}) {
  return (
    <div className="surface-lit surface p-10 text-center">
      {Icon && <Icon size={28} className="mx-auto mb-2 text-ink-muted" />}
      <p className={`text-sm ${tone === "success" ? "text-success" : "text-ink-muted"}`}>{title}</p>
      {action && <div className="mt-3 flex justify-center">{action}</div>}
    </div>
  );
}

/** حالة التحميل الموحّدة. */
/**
 * **حالةُ التحميل تحجز المساحةَ ولا تعِدُ بها.**
 *
 * كانت سطرَ نصٍّ واحداً (`جارٍ التحميل`) **محلَّ صفحةٍ كاملة** — وثلاثون
 * موضعاً تستدعيها هكذا:
 *
 * ```tsx
 * if (!data) return <LoadingState />;
 * ```
 *
 * **فحين تصل البيانات تنمو الصفحةُ من سطرٍ إلى شاشة**، ويقفز كلُّ ما فيها.
 * **ومن كان إصبعُه فوق موضعٍ ضغط غيرَه** — وهو `Layout Shift`، وممنوعٌ صراحةً
 * في معايير القبول.
 *
 * # ولماذا الافتراضيُّ قائمة
 *
 * **أكثرُ شاشات المنصة قوائم**: طلباتٌ ومتاجرُ وحركاتٌ وإشعارات. **والهيكلُ
 * يقول شكلَ ما هو آتٍ** — من رأى صفوفاً عرف أنّ قائمةً تُحمَّل.
 *
 * **و`text` تبقى لمن يحتاج سطراً** داخل بطاقةٍ صغيرةٍ لا صفحةً كاملة.
 */
export function LoadingState({
  label,
  variant = "list",
  rows = 4,
}: {
  label?: string;
  variant?: "list" | "stats" | "text";
  rows?: number;
}) {
  if (variant === "text") {
    /* **وسطرُ الانتظار على لوحٍ لا على الخلفيّة.**

       (كشفه جردُ السائق ٢٠٢٦-٠٨-٠٦ بعد حذف كرت المحتوى: **٣٫٦١** — والخلفيّةُ
        متدرّجةٌ فيتبدّل ما تحته.)

       **وذهابُ الكرت الجامع كشف كلَّ ما كان يستره** — وهذا أوّلُهم: سطرٌ
       وحيدٌ يملأ الشاشة أثناء الجلب. **ولوحٌ صغيرٌ حوله يكفي.** */
    return (
      <p className="surface-lit surface py-10 text-center text-sm text-ink-muted">
        {label ?? m.common.loading}
      </p>
    );
  }
  return (
    <div
      role="status"
      aria-live="polite"
      aria-label={label ?? m.common.loading}
      className="space-y-3"
    >
      {variant === "stats" ? <SkeletonStats /> : <SkeletonList rows={rows} />}
    </div>
  );
}

// ---------- صف قائمة ----------

/** صف قائمة موحّد — البديل المركزي للكروت المرصوفة يدوياً. */
export function ListRow({
  leading,
  title,
  subtitle,
  trailing,
  onClick,
  className = "",
}: {
  leading?: ReactNode;
  title: ReactNode;
  subtitle?: ReactNode;
  trailing?: ReactNode;
  onClick?: () => void;
  className?: string;
}) {
  return (
    <li
      onClick={onClick}
      className={`flex items-center gap-3 surface p-3 ${
        onClick ? "cursor-pointer transition-shadow hover:border-primary-edge hover:elev-1" : ""
      } ${className}`}
    >
      {leading}
      <div className="min-w-0 flex-1">
        <div className="truncate font-medium">{title}</div>
        {subtitle && <div className="mt-0.5 text-xs text-ink-muted">{subtitle}</div>}
      </div>
      {trailing && <div className="flex shrink-0 items-center gap-3">{trailing}</div>}
    </li>
  );
}

// ---------- كروت الإحصاء ----------

/** شبكة كروت الإحصاء الموحّدة. */
export function StatGrid({ children }: { children: ReactNode }) {
  return <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4">{children}</div>;
}

/** كرت مؤشر موحّد — شكل واحد لكل الأرقام في المشروع. */
export function StatCard({
  icon: Icon,
  label,
  value,
  sub,
  tone = "default",
  onClick,
  selected = false,
}: {
  icon?: IconType;
  label: string;
  value: string | number;
  sub?: string;
  tone?: "default" | "success" | "danger" | "accent";
  onClick?: () => void;
  selected?: boolean;
}) {
  const toneCls = {
    default: "text-ink",
    success: "text-success",
    danger: "text-danger",
    accent: "text-accent-dark",
  }[tone];
  const Tag = onClick ? "button" : "div";
  return (
    <Tag
      onClick={onClick}
      className={`rounded-card border bg-surface p-4 text-start transition-colors ${
        selected ? "border-primary ring-1 ring-primary-edge" : "border-line"
      } ${onClick ? "hover:border-primary-edge" : ""}`}
    >
      {Icon && <Icon size={18} className="mb-1 text-ink-muted" />}
      <p className={`figure ${toneCls}`}>
        {typeof value === "number" ? fmtNum(value) : value}
      </p>
      <p className="text-xs text-ink-muted">{label}</p>
      {sub && <p className="mt-0.5 text-xs text-ink-muted">{sub}</p>}
    </Tag>
  );
}

// ---------- النجوم ----------

/** عرض التقييم بالنجوم — نسخة واحدة بلون التمييز المركزي. */
export function Stars({
  value,
  size = "md",
  onChange,
}: {
  value: number;
  size?: "sm" | "md" | "lg";
  /** عند تمريرها تصبح النجوم قابلة للنقر (وضع التقييم) */
  onChange?: (v: number) => void;
}) {
  const n = Math.max(0, Math.min(5, Math.round(value)));
  const px = { sm: 14, md: 18, lg: 30 }[size];
  // نجوم من مجموعة الأيقونات لا من محرف ★ — تتبع التوكنز وتظهر متطابقة في كل نظام
  return (
    <span
      dir="ltr"
      className="inline-flex items-center gap-0.5 align-middle"
      role={onChange ? "radiogroup" : undefined}
      aria-label={`${n}/5`}
    >
      {[1, 2, 3, 4, 5].map((i) => {
        const star = (
          <IconStar
            size={px}
            strokeWidth={1.8}
            className={i <= n ? "fill-accent text-accent-text" : "text-line"}
          />
        );
        return onChange ? (
          <button
            key={i}
            type="button"
            onClick={() => onChange(i)}
            aria-label={String(i)}
            aria-checked={i === n}
            role="radio"
            className="rounded-control p-0.5 transition-transform hover:scale-110"
          >
            {star}
          </button>
        ) : (
          <span key={i}>{star}</span>
        );
      })}
    </span>
  );
}



// ---------- تبويبات ----------

export interface TabItem {
  /** المفتاح المستعمل في المقارنة — لا يُعرض */
  key: string;
  label: string;
  /** عدّاد اختياري بجانب العنوان — يُخفى إن كان صفراً */
  count?: number;
  /** رقم التبويب الحقيقي (مجموع، رصيد…) — يُعرض في نمط البطاقات ويُلوَّن بإشارته */
  value?: number;
}

/**
 * بطاقات تبويب — حين يحمل كل تبويب **رقماً** يهمّ القارئ.
 *
 * الشريط العادي يخفي الرقم حتى تضغط، فتضطرّ للمرور على التبويبات واحداً واحداً
 * لتعرف أين ذهب مالك. البطاقة تعرضه فوراً، فيصير شريط التنقّل نفسه لوحةَ ملخّص.
 * ولذلك لا تُستعمل إلا حيث توجد أرقام: بطاقاتٌ بلا أرقام مساحةٌ مهدورة —
 * لمثلها يُبنى شريط تبويب بسيط حين تدعو الحاجة، لا تُحشر في بطاقات فارغة.
 *
 * وتبقى مضغوطة عمداً — سطران لا ثلاثة — كي لا تدفع المحتوى الحقيقي خارج الشاشة.
 */
export function TabCards({
  items,
  active,
  onChange,
  className = "",
}: {
  items: TabItem[];
  active: string;
  onChange: (key: string) => void;
  className?: string;
}) {
  return (
    <div
      role="tablist"
      className={`grid grid-cols-2 gap-2 sm:grid-cols-3 lg:grid-cols-4 ${className}`}
    >
      {items.map((t) => {
        const on = t.key === active;
        const v = t.value;
        return (
          <button
            key={t.key}
            type="button"
            role="tab"
            aria-selected={on}
            onClick={() => onChange(t.key)}
            className={`rounded-card border p-3 text-start transition-colors ${
              on
                ? "border-primary bg-primary-tint"
                : "border-line bg-surface hover:border-primary-edge"
            }`}
          >
            <span className="flex items-center justify-between gap-2">
              <span
                className={`truncate text-sm font-medium ${on ? "text-primary-dark" : "text-ink"}`}
              >
                {t.label}
              </span>
              {!!t.count && (
                <CountBadge count={t.count} on={on} />
              )}
            </span>
            {v !== undefined && (
              <span
                className={`figure mt-1 block ${ v > 0 ? "text-success" : v < 0 ? "text-danger" : "text-ink-muted" }`}
                dir="ltr"
              >
                {v > 0 ? "+" : ""}
                {fmtNum(v)}
              </span>
            )}
          </button>
        );
      })}
    </div>
  );
}

// ---------- بطاقة كيان ----------

export interface EntityStat {
  label: string;
  value: ReactNode;
  /** أيقونة الحقل — تُميّزه عن جاره بلا قراءة، وتُقرأ من بعيد */
  icon?: ReactNode;
  /** لون القيمة — للأرقام التي تعني ربحاً أو خسارة */
  tone?: "default" | "success" | "danger" | "muted";
}

const statTone = {
  default: "text-ink",
  success: "text-success",
  danger: "text-danger",
  muted: "text-ink-muted",
} as const;

/**
 * بطاقة كيان — الشكل الموحّد لعرض «شيء له هوية»: متجر، سائق، مستخدم.
 *
 * بُنيت لأن الصفوف المسطّحة (`ListRow`) تصلح لسجلٍّ يُمسح بالعين، لا لكيانٍ
 * يُقاس ويُقارَن. الفرق أن الكيان له **أرقامه**: البطاقة تعرضها في شريط ثابت
 * الموضع، فتُقارَن بطاقتان بالنظر بلا قراءة.
 *
 * وثلاثة قرارات تحكم شكلها:
 *  - **الهوية أولاً** (صورة/أيقونة + اسم + تصنيف + حالة): من يبحث عن متجر
 *    يبحث باسمه لا برقمه.
 *  - **الأرقام في شريط بمواضع ثابتة**: عمود يقابل عموداً في البطاقة المجاورة.
 *  - **الإجراء في القاع مفصولاً**: لا يُضغط سهواً عند تصفّح البطاقات.
 */
export function EntityCard({
  media,
  title,
  subtitle,
  badge,
  stats,
  footer,
  actions,
  muted = false,
  spine,
  className = "",
}: {
  media?: ReactNode;
  title: ReactNode;
  subtitle?: ReactNode;
  badge?: ReactNode;
  stats?: EntityStat[];
  footer?: ReactNode;
  actions?: ReactNode;
  /** كيان غير فعّال — يبهت بلا أن يختفي */
  muted?: boolean;
  /**
   * عمودٌ ملوّنٌ على حافّة البطاقة يقول حالتَها.
   *
   * **يُقرأ قبل أن تُقرأ الشارة**: شبكةٌ من اثنتي عشرة بطاقة تُمسح بالعين
   * مسحاً، **والشارةُ نصٌّ يلزمه وقوف**. والعمودُ لونٌ يُلتقط في اللمحة
   * الأولى — فيعرف صاحبُه أيَّ بطاقةٍ يقصد قبل أن يقرأ شيئاً.
   */
  spine?: "primary" | "success" | "danger" | "warning" | "info" | "violet";
  className?: string;
}) {
  const SPINE: Record<string, string> = {
    primary: "border-s-primary",
    success: "border-s-success",
    danger: "border-s-danger",
    warning: "border-s-warning",
    info: "border-s-info",
    violet: "border-s-violet",
  };
  return (
    <div
      className={`flex flex-col rounded-card border bg-surface p-4 transition-shadow hover:elev-2 ${
        spine ? `border-s-4 ${SPINE[spine]} ` : ""
      }${
        muted ? "border-dashed border-line opacity-75" : "border-line"
      } ${className}`}
    >
      <div className="flex items-start gap-3">
        {media && <div className="shrink-0">{media}</div>}
        <div className="min-w-0 flex-1">
          <p className="truncate font-bold">{title}</p>
          {subtitle && <p className="mt-0.5 truncate text-xs text-ink-muted">{subtitle}</p>}
        </div>
        {badge && <div className="shrink-0">{badge}</div>}
      </div>

      {!!stats?.length && (
        <div
          className="mt-3 grid gap-2 border-t border-line-soft pt-3"
          style={{ gridTemplateColumns: `repeat(${stats.length}, minmax(0, 1fr))` }}
        >
          {stats.map((st, i) => (
            // حقلٌ مؤطَّر لكل رقم: بلا إطار تلتصق الأرقام فيُقرأ أحدها مكان
            // الآخر — وهي بطاقة تُمسح بالعين لا تُدرَس.
            <div key={i} className="min-w-0 rounded-control bg-field px-2 py-1.5 text-center">
              <p
                className={`truncate text-base font-bold ${statTone[st.tone ?? "default"]}`}
                dir="ltr"
              >
                {st.value}
              </p>
              <p className="mt-0.5 flex items-center justify-center gap-1 text-2xs text-ink-muted">
                {st.icon && (
                  <span className="shrink-0 [&>svg]:h-3.5 [&>svg]:w-3.5">{st.icon}</span>
                )}
                <span className="truncate">{st.label}</span>
              </p>
            </div>
          ))}
        </div>
      )}

      {footer && (
        <p className="mt-3 border-t border-line-soft pt-2.5 text-2xs leading-relaxed text-ink-muted">
          {footer}
        </p>
      )}

      {actions && (
        <div className="mt-3 flex flex-wrap gap-1.5 border-t border-line-soft pt-3">{actions}</div>
      )}
    </div>
  );
}

/**
 * ترويسةُ ورقةٍ رسمية — للفاتورة ولكشف الحساب معاً.
 *
 * **ثلاثةُ أثلاث بترتيبٍ ثابت**: العلامةُ في المبدأ، واسمُ المنصة في الوسط،
 * ووقتُ الطباعة في المنتهى. **ووقتُ الطباعة لا تاريخُ المستند** — واللفظُ
 * يقول ذلك صراحةً:
 *
 * ورقةٌ تحمل تاريخاً بلا لفظٍ يشرحه تُقرأ بعد شهرٍ على أنها تاريخُ الطلب،
 * **فيُحاسَب أحدٌ على يومٍ لم يحدث فيه شيء.** وتاريخُ المستند نفسه (الطلبُ
 * أو المدى) في السطر الذي تحتها — كلٌّ في موضعه ولا يلتبسان.
 *
 * **وواحدةٌ لا اثنتان**: كانت الفاتورةُ وكشفُ الحساب يبنيان ترويستيهما، فتُحسَّن
 * إحداهما وتُنسى الأخرى — وهذا كيف تنشأ الفروقُ التي لا يقصدها أحد.
 */
export function SheetHeader({ printedAt = new Date() }: { printedAt?: Date | string }) {
  return (
    <>
    <div className="mb-4 flex items-center justify-between gap-4 pb-4">
      {/* العلامة — نفسُ علامة الشاشة، فالورقةُ والتطبيقُ شيءٌ واحد */}
      {/* **العلامةُ تُطبع.**

          المتصفّحاتُ لا تطبع الخلفياتِ افتراضاً — فمربّعٌ ملوّنٌ بحرفٍ أبيض
          يخرج **بياضاً على بياض**: ورقةٌ رسمية بلا علامة. والوسمُ هنا تلتقطه
          قاعدةُ طباعةٍ تقلبه إلى إطارٍ وحرفٍ أسودين. */}
      {/* **والعلامةُ من الإعدادات** — شعارٌ إن رُفع وإلّا أوّلُ حرفٍ من الاسم.
          (قرارُ المالك ٢٠٢٦-٠٨-٠٦: «بنماذج الطباعة كما اتّفقنا».)
          **والطريقُ تحته بالنبرة** — اختصارُ اللوغو حين لا شعارَ مرفوع. */}
      <div data-print-mark className="relative shrink-0 overflow-hidden rounded-control">
        <BrandMark size={44} />
        <span aria-hidden className="absolute inset-x-0 bottom-0 h-1.5 bg-accent" />
      </div>

      <div className="min-w-0 flex-1 text-center">
        {/* **ولا اسمَ في الورقة أيضاً** — (قرارُ المالك ٢٠٢٦-٠٨-٠٦).
            **والعلامةُ تكفي**: هي ما يُعرَف به المُصدِر، والوصفُ تحتها يقول
            ما يفعل. */}
        <p className="truncate text-sm text-ink-muted">{m.common.appTagline}</p>
      </div>

      <div className="shrink-0 text-end text-xs text-ink-muted">
        <p>{m.shared.sheet.printedAt}</p>
        <p dir="ltr" className="font-medium text-ink">
          {fmtDate(printedAt)}
        </p>
        <p dir="ltr">{fmtTime(printedAt)}</p>
      </div>
    </div>
    {/* **توقيعُ العلامة** — خيطٌ يمضي من الأساسيّ إلى النبرة تحت الترويسة.
        وهو حدُّها في الوقت نفسه، فلا يزيد على الورقة سطراً. */}
    <div aria-hidden className="brand-rule -mt-4 mb-4 h-[3px] rounded-badge" />
    </>
  );
}
