"use client";

/**
 * نظام العرض المزدوج المركزي: جدول ↔ بطاقات.
 * تعريف أعمدة واحد يغذي الوضعين، والمبدّل يحفظ تفضيل كل مستخدم لكل شاشة
 * (localStorage) — كل شاشات القوائم في المنصة تستخدم هذا النظام حصراً.
 */

import { useEffect, useState, type ReactNode } from "react";
import { IconList, IconGrid } from "./icons";

export type ViewMode = "table" | "cards";

/** تفضيل العرض محفوظ لكل شاشة (screenId) على حدة */
export function useViewMode(screenId: string, fallback: ViewMode = "table") {
  const [view, setView] = useState<ViewMode>(fallback);

  useEffect(() => {
    const saved = localStorage.getItem(`rahalgo_view:${screenId}`);
    if (saved === "table" || saved === "cards") setView(saved);
  }, [screenId]);

  const change = (v: ViewMode) => {
    setView(v);
    localStorage.setItem(`rahalgo_view:${screenId}`, v);
  };
  return [view, change] as const;
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
  const base = "flex items-center gap-1.5 rounded-control px-3 py-1.5 text-sm transition-colors";
  const active = "bg-surface font-medium text-primary-dark shadow-sm";
  const idle = "text-ink-muted hover:text-ink";
  return (
    <div role="group" className="flex rounded-control border border-line bg-page p-1">
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
  /** primary: يظهر كعنوان البطاقة في وضع البطاقات */
  primary?: boolean;
  /** أيقونة معبرة للحقل — تظهر برأس العمود وفي تسمية حقل البطاقة */
  icon?: ReactNode;
}

function FieldLabel({ icon, text }: { icon?: ReactNode; text: string }) {
  return (
    <span className="inline-flex items-center gap-1.5">
      {icon && <span className="text-ink-muted/70 [&>svg]:h-4 [&>svg]:w-4">{icon}</span>}
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
}: {
  items: T[];
  getKey: (item: T) => string;
  columns: DataColumn<T>[];
  actions?: (item: T) => ReactNode;
  empty: string;
  view: ViewMode;
  onRowClick?: (item: T) => void;
}) {
  if (items.length === 0) {
    return (
      <div className="rounded-card border border-line bg-surface p-10 text-center text-ink-muted">
        {empty}
      </div>
    );
  }

  if (view === "cards") {
    const primaries = columns.filter((c) => c.primary);
    const rest = columns.filter((c) => !c.primary && !c.block);
    const blocks = columns.filter((c) => !c.primary && c.block);
    return (
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
        {items.map((item) => (
          <div
            key={getKey(item)}
            onClick={onRowClick ? () => onRowClick(item) : undefined}
            className={`flex flex-col rounded-card border border-line bg-surface p-4 transition-shadow hover:shadow-md ${onRowClick ? "cursor-pointer" : ""}`}
          >
            <div className="mb-3 border-b border-line pb-3">
              {primaries.map((c, i) => (
                <div key={c.id} className={i === 0 ? "text-base font-bold" : "mt-0.5 text-sm text-ink-muted"}>
                  {c.cell(item)}
                </div>
              ))}
            </div>
            {/* كل حقل سطرٌ مفصول بخطّ خفيف: بلا فاصل تسيح الحقول في كتلة واحدة
                فيُقرأ عنوانٌ مع قيمة جارِه — والبطاقة تُمسح بالعين لا تُدرَس. */}
            <dl className="flex-1 text-sm">
              {/* **ويُرشَّح لكلّ بطاقةٍ على حدة** — حقلٌ لا معنى له في هذا الصفّ
                  لا يُعرض فارغاً فيه. */}
              {rest.filter((c) => !c.hide?.(item)).map((c, i, shown) => (
                <div
                  key={c.id}
                  className={`flex items-start justify-between gap-3 py-2 ${
                    i < shown.length - 1 ? "border-b border-line/60" : ""
                  }`}
                >
                  <dt className="shrink-0 text-ink-muted">
                    <FieldLabel icon={c.icon} text={c.header} />
                  </dt>
                  <dd className="min-w-0 text-end">{c.cell(item)}</dd>
                </div>
              ))}
            </dl>

            {/* الحقول الطويلة بعرض البطاقة: تسميةٌ فوق ومحتوىً تحتها */}
            {blocks.filter((c) => !c.hide?.(item)).map((c) => (
              <div key={c.id} className="mt-3 border-t border-line/60 pt-3">
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
                className="mt-3 flex flex-wrap gap-1.5 border-t border-line pt-3 [&_button]:min-w-[6.5rem] [&_button]:flex-1 [&_button]:justify-center [&_button]:!px-2 [&_button]:text-center"
              >
                {actions(item)}
              </div>
            )}
          </div>
        ))}
      </div>
    );
  }

  return (
    <div className="overflow-x-auto rounded-card border border-line bg-surface">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-line text-ink-muted">
            {columns.map((c) => (
              <th key={c.id} className="p-3 text-start font-medium">
                <FieldLabel icon={c.icon} text={c.header} />
              </th>
            ))}
            {actions && <th className="p-3" />}
          </tr>
        </thead>
        <tbody>
          {items.map((item) => (
            <tr
              key={getKey(item)}
              onClick={onRowClick ? () => onRowClick(item) : undefined}
              className={`border-b border-line last:border-0 hover:bg-page/60 ${onRowClick ? "cursor-pointer" : ""}`}
            >
              {columns.map((c) => (
                <td key={c.id} className="p-3">
                  {c.cell(item)}
                </td>
              ))}
              {actions && (
                <td className="p-3" onClick={(e) => e.stopPropagation()}>
                  <div className="flex flex-nowrap justify-end gap-1.5 whitespace-nowrap">
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
