"use client";

/**
 * مكونات الواجهة المشتركة — تُستخدم في كل تطبيقات الويب حصراً (GROUND-RULES §1.2).
 * كل الأنماط من توكنز الثيم المركزي، وكلها RTL-جاهزة (خصائص منطقية فقط).
 */

import { useEffect, useRef, useState, type ReactNode } from "react";
import { createPortal } from "react-dom";
import { getMessages, defaultLocale, fmtNum, getDir } from "@rahalgo/i18n";
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
  primary: "bg-accent text-on-bright hover:opacity-90",
  secondary: "border border-line bg-surface text-ink hover:bg-row-hover",
  /* **وزرُّ الخطر بالتعبئة المصمتة لا بالفاتحة.**

     (كشفه جردُ السائق ٢٠٢٦-٠٨-٠٦: «إرسال رمز التأكيد» عند **١٫٧٥**.)

     كان `bg-danger` — **وهو أحمرُ فاتحٌ صُنع ليُقرأ نصّاً على بطاقةٍ داكنة**،
     **والأبيضُ عليه ١٫٧٥.** فزرُّ الحذف كان أقلَّ ما يُقرأ في الشاشة، **وهو
     الذي لا يُستدرَك.**

     **و`danger-solid` مصنوعٌ للتعبئة**: الأبيضُ عليه ٤٫٨٣ — **وهو التوكنُ
     نفسُه في عدّاد الإشعارات وأيقونات الخطر.** */
  danger: "bg-danger-solid text-on-solid hover:opacity-90",
  ghost: "text-ink-muted hover:bg-row-hover hover:text-ink",
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

/**
 * **زرٌّ يذهب بك — لا زرٌّ داخل رابط.**
 *
 * كان المشروع يكتب `<Link className="inline-block"><Button/></Link>` — **وهو
 * `<button>` داخل `<a>`**: تعشيشٌ يرفضه المعيار، **وقارئُ الشاشة يقرؤه
 * عنصرين متداخلين** فلا يعرف أهو رابطٌ أم زرّ.
 *
 * **والنغماتُ والمقاساتُ هي هي** — لا خريطةَ ثانيةٌ تشيخ وحدَها.
 */
export function ButtonLink({
  variant = "primary",
  size = "md",
  className = "",
  ...props
}: React.AnchorHTMLAttributes<HTMLAnchorElement> & {
  variant?: keyof typeof buttonVariants;
  size?: keyof typeof buttonSizes;
}) {
  return (
    <a
      {...props}
      className={`inline-flex items-center justify-center gap-2 rounded-control font-medium transition-colors ${buttonSizes[size]} ${buttonVariants[variant]} ${className}`}
    />
  );
}

// ---------- Input ----------

/* ══════════════════════════════════════════════════════════════════════
   **والحقلُ زجاجٌ لا سطحٌ ثانٍ فوق سطح**
   ══════════════════════════════════════════════════════════════════════

   (شهده المالك ٢٠٢٦-٠٨-٠٦ بصورة: البطاقةُ صارت زجاجاً **والحقلان داخلَها
    صندوقان أسودان** — «الحقولُ أيضاً يجب أن تكون شفّافةً بشكلٍ مركزيّ».)

   كانت `bg-surface` — **وهو لونُ البطاقة نفسِه.** فيقع سطحٌ فوق سطح: ٥٢٪
   فوق ٥٢٪ = **سبعةٌ وسبعون بالمئة حجاب.** **وحقلٌ مصمتٌ في بطاقةٍ زجاجيّةٍ
   يكسر الزجاجَ كلَّه** — والعينُ ترى الصندوقين قبل أن ترى اللوح.

   **و`field` غسلةٌ سوداءُ خفيفة** (٢٢٪): تُعمّق الحقلَ عمّا حولَه **بلا أن
   تحجب ما تحته** — فيبقى غائراً وتبقى الخلفيّةُ تُرى فيه.

   **والحدُّ هو ما يرسم الحقل لا اللون** — وهو `line` كما كان.
   ══════════════════════════════════════════════════════════════════════ */
export function Input({
  label,
  error,
  icon,
  id,
  className = "",
  wrapperClassName = "",
  type,
  ...props
}: React.InputHTMLAttributes<HTMLInputElement> & {
  label?: string;
  error?: string;
  /** أيقونة معبرة تظهر داخل الحقل (جهة البداية) */
  icon?: ReactNode;
  /**
   * **صنفٌ للغلاف لا للحقل** — لمن يضعه في صفّ `flex`.
   *
   * (شكوى المالك ٢٠٢٦-٠٨-٠٩: «حقلُ عنوان المكتب صغير».)
   *
   * **و`className` يصل الحقلَ لا غلافَه** — و`w-full` على حقلٍ داخلَ غلافٍ
   * لا ينمو يملأ الغلافَ وحدَه. **فيُعطى الغلافُ `flex-1` من هنا.**
   */
  wrapperClassName?: string;
}) {
  // حقول كلمات المرور تحصل تلقائياً على زر إظهار/إخفاء (جهة النهاية).
  const [reveal, setReveal] = useState(false);
  const isPassword = type === "password";
  const effectiveType = isPassword && reveal ? "text" : type;

  /* ══════════════════════════════════════════════════════════════════
     **والحشوةُ تتبع جهةَ الأيقونة — لا جهةَ نصِّ الحقل**
     ══════════════════════════════════════════════════════════════════

     (شهده المالك ٢٠٢٦-٠٨-٠٧: «ليش خربت؟ الأيقونةُ فايتة برقم الهاتف».)

     **كان هنا قلبٌ حين `dir="ltr"`** بحجّة أنّ حقلاً لاتينيّاً داخلَ صفحةٍ
     عربيّةٍ تنعكس جهاتُه. **وقِيس فتبيّن أنّه لا ينعكس**: حقلُ الهاتف
     أصنافُه `ps-3 pe-10` والمحسوبُ **يمين ١٢ ويسار ٤٠** — أي أنّ الحشوةَ
     المنطقيّةَ حُلّت باتّجاه الصفحة كما حُلّ موضعُ الأيقونة.

     **فاجتمعا على طرفين متقابلين**: الأيقونةُ يميناً بـ`start-3` والحشوةُ
     يساراً. **والنصُّ `text-end` جاء فوقها** — رقمٌ يُقرأ نصفُه.

     **والقاعدةُ بعد القياس واحدة**: الزخرفةُ والحشوةُ التي تُخلي لها مكاناً
     **تُكتبان بالمنطق نفسِه**، فلا يُفترض انعكاسٌ لا يقع.

     ══════════════════════════════════════════════════════════════════
     **وعاد العطبُ لأنّ الحقلَ وحدَه قد يخالف اتّجاهَ الصفحة**
     ══════════════════════════════════════════════════════════════════

     (شهده المالك ثانيةً ٢٠٢٦-٠٨-١٠ في أوّل شاشةٍ على الاستضافة: «الرقمُ
      والأيقونةُ فايتين ببعض».)

     **حقلُ الهاتف عليه `dir="ltr"`** — ورقمٌ دوليٌّ يُقرأ هكذا. **والحشوةُ
     المنطقيّةُ تُحلّ باتّجاه العنصر الذي كُتبت عليه**: `ps-10` على حقلٍ
     لاتينيٍّ تعني **يساراً**.

     **والأيقونةُ في الغلاف — واتّجاهُه اتّجاهُ الصفحة (يمين)**. فتقع
     الأيقونةُ يميناً والحشوةُ يساراً، **والرقمُ `text-end` يرتدّ إلى اليمين
     فيدخل تحتها.**

     **والإصلاحُ الأوّلُ صحيحٌ ولم يكن كافياً**: وحّد المنطقَ بين الاثنين،
     **لكنّه افترض أنّهما في اتّجاهٍ واحد** — وهما ليسا كذلك متى خالف الحقلُ
     صفحتَه.

     **فتُقلب الحشوةُ حين يُخالف** — لتقع في الجهة التي فيها الأيقونة فعلاً. */
  // **ويُقاس بالمقارنة لا بالافتراض**: `dir="rtl"` على صفحةٍ عربيّةٍ لا
  // يخالف شيئاً، **والقلبُ له يكسر ما كان سليماً.**
  const flipped = !!props.dir && props.dir !== getDir(defaultLocale);
  const padStart = icon ? (flipped ? "pe-10" : "ps-10") : flipped ? "pe-3" : "ps-3";
  const padEnd = isPassword ? (flipped ? "ps-10" : "pe-10") : flipped ? "ps-3" : "pe-3";
  return (
    <div className={wrapperClassName}>
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
          /* ══════════════════════════════════════════════════════════
             **ولا حلقةَ خارجيّةً عند التركيز — الحدُّ وحدَه يقول**
             ══════════════════════════════════════════════════════════

             (قرارُ المالك ٢٠٢٦-٠٨-٠٦: «ألغِ البوردر الخارجيّ الذي يظهر على
              حقول الإدخال عند الوقوف عليها».)

             كانت `focus:ring-2 focus:ring-primary-edge` — **فيصير للحقل
             حدّان**: حدُّه يتلوّن، وحلقةٌ باهتةٌ حولَه. **وخطّان متوازيان
             بلونٍ واحدٍ يُقرآن حدّاً سميكاً مهترئاً** لا تمييزاً.

             **وما زال التركيزُ يُرى**: `focus:border-primary` يقلب الحدَّ من
             الرماديّ إلى الأساسيّ — **وهو تغيُّرٌ يكفي** ولا يزيد على الحقل
             حجماً.

             **والمفاتيحُ وصناديقُ الاختيار تُبقي حلقتَها** — حلقتُها
             `focus-visible:` لا `focus:`: **تظهر لمن ينتقل بالكيبورد ولا
             تظهر لمن ضغط بالفأرة**، فلا تزاحم أحداً ولا يفقد أحدٌ موضعَه. */
          className={`w-full rounded-control border bg-field py-2.5 text-sm text-ink outline-none transition-colors placeholder:text-ink-dim focus:border-primary ${
            error ? "border-danger" : "border-line hover:border-line-soft"
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
 *	نافذةُ التقييم: `rounded-control … px-3 py-2 … focus:border-primary`
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
  /**
   * **ينمو بما فيه ثمّ يقف.**
   *
   * (قرارُ المالك ٢٠٢٦-٠٨-١٠: «ما يلزم كلُّ هذا النصّ — شريطُ نصٍّ يكفي».)
   *
   * **وحقلٌ يحجز خمسةَ أسطرٍ لمن يكتب سطراً يكذب على صاحبه**: يقول «أنا
   * أنتظر منك فقرة»، **فيتردّد من أراد أن يكتب «ربطة خبز».** والفراغُ
   * المحجوز يدفع ما تحته خارجَ الشاشة.
   *
   * **ولا يُقصّ ما يُكتب**: من احتاج خمسةَ أسطرٍ ناله — **يأخذها وهو يكتب
   * لا قبل أن يبدأ.**
   *
   * **واختياريٌّ لا افتراض**: حقولُ الملاحظات في اللوحات تُملأ بفقراتٍ
   * وارتفاعُها الثابتُ صوابٌ لها، **وتبديلُ سلوكِ سبعةَ عشرَ حقلاً لأجل
   * واحدٍ يكسر ما لم يُشتكَ منه.**
   */
  autoGrow = false,
  ...props
}: React.TextareaHTMLAttributes<HTMLTextAreaElement> & {
  label?: string;
  error?: string;
  autoGrow?: boolean;
}) {
  const used = String(props.value ?? "").length;
  const boxRef = useRef<HTMLTextAreaElement | null>(null);

  // **ويُقاس بعد كلّ تبدّل** — ولو جاء النصُّ من غير لوحة المفاتيح
  // (تعبئةٌ من عنوانٍ محفوظ، أو مسحٌ بعد إرسال).
  useEffect(() => {
    const el = boxRef.current;
    if (!autoGrow || !el) return;
    el.style.height = "auto";
    el.style.height = `${el.scrollHeight}px`;
  }, [autoGrow, props.value]);

  return (
    <div>
      {label && (
        <label htmlFor={id} className="mb-1.5 block text-sm font-medium text-ink">
          {label}
        </label>
      )}
      <textarea
        id={id}
        ref={boxRef}
        rows={props.rows ?? (autoGrow ? 1 : 3)}
        {...props}
        className={`w-full rounded-control border bg-field px-3 py-2.5 text-sm text-ink outline-none transition-colors placeholder:text-ink-dim focus:border-primary ${
          error ? "border-danger" : "border-line hover:border-line-soft"
        } ${autoGrow ? "resize-none overflow-y-auto" : ""} ${className}`}
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
        <span className="absolute inset-0 rounded-full border border-line bg-surface transition-colors peer-checked:border-primary peer-focus-visible:ring-2 peer-focus-visible:ring-primary-edge" />
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
  /* **والتسميةُ عقدةٌ لا نصّ** — تسميةُ الموافقة تحمل رابطَي الشروط
     والخصوصيّة داخلَها، **ورابطٌ داخل نصٍّ لا يُكتب حرفاً.** */
}: Omit<React.InputHTMLAttributes<HTMLInputElement>, "type"> & { label: React.ReactNode }) {
  return (
    <label htmlFor={id} className={`group flex cursor-pointer items-center gap-2 text-sm ${className}`}>
      <span className="relative flex h-[18px] w-[18px] shrink-0 items-center justify-center">
        <input
          id={id}
          type="checkbox"
          {...props}
          className="peer h-full w-full cursor-pointer appearance-none rounded-[5px] border border-line bg-surface transition-colors checked:border-primary checked:bg-primary focus-visible:ring-2 focus-visible:ring-primary-edge focus-visible:outline-none"
        />
        {/* **وعلامةُ الصحّ داكنةٌ لا بيضاء.**

            (شهده المالك ٢٠٢٦-٠٨-٠٦: «هذا على ما يبدو لونٌ فضّيّ».)

            كانت `text-on-solid` (أبيض) على `bg-primary` — **وتباينُهما ١٫٨١**،
            فتذوب العلامةُ في مربّعها **فيُقرأ المربّعُ لوحاً فضّيّاً فارغاً**
            لا صندوقاً محدَّداً.

            **و`on-bright` نظيرُها للتعبئة الساطعة** — تباينُها ٩٫٢٦: **وهو
            التوكنُ نفسُه الذي يحمله الزرُّ الأساسيُّ وبطاقةُ الرصيد**، فلا
            يُقرَّر هنا لونٌ ثانٍ. */}
        <IconCheck
          size={12}
          strokeWidth={3.5}
          className="pointer-events-none absolute text-on-bright opacity-0 transition-opacity peer-checked:opacity-100"
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
          className={`figure h-13 w-11 rounded-control border bg-field text-center font-mono text-ink outline-none transition-all sm:w-12 ${ d.trim() ? "border-primary bg-primary-tint text-primary" : "border-line hover:border-line-soft" } focus:border-primary`}
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
  /* **وبلا تسميةٍ لا غلاف** — كان يُغلَّف دائماً بـ`div`، **فالقائمةُ في
     شريطٍ علويٍّ تكسر صفَّه**: كتلةٌ تأخذ سطرَها بين حبّاتٍ متجاورة.

     **فبقيت قائمةٌ واحدةٌ في المشروع خارجَ المكوّن** (شريطُ بوّابة المتجر)
     تبني نفسَها بيدها — **ومن لم يجد المركزَ يصلح لموضعه بنى بجانبه.**
     (٢٠٢٦-٠٨-٠٨.) */
  const field = (
    <select
      id={id}
      {...props}
      className={`w-full rounded-control border border-line bg-field px-3 py-2 text-sm outline-none focus:border-primary ${className}`}
    >
      {children}
    </select>
  );
  if (!label) return field;
  return (
    <div>
      <label htmlFor={id} className="mb-1 block text-sm font-medium">
        {label}
      </label>
      {field}
    </div>
  );
}

// ---------- Badge ----------

/* **والنغماتُ صبغةٌ ونصٌّ من الدلالة نفسِها** — كانت `primary` وحدَها خارجَ
   اللغة (`bg-primary-tint text-primary`: تعبئةٌ مصمتةٌ ودرجةٌ لا دلالة)،
   **فتُقرأ خمسُ شاراتٍ أربعاً وواحدةً غريبة.** (طلبُ المالك ٢٠٢٦-٠٨-٠٧.)

   **وزيدت `accent` و`info`** لأنّ من لم يجد نغمتَه بنى شارتَه بيده — **وهو
   أصلُ الخمسِ مقاساتٍ التي وجدها الجرد.** */
const badgeVariants = {
  neutral: "bg-ink-faint text-ink-muted",
  primary: "bg-primary-tint text-primary",
  accent: "bg-accent-tint text-accent",
  success: "bg-success-tint text-success",
  warning: "bg-warning-tint text-warning",
  danger: "bg-danger-tint text-danger",
  info: "bg-info-tint text-info",
  violet: "bg-violet-tint text-violet",
} as const;

export function Badge({
  variant = "neutral",
  children,
  className = "",
  dir,
}: {
  variant?: keyof typeof badgeVariants;
  children: ReactNode;
  className?: string;
  /** للرموز اللاتينيّة (كودُ مندوبٍ مثلاً) — **تُقلب في سياقٍ عربيٍّ بدونه.** */
  dir?: "ltr" | "rtl";
}) {
  return (
    <span
      dir={dir}
      className={`inline-flex items-center rounded-badge px-2.5 py-0.5 text-xs font-medium ${badgeVariants[variant]} ${className}`}
    >
      {children}
    </span>
  );
}

// ---------- IconTile ----------

/* **وقرصُ الأيقونة شكلٌ ثالثٌ لا شارةٌ ولا فقّاعة**: مربّعٌ ملوّنٌ يجلس فيه
   رمزٌ وحدَه — رأسُ بطاقةِ متجر، وقفلُ شاشةٍ مقفلة، ورمزُ نوعِ تقرير.

   **وكان عشرةَ مواضعَ بستّةِ مقاساتٍ وثلاثةِ أنصافِ أقطار**: `h-9` و`h-10`
   و`h-11` و`h-12` و`h-14`، و`rounded-badge` و`rounded-card` و`rounded-control`
   — **للشيء نفسِه.** فتُفتح شاشتان فيُقرأ قرصاهما شيئين.
   (طلبُ المالك ٢٠٢٦-٠٨-٠٧: مطاردةُ ما هو خارجَ المركز.) */
const tileTones = {
  primary: "bg-primary-tint text-primary",
  accent: "bg-accent-tint text-accent",
  success: "bg-success-tint text-success",
  warning: "bg-warning-tint text-warning",
  danger: "bg-danger-tint text-danger",
  info: "bg-info-tint text-info",
  violet: "bg-violet-tint text-violet",
} as const;

const tileSizes = { sm: "h-9 w-9", md: "h-11 w-11", lg: "h-14 w-14" } as const;

export function IconTile({
  tone = "primary",
  size = "md",
  children,
  className = "",
}: {
  tone?: keyof typeof tileTones;
  size?: keyof typeof tileSizes;
  children: ReactNode;
  className?: string;
}) {
  return (
    <span
      className={`inline-flex shrink-0 items-center justify-center rounded-card ${tileSizes[size]} ${tileTones[tone]} ${className}`}
    >
      {children}
    </span>
  );
}

// ---------- CountBadge ----------

/* **وفقّاعةُ العدد ليست شارةَ نصّ.** الشارةُ تحمل كلمةً فتتّسع لها، والفقّاعةُ
   تحمل رقماً فتبقى دائرةً مهما كان.

   **وكانت خمساً تفرّقت**: ثلاثةُ مقاساتٍ (`h-4` · `h-5` · بلا ارتفاع)
   وثلاثُ إزاحاتٍ **وجهتان متعاكستان** — `-start` فوق الجرس في الشريط
   و`-end` فوق أيقونة الجوّال. **وهما الفكرةُ نفسُها.**
   (طلبُ المالك ٢٠٢٦-٠٨-٠٧: مطاردةُ ما هو خارجَ المركز.)

   **والنشطُ نبرةٌ مصمتةٌ لا صبغة**: كانت فقّاعةُ البطاقة النشطة
   `bg-primary-tint` **فوق بطاقةٍ نشطةٍ `bg-primary-tint`** — أي فقّاعةٌ
   لا تُرى أصلاً. */
/* **والعدّادُ بالنبرة بنصٍّ داكن** — لا أبيض: الأبيضُ على النبرة ١٫٣٠
   **يذوب**، والداكنُ ١٣٫٩٢. **والجرسُ يبقى أبيضَ كما هو.**
   (قرارُ المالك ٢٠٢٦-٠٨-٠٣: «العدّاد فقط وليس الجرس».) */
const countTones = {
  accent: "bg-accent text-on-bright",
  danger: "bg-danger-solid text-on-solid",
} as const;

export function CountBadge({
  count,
  tone = "accent",
  float = false,
  on = true,
  max = 99,
  className = "",
}: {
  count: number;
  tone?: keyof typeof countTones;
  /** تطفو فوق أيقونة — والأبُ يحتاج `relative`. */
  float?: boolean;
  /** حين تسكن صفّاً قابلاً للاختيار: الخامدةُ رماديّةٌ والنشطةُ بالنبرة. */
  on?: boolean;
  max?: number;
  className?: string;
}) {
  const paint = on ? countTones[tone] : "bg-ink-faint text-ink-muted";
  return (
    <span
      dir="ltr"
      className={
        /* ══════════════════════════════════════════════════════════
           **والعدّادُ يجاور الأيقونةَ لا يجلس فوقها**
           ══════════════════════════════════════════════════════════

           (شهده المالك ٢٠٢٦-٠٨-١١: «شوف الإشعارات شلون الأيقونةُ
            مقصوصة».)

           **وقِيس**: زرُّ الجرس ٣١ بكسلاً والأيقونةُ ١٩، والعدّادُ ٢٠
           مزاحٌ ٦ — **فيغطّي ثمانيةَ بكسلاتٍ منها، أي ٤٢٪.** فتُرى
           الأيقونةُ ناقصةً لا مغطّاة، **والعينُ تقرؤها عطباً في الرسم.**

           **فصُغّر إلى ١٦ وقلّت إزاحتُه** — التغطيةُ ٣١٪، **وحلقةٌ بلون
           السطح تفصله عمّا تحته** فيُقرأ شيئاً فوق شيءٍ لا قطعاً فيه.

           **ورقمٌ من خانتين يتّسع**: `min-w` لا عرضٌ ثابت. */
        (float
          ? "absolute -top-1 -end-1 flex h-4 min-w-4 items-center justify-center px-1 ring-2 ring-surface"
          : "inline-flex shrink-0 items-center px-1.5 py-0.5") +
        ` rounded-badge text-2xs font-bold tabular-nums ${paint} ${className}`
      }
    >
      {count > max ? `${fmtNum(max)}+` : fmtNum(count)}
    </span>
  );
}

// ---------- Modal ----------

const modalSizes = {
  md: "max-w-md",
  lg: "max-w-2xl",
  xl: "max-w-4xl",
  /* **ونموذجٌ فيه خريطةٌ يحتاج عرضاً لا طولاً.**
     (شكوى المالك ٢٠٢٦-٠٨-٠٨: «تعديل المتجر فورم طول بسكرول مزعج».)
     **الطولُ يُسكرَل والعرضُ لا** — فالشاشةُ عريضةٌ وكان النموذجُ يضيّق
     نفسَه إلى ٨٩٦ ثمّ يمتدّ إلى ١١٦٤ طولاً. */
  "2xl": "max-w-6xl",
} as const;

/**
 * **والطبقةُ تُرسم في جذر المستند لا حيث كُتبت.**
 *
 * (شكوى المالك ٢٠٢٦-٠٨-٠٩: «رسائل الشكوى لساتها تحت الكروت والعناصر، لازم
 *  تكون فوقهن».)
 *
 * # لماذا لم يكفِ `z-50`
 *
 * **`z-index` لا يُقارَن إلّا بين إخوةٍ في سياقِ تكديسٍ واحد.** ونافذةُ
 * الشكوى مكتوبةٌ **داخل بطاقة الطلب** — والبطاقةُ عليها `surface`، **وفيه
 * `backdrop-filter`.**
 *
 * **وكلُّ عنصرٍ له `backdrop-filter` أو `transform` أو `filter` يُنشئ سياقَ
 * تكديسٍ جديداً** — بل ويصير **الكتلةَ الحاوية لأحفاده `fixed`.** فالنافذةُ
 * لا تُقاس بالشاشة بل بالبطاقة، **و`z-50` فيها يزاحم إخوتَها داخلَ البطاقة
 * لا شريطَ الصفحة ولا البطاقاتِ الأخرى.**
 *
 * **فتظهر تحت ما هو فوقها في الشجرة** مهما رُفع رقمُها — وهو ما رآه المالك:
 * لوحٌ معتمٌ تقطعه عناصرُ من فوقُ ومن تحت.
 *
 * # ورفعُ الرقم ليس علاجاً
 *
 * **`z-[9999]` داخلَ سياقٍ محبوسٍ يبقى محبوساً** — يعلو على إخوته فقط.
 * **والعلاجُ أن تخرج من الشجرة**: تُرسم في `document.body` فتصير أختاً لكلّ
 * شيءٍ في الصفحة، **وعندها وحدَها يعني `z-50` ما يقوله.**
 *
 * **والحالةُ تبقى حيث كُتبت**: البوّابةُ تنقل الرسمَ لا الشجرةَ المنطقيّة —
 * فالأحداثُ والسياقاتُ تصعد كما كانت، **ولا يتغيّر شيءٌ في الشيفرة التي
 * تستعملها.**
 *
 * **وبعد التركيب لا قبله**: `document` لا وجودَ له في الخادم.
 */
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
  const mounted = useMounted();

  useEffect(() => {
    if (!open) return;
    const onKey = (e: KeyboardEvent) => e.key === "Escape" && onClose();
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [open, onClose]);

  if (!open || !mounted) return null;
  return createPortal(
    <div
      className="fixed inset-0 z-50 flex items-center justify-center scrim p-4"
      onClick={onClose}
    >
      <div
        role="dialog"
        aria-modal="true"
        className={`max-h-[90vh] w-full overflow-y-auto surface-modal p-6 ${modalSizes[size]}`}
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
          <h2 className="heading-card">{title}</h2>
          <button
            type="button"
            onClick={onClose}
            aria-label={m.common.close}
            title={m.common.close}
            className="taparea -me-1.5 -mt-1 flex h-8 w-8 shrink-0 items-center justify-center rounded-control text-ink-muted transition-colors hover:bg-row-hover hover:text-ink"
          >
            <IconClose size={18} />
          </button>
        </div>
        {children}
      </div>
    </div>,
    document.body,
  );
}

/** **مركّبٌ بعد؟** — `document` لا وجودَ له في الخادم، والبوّابةُ تحتاجه. */
function useMounted() {
  const [on, setOn] = useState(false);
  useEffect(() => setOn(true), []);
  return on;
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
    /* **وقسمُ النموذج لوحٌ لا فراغ.**

       (كشفه جردُ السائق ٢٠٢٦-٠٨-٠٦: وصفُ «عناويني» عند **٤٫٠٤** — لأنّ
        القسمَ كان `<section>` عارياً على الخلفيّة المتدرّجة.)

       **وكان يصحّ يومَ كان كرتُ المحتوى يلفّ الصفحةَ كلَّها** — فلمّا ذهب
       بقرار المالك **انكشف كلُّ ما كان يستظلّ به.**

       **وهو أثاثُ النموذج**: عنوانٌ وحدٌّ وحقول — **ولوحٌ حوله يجمعها ويُقرِئ
       ما بينها.** */
    <section className="surface p-4">
      <h3 className="mb-3 flex items-center gap-1.5 border-b border-line-soft pb-2 text-sm font-bold text-primary-dark">
        {icon && <span className="[&>svg]:h-4 [&>svg]:w-4">{icon}</span>}
        {title}
      </h3>
      {children}
    </section>
  );
}
