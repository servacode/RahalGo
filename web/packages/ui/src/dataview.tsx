"use client";

/**
 * نظام العرض المزدوج المركزي: جدول ↔ بطاقات.
 * تعريف أعمدة واحد يغذي الوضعين، والمبدّل يحفظ تفضيل كل مستخدم لكل شاشة
 * (localStorage) — كل شاشات القوائم في المنصة تستخدم هذا النظام حصراً.
 */

import { useEffect, useState, type ReactNode } from "react";

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
  id: string;
  header: string;
  cell: (item: T) => ReactNode;
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
    const rest = columns.filter((c) => !c.primary);
    return (
      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-3">
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
            <dl className="flex-1 space-y-2 text-sm">
              {rest.map((c) => (
                <div key={c.id} className="flex items-start justify-between gap-3">
                  <dt className="shrink-0 text-ink-muted">
                    <FieldLabel icon={c.icon} text={c.header} />
                  </dt>
                  <dd className="text-end">{c.cell(item)}</dd>
                </div>
              ))}
            </dl>
            {actions && (
              <div
                onClick={(e) => e.stopPropagation()}
                className="mt-3 flex flex-wrap justify-end gap-2 border-t border-line pt-3"
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
                  <div className="flex flex-wrap justify-end gap-2">{actions(item)}</div>
                </td>
              )}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function TableIcon() {
  return (
    <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" aria-hidden>
      <path d="M3 6h18M3 12h18M3 18h18" strokeLinecap="round" />
    </svg>
  );
}

function CardsIcon() {
  return (
    <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" aria-hidden>
      <rect x="3" y="3" width="8" height="8" rx="1.5" />
      <rect x="13" y="3" width="8" height="8" rx="1.5" />
      <rect x="3" y="13" width="8" height="8" rx="1.5" />
      <rect x="13" y="13" width="8" height="8" rx="1.5" />
    </svg>
  );
}
