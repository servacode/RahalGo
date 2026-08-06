"use client";

/**
 * مكونات الواجهة المشتركة — تُستخدم في كل تطبيقات الويب حصراً (GROUND-RULES §1.2).
 * كل الأنماط من توكنز الثيم المركزي، وكلها RTL-جاهزة (خصائص منطقية فقط).
 */

import { useEffect, useRef, useState, type ReactNode } from "react";
import { getMessages, defaultLocale, fmtNum } from "@rahalgo/i18n";
import { IconView, IconViewOff, IconCheck, IconClose } from "./icons";

const m = getMessages(defaultLocale);

// ---------- Button ----------

const buttonVariants = {
  /* **الزرُّ بالنبرة — وهي أسطعُ ما في اللوحة.**

     الأخضرُ صار خلفيةَ كلّ شيء: البطاقةُ والشريطُ والجانب. **وزرٌّ بلون
     البطاقة يذوب فيها**، والنبرةُ تقطعها فتُرى قبل أن تُقرأ — **بالإضاءة
     والإشباع لا بالدرجة**، فاللوحةُ أحاديّة.

     **ونصُّه داكنٌ لا أبيض**: الأبيضُ على النبرة ١٫٣٠ — **يذوب**، والداكنُ
     ١٣٫٩٢. (قرارُ المالك ٢٠٢٦-٠٨-٠٣ للأزرار، ولوناً ٢٠٢٦-٠٨-٠٦.) */
  primary: "bg-accent text-shell hover:opacity-90",
  secondary: "border border-line bg-surface text-ink hover:bg-page",
  danger: "bg-danger text-on-solid hover:bg-danger/90",
  ghost: "text-ink-muted hover:bg-page hover:text-ink",
} as const;

/**
 * مقاسان لا مقاس واحد.
 *
 * `lg` وُلد لتطبيق السائق: يمسك هاتفه بيدٍ واحدة وهو واقفٌ في الشارع، وزرٌّ
 * بارتفاع ٣٦ بكسل يُخطئه الإبهام. والمقاس هنا لا في التطبيق، وإلا صار كلُّ
 * شاشةٍ تُقدّر بنفسها فتتفاوت — وقد رأينا ذلك في هذا المشروع مرّاتٍ.
 */
const buttonSizes = {
  md: "px-4 py-2 text-sm",
  lg: "px-5 py-4 text-base",
} as const;

export function Button({
  variant = "primary",
  size = "md",
  className = "",
  ...props
}: React.ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: keyof typeof buttonVariants;
  size?: keyof typeof buttonSizes;
}) {
  return (
    <button
      {...props}
      className={`rounded-control font-medium transition-colors disabled:pointer-events-none disabled:opacity-60 ${buttonSizes[size]} ${buttonVariants[variant]} ${className}`}
    />
  );
}

// ---------- Input ----------

export function Input({
  label,
  error,
  icon,
  id,
  className = "",
  type,
  ...props
}: React.InputHTMLAttributes<HTMLInputElement> & {
  label?: string;
  error?: string;
  /** أيقونة معبرة تظهر داخل الحقل (جهة البداية) */
  icon?: ReactNode;
}) {
  // حقول كلمات المرور تحصل تلقائياً على زر إظهار/إخفاء (جهة النهاية).
  const [reveal, setReveal] = useState(false);
  const isPassword = type === "password";
  const effectiveType = isPassword && reveal ? "text" : type;

  // الزخارف (الأيقونة وزر الإظهار) تُوضع بالنسبة لاتجاه *الحاوية*، أما حشوة
  // الحقل فتتبع اتجاه *الحقل نفسه*. حقل ltr داخل صفحة rtl (رقم هاتف مثلاً)
  // ينعكس جانباه، فتبقى جهة الأيقونة بلا حشوة ويتداخل النص معها — لذا نحسب
  // الجانبين صراحةً بدل افتراض تطابق الاتجاهين.
  const flipped = props.dir === "ltr";
  const padStart =
    (flipped ? isPassword : !!icon) ? "ps-10" : "ps-3";
  const padEnd =
    (flipped ? !!icon : isPassword) ? "pe-10" : "pe-3";
  return (
    <div>
      {label && (
        <label htmlFor={id} className="mb-1.5 block text-sm font-medium text-ink">
          {label}
        </label>
      )}
      <div className="relative">
        {/* الأيقونة داخل الحقل دائماً — لا تُعلَّق بجانب العنوان */}
        {icon && (
          <span className="pointer-events-none absolute inset-y-0 start-3 flex items-center text-ink-muted [&>svg]:h-[18px] [&>svg]:w-[18px]">
            {icon}
          </span>
        )}
        <input
          id={id}
          type={effectiveType}
          {...props}
          className={`w-full rounded-control border bg-surface py-2.5 text-sm text-ink outline-none transition-colors placeholder:text-ink-muted/60 focus:border-primary focus:ring-2 focus:ring-primary/20 ${
            error ? "border-danger" : "border-line hover:border-ink-muted/40"
          } ${padStart} ${padEnd} ${className}`}
        />
        {isPassword && (
          <button
            type="button"
            onClick={() => setReveal((r) => !r)}
            tabIndex={-1}
            aria-label={reveal ? m.shared.hidePassword : m.shared.showPassword}
            className="absolute inset-y-0 end-3 flex items-center text-ink-muted transition-colors hover:text-ink [&>svg]:h-[18px] [&>svg]:w-[18px]"
          >
            {reveal ? <IconViewOff /> : <IconView />}
          </button>
        )}
      </div>
      {error && <p className="mt-1 text-xs text-danger">{error}</p>}
    </div>
  );
}

// ---------- Textarea ----------

/**
 * **حقلُ نصٍّ طويلٍ موحَّد** — كان مرتجَلاً في كلّ نافذة.
 *
 * # ما وجده فحصُ المركزية
 *
 * لم يكن في العُدّة `Textarea`، **فارتجلته كلُّ نافذةٍ بنفسها** — وافترقتا:
 *
 *	نافذةُ الشكوى : `rounded-input border border-line bg-surface p-2 text-sm`
 *	نافذةُ التقييم: `rounded-control … px-3 py-2 … focus:border-primary focus:ring-2`
 *
 * **وواحدةٌ منهما تُضيء عند التركيز والأخرى لا** — فمن كتب شكواه لا يعرف أين
 * يقف المؤشّر.
 *
 * **و`rounded-input` لا وجودَ له أصلاً**: ليس في الثيم، **فالصنفُ يُكتب ولا
 * يفعل شيئاً** — وحقلُ الشكوى مربّعُ الأركان وكلُّ حقلٍ في المنصة مستدير.
 * **وصنفٌ لا يوجد لا يصرخ**: لا خطأً في البناء ولا تحذيراً، يُقرأ سليماً
 * ويُرسم خطأً.
 *
 * **وحدُّ الطول يُعرض ولا يُخفى**: من كتب مئتَي حرفٍ في حقلٍ سقفُه مئةٌ يفقد
 * نصفَ ما كتب عند الإرسال.
 */
export function Textarea({
  label,
  error,
  id,
  className = "",
  ...props
}: React.TextareaHTMLAttributes<HTMLTextAreaElement> & {
  label?: string;
  error?: string;
}) {
  const used = String(props.value ?? "").length;
  return (
    <div>
      {label && (
        <label htmlFor={id} className="mb-1.5 block text-sm font-medium text-ink">
          {label}
        </label>
      )}
      <textarea
        id={id}
        rows={props.rows ?? 3}
        {...props}
        className={`w-full rounded-control border bg-surface px-3 py-2.5 text-sm text-ink outline-none transition-colors placeholder:text-ink-muted/60 focus:border-primary focus:ring-2 focus:ring-primary/20 ${
          error ? "border-danger" : "border-line hover:border-ink-muted/40"
        } ${className}`}
      />
      <div className="mt-1 flex items-start justify-between gap-2">
        {error ? <p className="text-xs text-danger">{error}</p> : <span />}
        {props.maxLength ? (
          <p className="shrink-0 text-xs text-ink-muted tabular-nums" dir="ltr">
            {fmtNum(used)}/{fmtNum(props.maxLength)}
          </p>
        ) : null}
      </div>
    </div>
  );
}

// ---------- Radio ----------

/**
 * **اختيارٌ واحدٌ من عدّة** — كان `<input type="radio" className="accent-primary">`.
 *
 * **و`accent-primary` يترك الرسمَ للمتصفّح**: دائرةُ ويندوز تخالف دائرةَ
 * أندرويد تخالف دائرةَ سفاري، **والمربّعُ المجاور مرسومٌ بتوكناتنا** — فيقف
 * شكلان في نموذجٍ واحد.
 *
 * **ومساحةُ الضغط هي السطرُ كلُّه لا الدائرةَ وحدَها**: إصبعٌ على جوّالٍ لا
 * يصيب ثمانيةَ عشرَ بكسلاً من أوّل مرّة.
 */
export function Radio({
  label,
  id,
  className = "",
  ...props
}: Omit<React.InputHTMLAttributes<HTMLInputElement>, "type"> & { label: ReactNode }) {
  return (
    <label
      htmlFor={id}
      className={`group flex cursor-pointer items-center gap-2 text-sm ${className}`}
    >
      <span className="relative flex h-[18px] w-[18px] shrink-0 items-center justify-center">
        <input id={id} type="radio" {...props} className="peer sr-only" />
        <span className="absolute inset-0 rounded-full border border-line bg-surface transition-colors peer-checked:border-primary peer-focus-visible:ring-2 peer-focus-visible:ring-primary/30" />
        {/* **والنواةُ تكبر لا تظهر فجأة** — حركةٌ قصيرةٌ تؤكّد أنّ الضغطةَ وقعت. */}
        <span className="relative h-2 w-2 scale-0 rounded-full bg-primary transition-transform peer-checked:scale-100" />
      </span>
      <span className="min-w-0">{label}</span>
    </label>
  );
}

// ---------- Checkbox ----------

/** مربّع اختيار موحّد — مرسوم بالتوكنز لا بمظهر المتصفح الافتراضي. */
export function Checkbox({
  label,
  id,
  className = "",
  ...props
}: Omit<React.InputHTMLAttributes<HTMLInputElement>, "type"> & { label: string }) {
  return (
    <label htmlFor={id} className={`group flex cursor-pointer items-center gap-2 text-sm ${className}`}>
      <span className="relative flex h-[18px] w-[18px] shrink-0 items-center justify-center">
        <input
          id={id}
          type="checkbox"
          {...props}
          className="peer h-full w-full cursor-pointer appearance-none rounded-[5px] border border-line bg-surface transition-colors checked:border-primary checked:bg-primary focus-visible:ring-2 focus-visible:ring-primary/25 focus-visible:outline-none"
        />
        <IconCheck
          size={12}
          strokeWidth={3.5}
          className="pointer-events-none absolute text-on-solid opacity-0 transition-opacity peer-checked:opacity-100"
        />
      </span>
      <span className="text-ink-muted transition-colors group-hover:text-ink">{label}</span>
    </label>
  );
}

// ---------- OtpInput ----------

/**
 * حقل رمز التحقق — خانة لكل رقم بدل حقل واحد طويل. الأرقام تُقرأ وتُصحَّح أسرع،
 * والتقدّم التلقائي واللصق يجعلان إدخال الرمز حركة واحدة.
 * الحاوية LTR دائماً: الأرقام تُملأ من اليسار مهما كانت لغة الواجهة.
 */
export function OtpInput({
  value,
  onChange,
  length = 6,
  autoFocus,
  boxLabel,
  onComplete,
}: {
  value: string;
  onChange: (v: string) => void;
  length?: number;
  autoFocus?: boolean;
  /** تسمية وصفية لكل خانة (تُمرَّر من المعجم) — {n} يُستبدل برقم الخانة */
  boxLabel: string;
  /** يُستدعى عند اكتمال كل الخانات — لتقديم الإرسال بلا نقرة إضافية */
  onComplete?: (v: string) => void;
}) {
  const refs = useRef<(HTMLInputElement | null)[]>([]);
  const digits = value.padEnd(length, " ").slice(0, length).split("");

  function commit(next: string) {
    onChange(next);
    if (next.length === length && !next.includes(" ")) onComplete?.(next);
  }

  function setAt(i: number, digit: string) {
    const chars = digits.map((d) => (d === " " ? "" : d));
    chars[i] = digit;
    commit(chars.join("").slice(0, length));
  }

  function onKey(i: number, e: React.KeyboardEvent<HTMLInputElement>) {
    if (e.key === "Backspace" && !(digits[i] ?? "").trim() && i > 0) {
      e.preventDefault();
      setAt(i - 1, "");
      refs.current[i - 1]?.focus();
      return;
    }
    // الأسهم تتحرك بصرياً: اليمين في RTL هو الخانة السابقة
    if (e.key === "ArrowLeft") refs.current[Math.min(length - 1, i + 1)]?.focus();
    if (e.key === "ArrowRight") refs.current[Math.max(0, i - 1)]?.focus();
  }

  return (
    <div dir="ltr" className="flex justify-center gap-2">
      {digits.map((d, i) => (
        <input
          key={i}
          ref={(el) => {
            refs.current[i] = el;
          }}
          value={d.trim()}
          inputMode="numeric"
          autoComplete={i === 0 ? "one-time-code" : "off"}
          autoFocus={autoFocus && i === 0}
          aria-label={boxLabel.replace("{n}", String(i + 1))}
          maxLength={1}
          onChange={(e) => {
            const digit = e.target.value.replace(/\D/g, "").slice(-1);
            setAt(i, digit);
            if (digit && i < length - 1) refs.current[i + 1]?.focus();
          }}
          onKeyDown={(e) => onKey(i, e)}
          onFocus={(e) => e.target.select()}
          onPaste={(e) => {
            e.preventDefault();
            const pasted = e.clipboardData.getData("text").replace(/\D/g, "").slice(0, length);
            if (!pasted) return;
            commit(pasted);
            refs.current[Math.min(pasted.length, length - 1)]?.focus();
          }}
          className={`h-13 w-11 rounded-control border bg-surface text-center font-mono text-xl font-bold text-ink outline-none transition-all sm:w-12 ${
            d.trim()
              ? "border-primary bg-primary-light/40 text-primary-dark"
              : "border-line hover:border-ink-muted/40"
          } focus:border-primary focus:ring-2 focus:ring-primary/20`}
        />
      ))}
    </div>
  );
}

// ---------- PasswordMeter ----------

/** قوة كلمة المرور: الطول أولاً ثم تنوّع المحارف — مؤشر بصري بألوان الحالات. */
export function passwordScore(pw: string): 0 | 1 | 2 | 3 {
  if (pw.length < 8) return pw.length === 0 ? 0 : 1;
  const variety =
    Number(/[a-z]/.test(pw)) + Number(/[A-Z]/.test(pw)) + Number(/\d/.test(pw)) + Number(/[^\w]/.test(pw));
  if (pw.length >= 12 && variety >= 3) return 3;
  if (variety >= 2) return 2;
  return 1;
}

/** شريط قوة كلمة المرور — يُظهر للمستخدم أثر ما يكتبه بدل رسالة رفض بعد الإرسال. */
export function PasswordMeter({ value, labels }: { value: string; labels: [string, string, string] }) {
  const score = passwordScore(value);
  if (score === 0) return null;
  const tone = (["bg-danger", "bg-warning", "bg-success"] as const)[score - 1];
  const text = (["text-danger", "text-warning", "text-success"] as const)[score - 1];
  return (
    <div className="flex items-center gap-2">
      <div className="flex flex-1 gap-1">
        {[1, 2, 3].map((i) => (
          <span
            key={i}
            className={`h-1 flex-1 rounded-badge transition-colors ${i <= score ? tone : "bg-line"}`}
          />
        ))}
      </div>
      <span className={`text-xs font-medium ${text}`}>{labels[score - 1]}</span>
    </div>
  );
}

// ---------- Select ----------

export function Select({
  label,
  id,
  className = "",
  children,
  ...props
}: React.SelectHTMLAttributes<HTMLSelectElement> & { label?: string }) {
  return (
    <div>
      {label && (
        <label htmlFor={id} className="mb-1 block text-sm font-medium">
          {label}
        </label>
      )}
      <select
        id={id}
        {...props}
        className={`w-full rounded-control border border-line bg-surface px-3 py-2 text-sm outline-none focus:border-primary focus:ring-2 focus:ring-primary/20 ${className}`}
      >
        {children}
      </select>
    </div>
  );
}

// ---------- Badge ----------

const badgeVariants = {
  neutral: "bg-page text-ink-muted",
  primary: "bg-primary-light text-primary-dark",
  success: "bg-success/10 text-success",
  warning: "bg-warning/10 text-warning",
  danger: "bg-danger/10 text-danger",
} as const;

export function Badge({
  variant = "neutral",
  children,
  className = "",
}: {
  variant?: keyof typeof badgeVariants;
  children: ReactNode;
  className?: string;
}) {
  return (
    <span
      className={`inline-flex items-center rounded-badge px-2.5 py-0.5 text-xs font-medium ${badgeVariants[variant]} ${className}`}
    >
      {children}
    </span>
  );
}

// ---------- Modal ----------

const modalSizes = {
  md: "max-w-md",
  lg: "max-w-2xl",
  xl: "max-w-4xl",
} as const;

export function Modal({
  open,
  onClose,
  title,
  size = "md",
  children,
}: {
  open: boolean;
  onClose: () => void;
  title: string;
  size?: keyof typeof modalSizes;
  children: ReactNode;
}) {
  useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => e.key === "Escape" && onClose();
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [open, onClose]);

  if (!open) return null;
  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-ink/40 p-4"
      onClick={onClose}
    >
      <div
        role="dialog"
        aria-modal="true"
        className={`max-h-[90vh] w-full overflow-y-auto rounded-card border border-line bg-surface p-6 elev-3 ${modalSizes[size]}`}
        onClick={(e) => e.stopPropagation()}
      >
        {/* **العنوانُ وزرُّ الإغلاق في سطرٍ واحد.**

            كان الإغلاقُ بثلاثةِ طرقٍ **لا يُرى أيٌّ منها**: الضغطُ خارج
            النافذة، ومفتاحُ `Esc`، وزرٌّ في أسفل بعض النوافذ لا كلِّها.
            **ومن لا يعرف أنّ الخارج يُغلق يبحث عن ✕ فلا يجده** — فيظنّ نفسه
            محبوساً، وعلى الهاتف لا `Esc` ولا «خارج» واضح.

            **وزرٌّ يُرى يُغني عن ثلاثةٍ تُعرَف بالتجربة.**
            (قرارُ المالك ٢٠٢٦-٠٨-٠٣.) */}
        <div className="mb-4 flex items-start justify-between gap-3">
          <h2 className="text-lg font-bold">{title}</h2>
          <button
            type="button"
            onClick={onClose}
            aria-label={m.common.close}
            title={m.common.close}
            className="taparea -me-1.5 -mt-1 flex h-8 w-8 shrink-0 items-center justify-center rounded-control text-ink-muted transition-colors hover:bg-page hover:text-ink"
          >
            <IconClose size={18} />
          </button>
        </div>
        {children}
      </div>
    </div>
  );
}

/** قسم مسمّى داخل النماذج الطويلة — لتنظيم الحقول في مجموعات واضحة */
export function FormSection({
  title,
  icon,
  children,
}: {
  title: string;
  icon?: ReactNode;
  children: ReactNode;
}) {
  return (
    <section>
      <h3 className="mb-3 flex items-center gap-1.5 border-b border-line pb-2 text-sm font-bold text-primary-dark">
        {icon && <span className="[&>svg]:h-4 [&>svg]:w-4">{icon}</span>}
        {title}
      </h3>
      {children}
    </section>
  );
}
