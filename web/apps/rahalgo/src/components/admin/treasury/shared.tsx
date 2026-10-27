"use client";

/** **أدواتٌ مشتركةٌ لتبويبات الخزينة** — رقمٌ بإشارته ولونه، وشارةُ حال، ومعادلةٌ، وحالُ فراغ. */

import type { ComponentType, ReactNode } from "react";
import { getMessages, defaultLocale, fmtNum, fmtMoney, fmtLongDate } from "@rahalgo/i18n";

const m = getMessages(defaultLocale);
export const T = m.admin.treasury;

/** **مبلغٌ بإشارته** — الداخلُ أخضر والخارجُ أحمر، وبأرقامٍ غربيّة. */
export function Signed({ value, className = "" }: { value: number; className?: string }) {
  const tone = value > 0 ? "text-success" : value < 0 ? "text-danger" : "text-ink-muted";
  return (
    <span dir="ltr" className={`font-bold tabular-nums ${tone} ${className}`}>
      {value > 0 ? "+" : value < 0 ? "−" : ""}
      {fmtNum(Math.abs(value))}
    </span>
  );
}

/** **مبلغٌ بلا إشارة.** */
export function Amount({ value }: { value: number }) {
  return (
    <span dir="ltr" className="font-bold tabular-nums">
      {fmtNum(value)}
    </span>
  );
}

/** **رسالةُ الخادم أو نصٌّ احتياطيّ.** */
export function sectionLabel(key: string): string {
  const s = T.approvals.sections as Record<string, string>;
  return s[key] ?? s.other ?? key;
}

/**
 * **يومٌ بالعربيّ الواضح** — «26 تشرين الأول 2026» بدل «2026-10-26» (تدقيقُ الخزينة ٢٠٢٦-١٠-٢٧).
 * والمدخلُ يومُ حسابٍ `YYYY-MM-DD` يُقرأ ظهراً بالتوقيت العالميّ فلا ينزلق يوماً بأيّ منطقة.
 */
export function fmtDay(iso: string): string {
  if (!/^\d{4}-\d{2}-\d{2}$/.test(iso)) return iso;
  return fmtLongDate(`${iso}T12:00:00Z`);
}

/** **«يوم الحساب بيبلّش الساعة ٩»** — كي لا يُظنّ «اليوم» من نصف الليل. */
export function dayStartsLine(hour: number | undefined): string {
  return T.overview.dayStarts.replace("{h}", fmtNum(hour ?? 9));
}

/** **يومٌ بعد أيّام** — حسابٌ على `YYYY-MM-DD` بالتوقيت العالميّ (لا منطقةَ متصفّح). */
export function shiftDay(iso: string, days: number): string {
  const d = new Date(`${iso}T12:00:00Z`);
  d.setUTCDate(d.getUTCDate() + days);
  return d.toISOString().slice(0, 10);
}

/**
 * **حالُ الفراغ** — دائرةٌ ملوّنةٌ وعنوانٌ وجملةٌ تقول ما يحدث بعد (التصميمُ الموحّد ٢٠٢٦-١٠-٢٦).
 */
export function EmptyCard({
  icon: Icon,
  title,
  hint,
  tone = "success",
  bare = false,
}: {
  icon: ComponentType<{ size?: number; className?: string }>;
  title: string;
  hint?: string;
  tone?: "success" | "accent" | "muted";
  /** **بلا إطار** — داخل بطاقةٍ قائمة. */
  bare?: boolean;
}) {
  const circle =
    tone === "success" ? "bg-success-solid text-on-solid" : tone === "accent" ? "bg-accent text-on-bright" : "bg-field text-ink-muted";
  return (
    <div className={`flex flex-col items-center gap-3 p-8 text-center ${bare ? "" : "surface"}`}>
      <span className={`inline-flex h-16 w-16 items-center justify-center rounded-full ${circle}`}>
        <Icon size={28} />
      </span>
      <h2 className="heading-section">{title}</h2>
      {hint && <p className="max-w-md text-sm text-ink-muted">{hint}</p>}
    </div>
  );
}

/**
 * **معادلةٌ تُقرأ على الجوّال** (تدقيقُ الخزينة ٢٠٢٦-١٠-٢٧) — على الشاشة الصغيرة شبكةُ عمودين بلا
 * إشارات والنتيجةُ بعرض الصفّ، **وعلى الكبيرة سطرٌ واحدٌ بإشاراته**: البدايةُ + الداخل − الطالع = النهاية.
 * كانت أربعُ خاناتٍ وثلاثُ إشاراتٍ في صفٍّ يلتفّ فتبقى الإشارةُ وحدَها آخرَ السطر.
 */
export interface EqTerm {
  label: string;
  value: number;
  /** الإشارةُ قبل الخانة على الشاشة الكبيرة — «+» أو «−» أو «±» أو «=». */
  op?: string;
  sub?: string;
  hint?: string;
  tone?: "success" | "danger" | "warning";
  /** **النتيجة** — إطارٌ ملوّنٌ وعرضُ الصفّ على الجوّال. */
  result?: boolean;
  /** يُكتب بإشارته (+/−) لا بقيمته المطلقة. */
  signed?: boolean;
}

export function Equation({ terms }: { terms: EqTerm[] }) {
  return (
    <div className="grid grid-cols-2 gap-2 lg:flex lg:items-stretch">
      {terms.map((t, i) => (
        <EqPart key={i} t={t} />
      ))}
    </div>
  );
}

function EqPart({ t }: { t: EqTerm }) {
  const color = t.tone === "success" ? "text-success" : t.tone === "danger" ? "text-danger" : t.tone === "warning" ? "text-warning" : "";
  return (
    <>
      {t.op && <EqOp>{t.op}</EqOp>}
      <div
        className={`min-w-0 rounded-card border p-3 lg:flex-1 ${t.result ? "col-span-2 border-accent bg-field" : "border-line-soft"}`}
      >
        <span className="block text-xs text-ink-muted">{t.label}</span>
        <span className={`figure block leading-tight ${color}`}>
          {t.signed ? (
            <>
              <span dir="ltr">
                {t.value > 0 ? "+" : t.value < 0 ? "−" : ""}
                {fmtNum(Math.abs(t.value))}
              </span>{" "}
              {m.common.currency}
            </>
          ) : (
            fmtMoney(t.value)
          )}
        </span>
        {t.hint && <span className="block text-2xs text-ink-muted">{t.hint}</span>}
        {t.sub && <span className="block text-2xs text-ink-muted">{t.sub}</span>}
      </div>
    </>
  );
}

function EqOp({ children }: { children: ReactNode }) {
  return <span className="figure hidden items-center px-1 text-ink-muted lg:flex">{children}</span>;
}
