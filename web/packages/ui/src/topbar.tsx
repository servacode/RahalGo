"use client";

/**
 * الشريط العلوي الموحّد — كرت عائم واحد بروح واحدة في كل التطبيقات: اللوحات
 * (إدارة/متجر/مندوب) وموقع الزبون. كان لكل واحد شريطه المرتجل بارتفاعات وحواف
 * مختلفة، فصار المصدر هنا وكلٌّ يمرّر محتواه فقط.
 */

import type { ComponentType, ReactNode } from "react";
import { getMessages, defaultLocale, fmtNum } from "@rahalgo/i18n";
import { IconLogout } from "./icons";

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
      className={`mb-3 flex items-center gap-3 rounded-card border border-line bg-surface px-5 py-3.5 shadow-sm ${
        sticky ? "sticky top-3 z-40" : ""
      }`}
    >
      {start}
      <div className="ms-auto flex items-center gap-2">{children}</div>
    </header>
  );
}

/**
 * مقاس أيقونات الشريط — رقم واحد لا يقرّره كل عنصر بنفسه.
 *
 * كانت أربعة مقاسات في شريط واحد (14 و15 و16 و18) فبدت الأيقونات غير متساوية
 * وإن كان كلٌّ منها سليماً وحده. التفاوت في المقاس يُقرأ فوضىً حتى لو لم يُلحظ
 * سببه، والصورة الشخصية وحدها تكبر لأنها هوية لا رمز.
 */
//
// **وكبر المقاس بطلب المالك (٢٠٢٦-٠٨-٠١)**: شريطٌ ضيّق برموزٍ صغيرة يُقرأ
// بمشقّة — والرموز بلا تسميات لا تُفهم إلا بالنظر إليها. والزيادةُ هنا
// مركزيةٌ فيرثها الخمسة معاً؛ ولو كُبّر في الموقع وحده لانفصل شكلُه عن اللوحات
// وهي منصّةٌ واحدة.
export const TOPBAR_ICON = 20;
export const TOPBAR_AVATAR = 36;

/** ارتفاع وحواف موحّدة لكل عناصر الشريط — لا يقرّر كل عنصر مقاسه بنفسه. */
const chipBase =
  "flex items-center gap-2 rounded-control px-3 py-2 text-sm transition-colors";

const chipTones = {
  /** محايد — اختصار عادي */
  plain: "text-ink-muted hover:bg-page hover:text-ink",
  /** نشِط — القسم المفتوح حالياً */
  active: "bg-primary-light font-medium text-primary-dark",
  /** بارز — إجراء رئيسي (السلة مثلاً) */
  primary: "bg-primary font-medium text-white hover:bg-primary-dark",
  /** ثانوي مميّز — التقييم/التسوّق كزبون */
  accent: "bg-accent/10 font-bold text-accent-dark hover:bg-accent/20",
  /** خطر — الخروج. ممتلئ لا شفّاف: زرّ الخروج يجب أن يُميَّز بلمحة كي لا
   *  يُضغط سهواً، والنصّ الأحمر على أبيض يذوب بين بقية العناصر. */
  danger: "bg-danger-solid font-medium text-white hover:opacity-90",
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
      /* **دائريّةٌ لا مربّعةٌ بزوايا.** `rounded-badge` نصفُ قطرٍ ثابت — على
         حجمٍ صغيرٍ يبدو مربّعاً مشذّبَ الأركان. **وصورةُ الشخص دائرةٌ في كلّ
         مكان**، وشكلٌ يخالف ما اعتادته العينُ يُقرأ خطأً في التصميم.
         (قرارُ المالك ٢٠٢٦-٠٨-٠٣.) */
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
      /* **العدّادُ برتقاليٌّ بنصٍّ داكن** — لا أبيض: الأبيضُ على البرتقاليّ
         ٢٫٢٢ **يذوب**، والداكنُ ٨٫٤٩. **والجرسُ يبقى أبيضَ كما هو.**
         (قرارُ المالك ٢٠٢٦-٠٨-٠٣: «العدّاد فقط وليس الجرس».) */
      className={`absolute -top-1.5 -start-1.5 flex h-5 min-w-5 items-center justify-center rounded-badge px-1 text-xs font-bold ${
        tone === "danger" ? "bg-danger-solid text-white" : "bg-accent text-shell"
      }`}
    >
      {fmtNum(count)}
    </span>
  );
}

// ---------- مجموعة أدوات الشريط ----------

/**
 * الأدوات على يسار الشريط — **تركيبٌ واحد لكل تطبيقات المشروع**.
 *
 * كانت العناصر مشتركة والتركيبُ مبنيّاً مرّتين: مرّة في هيكل اللوحات ومرّة في
 * شريط الموقع. فاختلفا بلا أن يقصد أحد — الصورة باسمٍ هنا وبلا اسمٍ هناك،
 * والخروج يظهر عند `sm` في واحد و`lg` في الآخر. عناصرٌ مشتركة بتركيبٍ مكرّر
 * تُنتج شريطين مختلفين، وهذا هو الانحراف الذي تمنعه المركزية لا التكرار وحده.
 *
 * والترتيب هنا **واحد لا يُبدَّل**: الإشعارات، فالمحفظة، فما يخصّ التطبيق،
 * فالحساب، فالخروج. من يعرف موضع زرٍّ في لوحةٍ يجده في مكانه في الأخرى.
 */
export function TopBarActions({
  Link,
  notifications,
  walletHref,
  balance,
  walletIcon,
  accountHref,
  accountLabel,
  avatarUrl,
  name,
  onLogout,
  logoutLabel,
  extras,
  active = "",
}: {
  Link: LinkType;
  /** مكوّن الجرس الحيّ — يُمرَّر جاهزاً لأنه يحتاج api وwsUrl الخاصَّين بالتطبيق */
  notifications?: ReactNode;
  walletHref?: string;
  balance?: number;
  walletIcon?: ReactNode;
  accountHref: string;
  accountLabel: string;
  avatarUrl: string | null;
  name: string;
  onLogout: () => void;
  logoutLabel: string;
  /** ما يخصّ التطبيق وحده: السلة، الطلبات، لوحتي، شارة التقييم */
  extras?: ReactNode;
  /** المسار الحالي — لتمييز القسم المفتوح */
  active?: string;
}) {
  return (
    <>
      {notifications}
      {walletHref && (
        <WalletPill Link={Link} href={walletHref} balance={balance ?? 0} icon={walletIcon} />
      )}
      {extras}
      {/* الصورة وحدها بلا اسم: صاحبها يعرف اسمه، وإطالةُ الشريط به تزاحم ما يفيده */}
      <TopBarLink
        Link={Link}
        href={accountHref}
        title={accountLabel}
        aria-label={accountLabel}
        tone={active.startsWith(accountHref) ? "active" : "plain"}
        className="!p-1"
      >
        <Avatar url={avatarUrl} name={name} size={TOPBAR_AVATAR} />
      </TopBarLink>
      {/* **الخروجُ لا يُزاحم على الهاتف.**

          كان رقعةً حمراءَ مصمتةً في أضيق شريط: **أبرزُ ما في الشاشة، وهو
          أقلُّ ما يُستعمل** — ويجاور الصورةَ والجرسَ فيُضغط بالخطأ، **وإصبعٌ
          واحدةٌ على شاشةِ هاتفٍ لا تُخطئ خطأً يُتدارَك.**

          فيبقى في الشاشات الواسعة كما كان، **وعلى الهاتف يُطوى إلى صفحة
          «حسابي»** — وهي موضعُه المعتاد في كلّ تطبيق.
          (قرارُ المالك ٢٠٢٦-٠٨-٠٣.) */}
      <span className="hidden sm:contents">
        <TopBarChip tone="danger" onClick={onLogout} title={logoutLabel} aria-label={logoutLabel}>
          <IconLogout size={TOPBAR_ICON} />
          <span className="hidden lg:inline">{logoutLabel}</span>
        </TopBarChip>
      </span>
    </>
  );
}
