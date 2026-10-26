"use client";

import type { ComponentType, ReactNode } from "react";

type IconT = ComponentType<{ size?: number; className?: string }>;

export interface HeroTab<K extends string> {
  key: K;
  label: string;
  icon?: IconT;
  /** عددٌ صغيرٌ بجانب التبويب — وفارغُه لا يُرسم. */
  count?: number | null;
  /** **عددٌ يحتاج انتباهاً** — يُلوَّن برتقاليّاً. */
  warn?: boolean;
}

/**
 * **رأسُ الصفحة الموحّد في اللوحة** (قرارُ المالك ٢٠٢٦-١٠-٢٦: «التصميم الاحترافي» للسوق وطلبات الانضمام
 * والتوسّع والمشاكل والطوارئ) — بطاقةٌ واحدة: أيقونةٌ في دائرة، وعنوانٌ وسطرُ شرح، وأفعالٌ على الطرف،
 * ومحتوى إضافيٌّ اختياريّ (أرقام)، **وتحتها التبويباتُ بسطرٍ واحدٍ بأيقوناتها وعدِّها.**
 */
export function PageHero<K extends string>({
  icon: Icon,
  title,
  subtitle,
  actions,
  children,
  tabs,
  value,
  onChange,
}: {
  icon: IconT;
  title: string;
  subtitle?: ReactNode;
  actions?: ReactNode;
  children?: ReactNode;
  tabs?: HeroTab<K>[];
  value?: K;
  onChange?: (k: K) => void;
}) {
  return (
    <section className="surface mb-4 overflow-hidden">
      <div className="flex flex-wrap items-center justify-between gap-3 p-4">
        <div className="flex min-w-0 items-center gap-3">
          <span className="inline-flex h-11 w-11 shrink-0 items-center justify-center rounded-full bg-accent text-on-bright">
            <Icon size={22} />
          </span>
          <div className="min-w-0">
            <h1 className="heading-page">{title}</h1>
            {subtitle && <p className="text-xs text-ink-muted">{subtitle}</p>}
          </div>
        </div>
        {actions && <div className="flex shrink-0 items-center gap-2">{actions}</div>}
      </div>
      {children && <div className="px-4 pb-4">{children}</div>}
      {tabs && tabs.length > 0 && (
        <nav className="flex gap-1 overflow-x-auto border-t border-line-soft px-2" role="tablist">
          {tabs.map(({ key, label, icon: TabIcon, count, warn }) => {
            const on = value === key;
            return (
              <button
                key={key}
                type="button"
                role="tab"
                aria-selected={on}
                onClick={() => onChange?.(key)}
                className={`relative inline-flex shrink-0 items-center gap-2 px-3 py-3 text-sm font-bold transition-colors ${
                  on ? "text-ink" : "text-ink-muted hover:text-ink"
                }`}
              >
                {TabIcon && <TabIcon size={16} />}
                {label}
                {count != null && (
                  <span
                    className={`min-w-[1.5rem] rounded-full px-1.5 py-0.5 text-center text-2xs tabular-nums ${
                      warn && count > 0
                        ? "bg-warning-solid text-on-solid"
                        : on
                          ? "bg-accent text-on-bright"
                          : "bg-field text-ink-muted"
                    }`}
                  >
                    {count}
                  </span>
                )}
                {on && <span className="absolute inset-x-2 bottom-0 h-0.5 rounded-full bg-accent" />}
              </button>
            );
          })}
        </nav>
      )}
    </section>
  );
}
