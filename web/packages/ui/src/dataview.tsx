"use client";

/**
 * نظام العرض المزدوج المركزي: جدول ↔ بطاقات.
 * تعريف أعمدة واحد يغذي الوضعين، والمبدّل يحفظ تفضيل كل مستخدم لكل شاشة
 * (localStorage) — كل شاشات القوائم في المنصة تستخدم هذا النظام حصراً.
 */

import { useEffect, useState, type ReactNode } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { IconList, IconGrid } from "./icons";

const L = getMessages(defaultLocale).common;

export type ViewMode = "table" | "cards";

/**
 * **حدُّ الشاشة الذي لا يحتمل جدولاً.**
 *
 * جدولٌ بخمسة أعمدةٍ يحتاج نحوَ سبعمئة بكسل. **وتحتها ينزلق أفقيّاً**،
 * **والقراءةُ الأفقيّةُ في واجهةٍ عربيةٍ أسوأ**: العينُ ترجع إلى اليمين فلا
 * تجد أوّلَ السطر.
 */
const TABLE_MIN = 768;

/**
 * useViewMode تفضيلُ العرض محفوظٌ لكلّ شاشة — **والجوّالُ يغلب التفضيل.**
 *
 * # المسألة
 *
 * الافتراضُ `table` والتفضيلُ يُحفظ. **فمن فتح اللوحةَ على جوّالٍ يرى جدولاً
 * ينزلق أفقيّاً**، ومن اختار «جدول» على لابتوبه **يحمل اختيارَه إلى هاتفه**
 * لأنّ التفضيلَ في `localStorage` لا في الجهاز.
 *
 * # ولا يُبدَّل المحفوظ
 *
 * **الغلبةُ في العرض لا في التخزين**: من فضّل الجدولَ على مكتبه يجده كما
 * تركه، **وهاتفُه يعرض بطاقاتٍ ولا يمحو تفضيلَه.**
 *
 * # والقياسُ بعد التركيب لا قبله
 *
 * `window` لا وجودَ له في الخادم — **وقراءتُه في أوّل رسمٍ تكسر الترطيب.**
 * **والهيكلُ العظميُّ يشتري الوقت**: الشاشةُ تعرض هيكلاً حتّى تصل البيانات،
 * **وقد قِيس العرضُ قبلها.**
 */
export function useViewMode(screenId: string, fallback: ViewMode = "table") {
  const [view, setView] = useState<ViewMode>(fallback);
  const [narrow, setNarrow] = useState(false);

  useEffect(() => {
    const saved = localStorage.getItem(`rahalgo_view:${screenId}`);
    if (saved === "table" || saved === "cards") setView(saved);
  }, [screenId]);

  useEffect(() => {
    const mq = window.matchMedia(`(max-width: ${TABLE_MIN - 1}px)`);
    const sync = () => setNarrow(mq.matches);
    sync();
    // **ويتابع الدوران**: من قلب هاتفَه أفقيّاً يتّسع فيعود الجدول.
    mq.addEventListener("change", sync);
    return () => mq.removeEventListener("change", sync);
  }, []);

  const change = (v: ViewMode) => {
    setView(v);
    localStorage.setItem(`rahalgo_view:${screenId}`, v);
  };
  // **والمعروضُ غيرُ المحفوظ** — الأوّلُ يغلبه الجهاز، والثاني يخصّ صاحبَه.
  return [narrow ? ("cards" as ViewMode) : view, change, { narrow }] as const;
}

export function ViewToggle({
  view,
  onChange,
  tableLabel,
  cardsLabel,
}: {
  view: ViewMode;
  onChange: (v: ViewMode) => void;
  tableLabel: string;
  cardsLabel: string;
}) {
  const base =
    "flex items-center gap-1.5 rounded-control px-3 py-1.5 text-sm transition-colors";
  /* **والمختارةُ كحبّةِ الترشيح** — كانت `bg-surface text-primary-dark`:
     سطحٌ على سطحٍ لا يكاد يُرى، **ودرجةٌ لا دلالة.** (قرارُ المالك
     ٢٠٢٦-٠٨-٠٧: «الزرّ لسّا آخذٌ ألواناً غيرَ الثيم».) */
  const active = "bg-primary font-medium text-on-bright elev-1";
  const idle = "text-ink-muted hover:text-ink";
  return (
    /* **ويُخفى حيث لا يفعل شيئاً.**

       تحت ٧٦٨ بكسلاً يُعرض بطاقاتٍ حتماً، **وزرُّ «جدول» يُضغط ولا يقع
       شيء** — فيُقرأ عطباً. **وزرٌّ معطَّلٌ يُسأل عن سببه، وزرٌّ غائبٌ لا
       يُفتقد.** */
    <div
      role="group"
      /* **والحاضنُ زجاجٌ لا لوحٌ معتم** — `bg-field` صندوقٌ أسودُ فوق تدرّج. */
      className="surface hidden rounded-control p-1 md:flex"
    >
      <button
        type="button"
        aria-pressed={view === "table"}
        onClick={() => onChange("table")}
        className={`${base} ${view === "table" ? active : idle}`}
      >
        <TableIcon />
        {tableLabel}
      </button>
      <button
        type="button"
        aria-pressed={view === "cards"}
        onClick={() => onChange("cards")}
        className={`${base} ${view === "cards" ? active : idle}`}
      >
        <CardsIcon />
        {cardsLabel}
      </button>
    </div>
  );
}

export interface DataColumn<T> {
  /**
   * حقلٌ **يأخذ عرض البطاقة كاملاً** بدل صفّ «تسمية ← قيمة».
   *
   * قوائمُ الأصناف والملاحظاتُ الطويلة تُحشَر في العمود الأيسر الضيّق فتتكسّر
   * كلماتُها ويصعب مسحُها بالعين. **وما يُقرأ سطراً سطراً لا يُوضَع في خانة.**
   */
  block?: boolean;
  id: string;
  header: string;
  cell: (item: T) => ReactNode;
  /**
   * **يُخفى الحقلُ لصفٍّ لا معنى له فيه.**
   *
   * «إثباتُ التسليم» في طلبٍ لم يُقبل بعد **سطرٌ فارغٌ يُسأل عنه ولا جواب**،
   * و«سببُ الإنهاء» في طلبٍ يمشي كذلك. **وحقلٌ يظهر فارغاً دائماً يُتعلَّم
   * تجاهلُه**، ثمّ يمتلئ يوماً فلا يُنظر إليه.
   *
   * **وفي وضع البطاقات وحدَه**: الجدولُ أعمدتُه ثابتةٌ لكلّ الصفوف، **وإخفاءُ
   * عمودٍ لصفٍّ يُزحزح ما بعده.**
   */
  hide?: (item: T) => boolean;
  /**
   * **خليّةٌ مختصرةٌ للجدول** — حين لا يصلح شكلُ البطاقة في خانة.
   *
   * فاتورةٌ من عشرة سطورٍ تُقرأ في بطاقةٍ وتُفسد صفَّ جدول: **ترتفع الصفوفُ
   * وتتباين أطوالُها، فيُقرأ الجدولُ عشوائياً.** فيُعرض في الجدول زرٌّ يفتح
   * ما يلزم، **وفي البطاقة يُعرض كاملاً.**
   *
   * (ملاحظةُ المالك ٢٠٢٦-٠٨-٠٤: «بالجدول يكفي أن يكون زرٌّ اسمُه الفاتورة
   * يعرض بنافذةٍ منبثقة ليبقى الشكلُ بصرياً بحالٍ احترافية».)
   */
  tableCell?: (item: T) => ReactNode;
  /**
   * **حقلٌ لوضعٍ دون آخر.**
   *
   * الحالةُ في البطاقة شارةٌ في الترويسة مقابلَ الرقم — **وفي الجدول عمودٌ له
   * رأسٌ يُقرأ.** ولو عُرضت في الوضعين بالتعريف نفسِه لَظهرت مرّتين في
   * البطاقة، **أو غاب رأسُها في الجدول فيُقرأ العمودُ بلا اسم.**
   */
  only?: "table" | "cards";
  /**
   * **حقلٌ بلا تسمية في البطاقة** — محتواه يقول ما هو.
   *
   * (قرارُ المالك ٢٠٢٦-٠٨-١٢: «بدل الزبون فوراً نحطّ اسم الزبون، لا داعي
   *  لكلمة الزبون».)
   *
   * **واسمٌ ورقمُ هاتفٍ لا يُسأل عمّن هما**: التسميةُ تشغل نصفَ السطر
   * **وتقول ما هو ظاهر.** ويبقى رأسُ العمود في الجدول — **هناك لا يقول
   * المحتوى ما هو**: عمودٌ بلا رأسٍ يُقرأ بالتخمين.
   */
  noLabel?: boolean;
  /** primary: يظهر كعنوان البطاقة في وضع البطاقات */
  primary?: boolean;
  /** أيقونة معبرة للحقل — تظهر برأس العمود وفي تسمية حقل البطاقة */
  icon?: ReactNode;
}

function FieldLabel({ icon, text }: { icon?: ReactNode; text: string }) {
  return (
    <span className="inline-flex items-center gap-1.5">
      {icon && (
        <span className="text-ink-dim [&>svg]:h-4 [&>svg]:w-4">
          {icon}
        </span>
      )}
      {text}
    </span>
  );
}

export function DataView<T>({
  items,
  getKey,
  columns,
  actions,
  empty,
  view,
  onRowClick,
  card,
}: {
  items: T[];
  getKey: (item: T) => string;
  columns: DataColumn<T>[];
  actions?: (item: T) => ReactNode;
  empty: string;
  view: ViewMode;
  onRowClick?: (item: T) => void;
  /**
   * **كرتٌ خاصٌّ بدل الكرت المبنيّ من الأعمدة.**
   *
   * (طلبُ المالك ٢٠٢٦-٠٨-٠٨: البلاغات كروتٌ وجداول «مثل باقي المشروع».)
   *
   * الكرتُ العامُّ يُبنى من الأعمدة — وهو الصحيح لجدولِ حساباتٍ أو طلبات.
   * **والبلاغُ له كرتُه المعروف** (`ComplaintCard`) تستعمله لوحاتُ المتجر
   * والسائق والمندوب والزبون. **فلو بنته الإدارةُ من أعمدتها لَقُرئ شيئاً
   * آخر** — وهو الشيءُ نفسُه.
   *
   * **والجدولُ يبقى من الأعمدة**: الإدارةُ تمسح مئةَ بلاغٍ بالعين، والكرتُ
   * لمن يقرأ واحداً.
   */
  card?: (item: T) => ReactNode;
}) {
  if (items.length === 0) {
    return (
      <div className="surface p-10 text-center text-ink-muted">
        {empty}
      </div>
    );
  }

  if (view === "cards" && card) {
    return (
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-3">
        {items.map((item) => (
          <div key={getKey(item)}>{card(item)}</div>
        ))}
      </div>
    );
  }

  if (view === "cards") {
    const forCards = columns.filter((c) => c.only !== "table");
    const primaries = forCards.filter((c) => c.primary);
    const rest = forCards.filter((c) => !c.primary && !c.block);
    const blocks = forCards.filter((c) => !c.primary && c.block);
    return (
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
        {items.map((item) => (
          <div
            key={getKey(item)}
            onClick={onRowClick ? () => onRowClick(item) : undefined}
            className={`flex flex-col surface p-4 transition-shadow hover:elev-2 ${onRowClick ? "cursor-pointer" : ""}`}
          >
            {/* ══════════════════════════════════════════════════════
                **والترويسةُ صفٌّ لا عمود**
                ══════════════════════════════════════════════════════

                (قرارُ المالك ٢٠٢٦-٠٨-٠٧: «لازم يكون المبلغ محاذاةً لليسار
                 أيضاً — وطبعاً كلُّ شيءٍ ذكرتُه يجب أن يتمّ بشكلٍ مركزيّ».)

                **كانت الأوّليّاتُ مرصوفةً فوق بعضها** — الاسمُ ثمّ الرقمُ
                تحته، وكلاهما ملتصقٌ بالحافّة نفسِها. **فيُقرآن شيئاً واحداً
                من سطرين** لا اسماً وقيمة.

                **فصارا طرفَي سطر**: الاسمُ حيث تبدأ القراءة والرقمُ حيث
                تنتهي — **والعينُ تمسح عموداً من الأرقام في بطاقاتٍ متجاورة
                لأنّها كلَّها على استقامةٍ واحدة.**

                **وما زاد على اثنتين يلتفّ** ولا يزاحم. */}
            <div className="mb-3 flex flex-wrap items-baseline justify-between gap-x-3 gap-y-1 border-b border-line-soft pb-3">
              {primaries.map((c, i) => (
                <div key={c.id} className={i === 0 ? "text-base font-bold" : "text-sm text-ink-muted"}>
                  {c.cell(item)}
                </div>
              ))}
            </div>
            {/* كل حقل سطرٌ مفصول بخطّ خفيف: بلا فاصل تسيح الحقول في كتلة واحدة
                فيُقرأ عنوانٌ مع قيمة جارِه — والبطاقة تُمسح بالعين لا تُدرَس. */}
            <dl className="flex-1 text-sm">
              {/* **ويُرشَّح لكلّ بطاقةٍ على حدة** — حقلٌ لا معنى له في هذا الصفّ
                  لا يُعرض فارغاً فيه. */}
              {rest
                .filter((c) => !c.hide?.(item))
                .map((c, i, shown) => (
                  <div
                    key={c.id}
                    className={`flex items-start justify-between gap-3 py-2 ${
                      i < shown.length - 1 ? "border-b border-line-soft" : ""
                    }`}
                  >
                    {!c.noLabel && (
                      <dt className="shrink-0 text-ink-muted">
                        <FieldLabel icon={c.icon} text={c.header} />
                      </dt>
                    )}
                    {/* **وبلا تسميةٍ يأخذ السطرَ كلَّه** — فيوزّع ما فيه
                        يمينا ويسارا كما يوزّعه الحقلُ المُسمّى. */}
                    <dd className={c.noLabel ? "w-full" : "min-w-0 text-end"}>
                      {c.cell(item)}
                    </dd>
                  </div>
                ))}
            </dl>

            {/* الحقول الطويلة بعرض البطاقة: تسميةٌ فوق ومحتوىً تحتها */}
            {blocks
              .filter((c) => !c.hide?.(item))
              .map((c) => (
                <div key={c.id} className="mt-3 border-t border-line-soft pt-3">
                  <p className="mb-1 text-xs text-ink-muted">
                    <FieldLabel icon={c.icon} text={c.header} />
                  </p>
                  <div className="text-sm">{c.cell(item)}</div>
                </div>
              ))}

            {actions && (
              <div
                onClick={(e) => e.stopPropagation()}
                /* الأزرار تتقاسم السطر ما دامت تتسع، وتنزل سطراً جديداً بدل أن
                   تفيض خارج البطاقة — النصوص العربية تطول ولا تُقصّ. */
                className="mt-3 flex flex-wrap gap-1.5 border-t border-line-soft pt-3 [&_button]:min-w-[6.5rem] [&_button]:flex-1 [&_button]:justify-center [&_button]:!px-2 [&_button]:text-center"
              >
                {actions(item)}
              </div>
            )}
          </div>
        ))}
      </div>
    );
  }

  /**
   * **والإخفاءُ يسري على الجدول أيضاً — بعمودٍ لا بخليّة.**
   *
   * أعمدةُ الجدول ثابتةٌ لكلّ الصفوف: **إخفاءُ خليّةٍ في صفٍّ يُزحزح ما بعدها
   * فينهار الجدول.** فيُخفى **العمودُ كلُّه** حين لا يحتاجه صفٌّ واحدٌ ممّا
   * يُعرض — وذلك عينُ ما تفعله البطاقات، بحدّها الأدنى.
   *
   * **وقاعدةٌ تُطبَّق في وضعٍ وتُنسى في الآخر ليست قاعدة** (ملاحظةُ المالك
   * ٢٠٢٦-٠٨-٠٤): «التعديلاتُ يجب أن تُطبَّق في حالة الجداول أو الكروت بنفس
   * الوقت، لا نطبّق تعديلاتٍ في مكانٍ ونترك الآخر».
   */
  const shown = columns.filter(
    (c) => c.only !== "cards" && (!c.hide || items.some((it) => !c.hide!(it))),
  );

  return (
    <div className="overflow-x-auto surface">
      <table className="w-full text-sm">
        <thead>
          {/* **والرؤوسُ فوق قيمها لا بجانبها.**

              محاذاةٌ إلى الوسط تجعل العمودَ كتلةً واحدةً تُمسح بالعين: **رأسٌ
              وقيمٌ على محورٍ واحد.** وبمحاذاة البدء تتباعد الرؤوسُ عن قيمها
              حين تختلف أطوالُها، **فيُقرأ رأسٌ مع قيمة جارِه.** */}
          <tr className="border-b border-line-soft text-ink-muted">
            {shown.map((c) => (
              <th
                key={c.id}
                className="whitespace-nowrap p-3 text-center align-middle font-medium"
              >
                <span className="inline-flex justify-center">
                  <FieldLabel icon={c.icon} text={c.header} />
                </span>
              </th>
            ))}
            {/* **وعمودُ الأفعال له رأسٌ أيضاً** — عمودٌ بلا اسمٍ يُقرأ زائداً. */}
            {actions && (
              <th className="whitespace-nowrap p-3 text-center align-middle font-medium">
                {L.actions}
              </th>
            )}
          </tr>
        </thead>
        <tbody>
          {items.map((item) => (
            <tr
              key={getKey(item)}
              onClick={onRowClick ? () => onRowClick(item) : undefined}
              className={`border-b border-line-soft last:border-0 hover:bg-row-hover ${onRowClick ? "cursor-pointer" : ""}`}
            >
              {shown.map((c) => (
                // **والصفوفُ متساويةُ الارتفاع، والمحتوى في وسطها.**
                //
                // خليّةٌ تحمل سطراً وأخرى تحمل خمسةً تجعل الصفَّ يتمدّد
                // **والقيمُ تسبح في فراغه**، فيُقرأ الجدولُ عشوائياً.
                <td key={c.id} className="p-3 text-center align-middle">
                  {c.hide?.(item) ? (
                    <span className="text-ink-muted">—</span>
                  ) : (
                    (c.tableCell ?? c.cell)(item)
                  )}
                </td>
              ))}
              {actions && (
                <td className="p-3 align-middle" onClick={(e) => e.stopPropagation()}>
                  <div className="flex flex-nowrap justify-center gap-1.5 whitespace-nowrap">
                    {actions(item)}
                  </div>
                </td>
              )}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

// **الأيقونتان من المركز لا مرسومتين هنا.**
//
// كانتا `<svg>` بالحرف — وهما الوحيدتان في المشروع كلِّه. **وأيقونةٌ تُرسم في
// مكوّنٍ تُرسم ثانيةً في غيره بخطٍّ مختلف**، فتفترق سماكتُها ومقاسُها عن
// أخواتها ولا يلاحظ أحدٌ إلّا حين تُصفّ بجانبها.
function TableIcon() {
  return <IconList size={15} />;
}

function CardsIcon() {
  return <IconGrid size={15} />;
}
