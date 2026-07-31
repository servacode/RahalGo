"use client";

/**
 * مكونات الواجهة المشتركة — تُستخدم في كل تطبيقات الويب حصراً (GROUND-RULES §1.2).
 * كل الأنماط من توكنز الثيم المركزي، وكلها RTL-جاهزة (خصائص منطقية فقط).
 */

import { useEffect, useState, type ReactNode } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { IconView, IconViewOff, IconCheck } from "./icons";

const m = getMessages(defaultLocale);

// ---------- Button ----------

const buttonVariants = {
  primary: "bg-primary text-white hover:bg-primary-dark",
  secondary: "border border-line bg-surface text-ink hover:bg-page",
  danger: "bg-danger text-white hover:bg-danger/90",
  ghost: "text-ink-muted hover:bg-page hover:text-ink",
} as const;

export function Button({
  variant = "primary",
  className = "",
  ...props
}: React.ButtonHTMLAttributes<HTMLButtonElement> & {
  variant?: keyof typeof buttonVariants;
}) {
  return (
    <button
      {...props}
      className={`rounded-control px-4 py-2 text-sm font-medium transition-colors disabled:pointer-events-none disabled:opacity-60 ${buttonVariants[variant]} ${className}`}
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
          } ${icon ? "ps-10" : "ps-3"} ${isPassword ? "pe-10" : "pe-3"} ${className}`}
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
          className="pointer-events-none absolute text-white opacity-0 transition-opacity peer-checked:opacity-100"
        />
      </span>
      <span className="text-ink-muted transition-colors group-hover:text-ink">{label}</span>
    </label>
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
        className={`max-h-[90vh] w-full overflow-y-auto rounded-card border border-line bg-surface p-6 shadow-lg ${modalSizes[size]}`}
        onClick={(e) => e.stopPropagation()}
      >
        <h2 className="mb-4 text-lg font-bold">{title}</h2>
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
