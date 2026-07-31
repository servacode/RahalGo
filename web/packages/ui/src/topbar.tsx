"use client";

/**
 * الشريط العلوي الموحّد — كرت عائم واحد بروح واحدة في كل التطبيقات: اللوحات
 * (إدارة/متجر/مندوب) وموقع الزبون. كان لكل واحد شريطه المرتجل بارتفاعات وحواف
 * مختلفة، فصار المصدر هنا وكلٌّ يمرّر محتواه فقط.
 */

import type { ComponentType, ReactNode } from "react";
import { getMessages, defaultLocale, fmtNum } from "@rahalgo/i18n";

const m = getMessages(defaultLocale);

type LinkType = ComponentType<{
  href: string;
  className?: string;
  title?: string;
  children: ReactNode;
  onClick?: () => void;
  "aria-label"?: string;
}>;

/** كرت الشريط العلوي العائم. start يمين (شعار/عنوان)، children يسار (الأدوات). */
export function TopBar({
  start,
  children,
  sticky = false,
}: {
  start?: ReactNode;
  children?: ReactNode;
  /** يلتصق أعلى الشاشة عند التمرير — مفيد لموقع الزبون الطويل */
  sticky?: boolean;
}) {
  return (
    <header
      className={`mb-3 flex items-center gap-3 rounded-card border border-line bg-surface px-4 py-2.5 shadow-sm ${
        sticky ? "sticky top-3 z-40" : ""
      }`}
    >
      {start}
      <div className="ms-auto flex items-center gap-2">{children}</div>
    </header>
  );
}

/** ارتفاع وحواف موحّدة لكل عناصر الشريط — لا يقرّر كل عنصر مقاسه بنفسه. */
const chipBase =
  "flex items-center gap-1.5 rounded-control px-2.5 py-1.5 text-sm transition-colors";

const chipTones = {
  /** محايد — اختصار عادي */
  plain: "text-ink-muted hover:bg-page hover:text-ink",
  /** نشِط — القسم المفتوح حالياً */
  active: "bg-primary-light font-medium text-primary-dark",
  /** بارز — إجراء رئيسي (السلة مثلاً) */
  primary: "bg-primary font-medium text-white hover:bg-primary-dark",
  /** ثانوي مميّز — التقييم/التسوّق كزبون */
  accent: "bg-accent/10 font-bold text-accent-dark hover:bg-accent/20",
  /** خطر — الخروج */
  danger: "text-danger hover:bg-danger/10",
} as const;

export type ChipTone = keyof typeof chipTones;

export function TopBarChip({
  tone = "plain",
  className = "",
  ...props
}: React.ButtonHTMLAttributes<HTMLButtonElement> & { tone?: ChipTone }) {
  return <button {...props} className={`${chipBase} ${chipTones[tone]} ${className}`} />;
}

/** نفس الحبّة لكن رابطاً — كي لا يختلف مقاس الرابط عن الزر بجانبه. */
export function TopBarLink({
  Link,
  href,
  tone = "plain",
  title,
  className = "",
  children,
  ...rest
}: {
  Link: LinkType;
  href: string;
  tone?: ChipTone;
  title?: string;
  className?: string;
  children: ReactNode;
  "aria-label"?: string;
}) {
  return (
    <Link
      {...rest}
      href={href}
      title={title}
      className={`${chipBase} ${chipTones[tone]} ${className}`}
    >
      {children}
    </Link>
  );
}

/** رصيد المحفظة — عنصر واحد كان مكرراً حرفياً في الهيكل وفي شريط الزبون. */
export function WalletPill({
  Link,
  href,
  balance,
  icon,
}: {
  Link: LinkType;
  href: string;
  balance: number;
  icon: ReactNode;
}) {
  return (
    <TopBarLink
      Link={Link}
      href={href}
      title={m.terms.wallet}
      className="bg-primary-light font-bold text-primary-dark hover:bg-primary-light/70"
    >
      {icon}
      <span dir="ltr">{fmtNum(balance)}</span>
      <span className="hidden text-xs font-normal sm:inline">{m.common.currency}</span>
    </TopBarLink>
  );
}

/** الصورة الشخصية — كانت مكتوبة يدوياً في ثلاثة أماكن بمقاسات مختلفة. */
export function Avatar({
  url,
  name,
  size = 28,
}: {
  url: string | null;
  name: string;
  size?: number;
}) {
  return (
    <span
      style={{ width: size, height: size }}
      className="flex shrink-0 items-center justify-center overflow-hidden rounded-full bg-primary-light text-sm font-bold text-primary-dark"
    >
      {url ? (
        // eslint-disable-next-line @next/next/no-img-element
        <img src={url} alt="" className="h-full w-full object-cover" />
      ) : (
        (name || m.terms.avatarFallback).slice(0, 1)
      )}
    </span>
  );
}

/** عدّاد صغير فوق أيقونة (السلة، الإشعارات). */
export function CountBadge({ count, tone = "accent" }: { count: number; tone?: "accent" | "danger" }) {
  if (count <= 0) return null;
  return (
    <span
      className={`absolute -top-1.5 -start-1.5 flex h-5 min-w-5 items-center justify-center rounded-badge px-1 text-xs font-bold text-white ${
        tone === "danger" ? "bg-danger" : "bg-accent"
      }`}
    >
      {fmtNum(count)}
    </span>
  );
}

/** لوحة قائمة منسدلة — نفس حواف الكرت المركزي. */
export function MenuPanel({ children }: { children: ReactNode }) {
  return (
    <div className="absolute end-0 mt-1 w-52 overflow-hidden rounded-card border border-line bg-surface py-1 shadow-lg">
      {children}
    </div>
  );
}

/** سطر داخل القائمة المنسدلة — رابطاً أو زراً بنفس المقاس. */
export function MenuItem({
  Link,
  href,
  icon,
  label,
  onClick,
  tone = "plain",
}: {
  Link?: LinkType;
  href?: string;
  icon: ReactNode;
  label: string;
  onClick?: () => void;
  tone?: "plain" | "accent" | "danger";
}) {
  const cls = `flex w-full items-center gap-2.5 px-3 py-2 text-sm ${
    { plain: "text-ink hover:bg-page", accent: "text-accent-dark hover:bg-accent/10", danger: "text-danger hover:bg-danger/10" }[tone]
  }`;
  if (Link && href) {
    return (
      <Link href={href} className={cls}>
        {icon}
        {label}
      </Link>
    );
  }
  return (
    <button onClick={onClick} className={cls}>
      {icon}
      {label}
    </button>
  );
}
