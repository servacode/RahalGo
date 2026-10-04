"use client";

/**
 * **اختيارٌ مع بحث** — قائمةٌ طويلةٌ تُبحث بالاسم بدل أن تُمرَّر.
 *
 * (قرارُ المالك ٢٠٢٦-١٠-٠٤ — سجلُّ الطلبات: «المتجر والسائق قوائمُ اختيارٍ مع
 * بحث».) **ستّةٌ وعشرون متجراً في قائمةٍ منسدلةٍ تُمرَّر بالعين**، وتصير مئةً
 * غداً. فيُكتب حرفان فتبقى ثلاثة.
 *
 * **والخيارُ الأوّلُ «الكلّ» دائماً** (`value=""`) — فلا يُحبَس من اختار.
 */

import { useEffect, useId, useRef, useState } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { IconChevronDown, IconSearch } from "./icons";

const m = getMessages(defaultLocale);

export interface SearchSelectOption {
  value: string;
  label: string;
}

export function SearchSelect({
  value,
  onChange,
  options,
  allLabel,
  ariaLabel,
  className = "",
}: {
  value: string;
  onChange: (value: string) => void;
  options: SearchSelectOption[];
  /** نصُّ «الكلّ» — الخيارُ الفارغ. */
  allLabel: string;
  ariaLabel?: string;
  className?: string;
}) {
  const [open, setOpen] = useState(false);
  const [q, setQ] = useState("");
  const box = useRef<HTMLDivElement | null>(null);
  const listId = useId();

  useEffect(() => {
    if (!open) return;
    const away = (e: MouseEvent) => {
      if (box.current && !box.current.contains(e.target as Node)) setOpen(false);
    };
    const key = (e: KeyboardEvent) => e.key === "Escape" && setOpen(false);
    document.addEventListener("mousedown", away);
    document.addEventListener("keydown", key);
    return () => {
      document.removeEventListener("mousedown", away);
      document.removeEventListener("keydown", key);
    };
  }, [open]);

  const picked = options.find((o) => o.value === value);
  const needle = q.trim().toLowerCase();
  const shown = needle
    ? options.filter((o) => o.label.toLowerCase().includes(needle))
    : options;

  const pick = (v: string) => {
    onChange(v);
    setOpen(false);
    setQ("");
  };

  return (
    <div ref={box} className={`relative ${className}`}>
      <button
        type="button"
        aria-label={ariaLabel ?? allLabel}
        aria-haspopup="listbox"
        aria-expanded={open}
        aria-controls={listId}
        onClick={() => setOpen((v) => !v)}
        className="flex w-full items-center justify-between gap-2 rounded-control border border-line bg-field px-3 py-2 text-start text-sm outline-none focus:border-primary"
      >
        <span className="truncate">{picked ? picked.label : allLabel}</span>
        <IconChevronDown size={14} className="shrink-0 text-ink-muted" />
      </button>
      {open && (
        <div className="surface absolute inset-x-0 top-full z-30 mt-1 min-w-[12rem] p-2 shadow-lift">
          <div className="mb-2 flex items-center gap-2 rounded-control border border-line bg-field px-2">
            <IconSearch size={14} className="shrink-0 text-ink-muted" />
            <input
              autoFocus
              value={q}
              onChange={(e) => setQ(e.target.value)}
              placeholder={m.common.search}
              aria-label={m.common.search}
              className="w-full bg-transparent py-1.5 text-sm outline-none"
            />
          </div>
          <ul id={listId} role="listbox" className="max-h-64 overflow-y-auto">
            {[{ value: "", label: allLabel }, ...shown].map((o) => (
              <li key={o.value || "__all"} role="option" aria-selected={o.value === value}>
                <button
                  type="button"
                  onClick={() => pick(o.value)}
                  className={`w-full rounded-control px-2 py-1.5 text-start text-sm hover:bg-field ${
                    o.value === value ? "font-bold text-accent-text" : ""
                  }`}
                >
                  {o.label}
                </button>
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  );
}
