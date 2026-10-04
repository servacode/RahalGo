"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **قائمةُ أفعالٍ منسدلة — «إجراءات ▾» و«⋯» و«إضافة ▾»** (قسمُ الحسابات، ٢٠٢٦-١٠-٠٤)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **سبعةُ أزرارٍ متلاصقةٍ بألوانٍ مختلفة تنكسر على الجوّال** — فيبقى ظاهراً ما يُفعل
 * كلَّ يوم، **والباقي في قائمةٍ تُفتح.** والخطرُ بالأحمر في آخرها بحدٍّ يفصله.
 *
 * **وتُرسم في جذر المستند** (بوّابة) — كقائمة الحساب: حاويةٌ تقصّ أو عليها
 * `backdrop-filter` تأسر الثابتَ داخلها فلا يُرى منه شيء.
 */

import { useCallback, useEffect, useLayoutEffect, useRef, useState, type ComponentType, type ReactNode } from "react";
import { createPortal } from "react-dom";
import { IconChevronDown } from "./icons";

export interface ActionMenuItem {
  key: string;
  label: ReactNode;
  icon?: ComponentType<{ size?: number; className?: string }>;
  onSelect: () => void;
  /** **فعلٌ خطِر** — بالأحمر وبحدٍّ فوقه. */
  danger?: boolean;
  disabled?: boolean;
  /** **سطرٌ صغيرٌ تحت الاسم** — يقول ما سيقع. */
  hint?: ReactNode;
}

export function ActionMenu({
  label,
  icon: Icon,
  items,
  variant = "secondary",
  ariaLabel,
  width = 240,
}: {
  label: ReactNode;
  icon?: ComponentType<{ size?: number; className?: string }>;
  items: readonly ActionMenuItem[];
  variant?: "primary" | "secondary" | "ghost";
  ariaLabel?: string;
  width?: number;
}) {
  const [open, setOpen] = useState(false);
  const [at, setAt] = useState({ top: 0, left: 0 });
  const btn = useRef<HTMLButtonElement>(null);
  const panel = useRef<HTMLDivElement>(null);

  const place = useCallback(() => {
    const r = btn.current?.getBoundingClientRect();
    if (!r) return;
    const vw = document.documentElement.clientWidth;
    const vh = document.documentElement.clientHeight;
    const left = Math.min(Math.max(r.right - width, 8), vw - width - 8);
    const h = panel.current?.offsetHeight ?? 0;
    const below = r.bottom + 6;
    const top = h && below + h > vh - 8 ? Math.max(8, r.top - h - 6) : below;
    setAt({ top, left });
  }, [width]);

  useLayoutEffect(() => {
    if (open) place();
  }, [open, place]);

  useEffect(() => {
    if (!open) return;
    const away = (e: PointerEvent) => {
      const t = e.target as Node;
      if (btn.current?.contains(t) || panel.current?.contains(t)) return;
      setOpen(false);
    };
    const esc = (e: KeyboardEvent) => e.key === "Escape" && setOpen(false);
    document.addEventListener("pointerdown", away);
    document.addEventListener("keydown", esc);
    window.addEventListener("scroll", place, true);
    window.addEventListener("resize", place);
    return () => {
      document.removeEventListener("pointerdown", away);
      document.removeEventListener("keydown", esc);
      window.removeEventListener("scroll", place, true);
      window.removeEventListener("resize", place);
    };
  }, [open, place]);

  if (items.length === 0) return null;

  const tone =
    variant === "primary"
      ? "bg-accent text-on-bright hover:opacity-90"
      : variant === "ghost"
        ? "text-ink-muted hover:bg-row-hover hover:text-ink"
        : "border border-line bg-surface text-ink hover:bg-row-hover";

  const safe = items.filter((it) => !it.danger);
  const risky = items.filter((it) => it.danger);

  const row = (it: ActionMenuItem) => {
    const I = it.icon;
    return (
      <button
        key={it.key}
        type="button"
        role="menuitem"
        disabled={it.disabled}
        onClick={() => {
          setOpen(false);
          it.onSelect();
        }}
        className={`flex w-full items-start gap-2.5 px-3 py-2.5 text-start text-sm transition-colors disabled:cursor-not-allowed disabled:opacity-50 ${
          it.danger ? "text-danger hover:bg-danger-tint" : "text-ink hover:bg-row-hover"
        }`}
      >
        {I && <I size={16} className="mt-0.5 shrink-0" />}
        <span className="min-w-0">
          <span className="block">{it.label}</span>
          {it.hint && <span className="mt-0.5 block text-xs text-ink-muted">{it.hint}</span>}
        </span>
      </button>
    );
  };

  return (
    <>
      <button
        ref={btn}
        type="button"
        aria-haspopup="menu"
        aria-expanded={open}
        aria-label={ariaLabel}
        onClick={() => setOpen((v) => !v)}
        className={`inline-flex items-center gap-1.5 whitespace-nowrap rounded-control px-2.5 py-2 text-sm font-medium transition-colors ${tone}`}
      >
        {Icon && <Icon size={15} />}
        {label}
        <IconChevronDown size={14} className={`transition-transform ${open ? "rotate-180" : ""}`} />
      </button>
      {open &&
        typeof document !== "undefined" &&
        createPortal(
          <div
            ref={panel}
            role="menu"
            style={{ top: at.top, left: at.left, width }}
            className="fixed z-[80] max-h-[70vh] overflow-y-auto surface-raised elev-3 py-1"
          >
            {safe.map(row)}
            {risky.length > 0 && safe.length > 0 && <div className="my-1 border-t border-line-soft" />}
            {risky.map(row)}
          </div>,
          document.body,
        )}
    </>
  );
}
