"use client";

/**
 * الشريط العلوي الموحّد — كرت عائم واحد بروح واحدة في كل التطبيقات: اللوحات
 * (إدارة/متجر/مندوب) وموقع الزبون. كان لكل واحد شريطه المرتجل بارتفاعات وحواف
 * مختلفة، فصار المصدر هنا وكلٌّ يمرّر محتواه فقط.
 */

import { useEffect, useRef, useState, type ComponentType, type ReactNode } from "react";
import { getMessages, defaultLocale, fmtNum } from "@rahalgo/i18n";
import { IconChevronDown, IconLogout } from "./icons";

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
  shape = "flush",
}: {
  start?: ReactNode;
  children?: ReactNode;
  /** يلتصق أعلى الشاشة عند التمرير — مفيد لموقع الزبون الطويل */
  sticky?: boolean;
  /**
   * **شكلُ الشريط — وهو الفرقُ بين موقعٍ ولوحةِ تحكّم.**
   *
   * (قرارُ المالك ٢٠٢٦-٠٨-٠٦: «التوب بار في الموقع والزبون بدون بادينك
   *  وحواف مستديرة، وباقي اللوحات خلّيها بحواف مستديرة مع بادينك».)
   *
   * - `flush` — **ممتدٌّ من حافّةٍ إلى حافّة**، يفصله عن المحتوى حدٌّ سفليّ.
   *   وهو شكلُ الموقع: **من فتحه جاء ليتصفّح**، والصفحةُ تأخذ العرضَ كلَّه.
   *
   * - `card`  — **لوحٌ مدوَّرٌ يطفو داخل حشوة اللوحة**، أخوُ لوح المحتوى
   *   تحته. وهو شكلُ لوحة التحكّم: **من فتحها جاء ليعمل في أدوات**،
   *   وأدواتٌ في ألواحٍ تُقرأ أدواتٍ.
   *
   * **وواحدٌ لا يصلح للاثنين**: لوحٌ مدوَّرٌ في الموقع يُقرأ صندوقاً أوّلَ في
   *   شاشةٍ من ثلاثة صناديق، **وشريطٌ ممتدٌّ في اللوحة يلتصق بسايدبارها
   *   فيصير الاثنان كتلةً واحدة.**
   */
  shape?: "flush" | "card";
}) {
  return (
    /*
      **شريطٌ لا يدفع الصفحةَ جانباً.**

      كان `flex` بلا `min-w-0` ولا حدٍّ للفيض: الشعارُ وثمانيةُ اختصاراتٍ
      ورصيدُ المحفظة برقمه **يزيدون عن عرض الجوّال**، فيتمدّد الشريطُ ويجرّ
      `body` معه — **والصفحةُ كلُّها تنزلق أفقيّاً**، وكلُّ سطرٍ فيها يبدأ من
      خارج الشاشة.

      **وحشوةٌ أصغرُ على الصغير** (`px-3`) تكسب أربعين بكسلاً — وهي فرقُ
      اختصارٍ كامل.
    */
    <header
      className={`flex items-center gap-2 bg-surface px-3 py-3 sm:gap-3 ${
        shape === "card"
          ? /* **لوحٌ كلوح المحتوى تحته** — `surface-lit` لا حدٌّ: الأرضُ
               والبطاقةُ لونٌ واحدٌ منذ ٢٠٢٦-٠٨-٠٦، **وخيطُ الضوء أعلى اللوح
               هو ما يرفعه عمّا تحته.** (وهو عينُ ما يفعله `<main>`، فلو
               اختلفا لَقُرئا شيئين.) */
            "surface-lit rounded-card"
          : /*
              **شريطٌ يمتدّ لا كرتٌ يطفو.** (قرارُ المالك ٢٠٢٦-٠٨-٠٦.)

              كان `rounded-card` بهوامشَ من الجوانب الثلاثة — **فيُقرأ صندوقاً
              أوّلَ في شاشةٍ من ثلاثة صناديق**، وشكلُ لوحةِ تحكّمٍ لا شكلُ موقع.

              **والفصلُ بحدٍّ لا بهامش**: الحدُّ السفليُّ يقول «هنا ينتهي الشريطُ
              ويبدأ المحتوى» **بلا أن يقتطع من العرض شيئاً.** والهامشُ يفصل
              بإبعادٍ والحدُّ يفصل بخطّ — **والموقعُ الرسميُّ يفصل بخطّ.**
            */
            "border-b border-line"
      } ${
        /* **ويلتصق عند الصفر**: موضعُ الالتصاق كان مشتقّاً من الزوايا — لوحٌ
           مدوَّرٌ يطفو بعيداً عن الحافّة، **وشريطٌ ممتدٌّ يلتصق بها.** */
        sticky ? "sticky top-0 z-40 bg-surface/85 backdrop-blur-xl" : ""
      }`}
    >
      {/* **والشعارُ يتقلّص ولا يُقصّ** — بلا `min-w-0` يفرض عرضَه كاملاً. */}
      {/* **وفجوةٌ لما يجاور العلامة.** كانت الحاويةُ بلا `gap` لأنّها لم
          تحمل إلّا الشعار. **ومن وضع رابطاً بجانبه لَالتصق به** — والتطبيقاتُ
          التي تمرّر شعاراً وحدَه لا يتغيّر شكلُها. */}
      <div className="flex min-w-0 items-center gap-1 sm:gap-2">{start}</div>

      {/*
        **وصفُّ الأدوات ينزلق وحدَه إن ضاق.**

        **والالتفافُ إلى سطرٍ ثانٍ مرفوض**: الشريطُ يعلو فيدفع المحتوى، **ومن
        ثبّته (`sticky`) يأكل ثلثَ شاشة الجوّال.**

        `[scrollbar-width:none]` — **شريطُ تمريرٍ داخل الشريط يُقرأ عطباً**،
        والانزلاقُ بالإصبع لا يحتاج مقبضاً يُرى.
      */}
      <div className="ms-auto flex min-w-0 items-center gap-1.5 overflow-x-auto [-ms-overflow-style:none] [scrollbar-width:none] sm:gap-2 [&::-webkit-scrollbar]:hidden">
        {children}
      </div>
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

/** ارتفاع وحواف موحّدة لكل عناصر الشريط — لا يقرّر كل عنصر مقاسه بنفسه.
 *
 * **و`shrink-0` لأنّ الصفَّ ينزلق**: بدونها يضغط `flex` الحبّاتِ حتّى تتداخل
 * أيقوناتُها، **فيصير الشريطُ صفّاً من رموزٍ مقصوصة** بدل أن ينزلق سليماً.
 *
 * **وحشوةٌ أضيقُ على الجوّال** (`px-2`) — ثمانيةُ اختصاراتٍ × ثمانية بكسلات
 * تكسب اختصاراً كاملاً في العرض. */
const chipBase =
  "flex shrink-0 items-center gap-2 rounded-control px-2 py-2 text-sm transition-colors sm:gap-2 sm:px-3";

/**
 * **لا صندوقَ خلف الأيقونات.**
 *
 * كان لكلّ أيقونةٍ مستطيلٌ ملوّنٌ خلفها — **وشريطٌ فيه ستّةُ مستطيلاتٍ متجاورة
 * يصير سلسلةَ صناديقَ لا صفَّ أدوات**، والعينُ تعدّ الحدودَ قبل أن تقرأ ما
 * فيها. **والحالةُ تُقال بلون الأيقونة لا بصندوقٍ حولها.**
 * (قرارُ المالك ٢٠٢٦-٠٨-٠٣: «هذا المربع خلف الأيقونات لا أريده أبداً».)
 */
const chipTones = {
  /** محايد — اختصار عادي */
  plain: "text-ink-muted hover:text-ink",
  /** نشِط — القسم المفتوح حالياً. **أبيضُ بلا صندوق**: الباهتُ يخفت
   *  والنشِطُ يسطع، **والفرقُ بينهما يكفي بلا لونٍ ثالث.**
   *  (قرارُ المالك ٢٠٢٦-٠٨-٠٣: «أيقونة الطلبات ترجع بيضاء لا برتقالية».) */
  active: "font-medium text-ink",
  /** بارز — إجراء رئيسي (السلة مثلاً) */
  primary: "font-medium text-ink hover:text-accent-text",
  /** ثانوي مميّز — التقييم/التسوّق كزبون */
  accent: "font-bold text-accent-text hover:opacity-80",
  /** خطر — الخروج. ممتلئ لا شفّاف: زرّ الخروج يجب أن يُميَّز بلمحة كي لا
   *  يُضغط سهواً، والنصّ الأحمر على أبيض يذوب بين بقية العناصر. */
  danger: "bg-danger-solid font-medium text-on-solid hover:opacity-90",
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
      className="font-bold text-ink hover:text-accent-text"
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
        <img src={url} alt="" loading="lazy" className="h-full w-full object-cover" />
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
      /* **العدّادُ بالنبرة بنصٍّ داكن** — لا أبيض: الأبيضُ على النبرة
         ١٫٣٠ **يذوب**، والداكنُ ١٣٫٩٢. **والجرسُ يبقى أبيضَ كما هو.**
         (قرارُ المالك ٢٠٢٦-٠٨-٠٣: «العدّاد فقط وليس الجرس».) */
      className={`absolute -top-1.5 -start-1.5 flex h-5 min-w-5 items-center justify-center rounded-badge px-1 text-xs font-bold ${
        tone === "danger" ? "bg-danger-solid text-on-solid" : "bg-accent text-on-bright"
      }`}
    >
      {fmtNum(count)}
    </span>
  );
}

// ---------- قائمةُ الحساب ----------

/** بندٌ في قائمة الحساب — أيقونةٌ ونصٌّ ووجهة. */
export interface AccountMenuItem {
  href: string;
  label: string;
  icon: ComponentType<{ size?: number; className?: string }>;
}

/**
 * **قائمةُ الحساب — الاسمُ بجانب الصورة وما تحته ينفتح بنقرة.**
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-٠٦: «بجانب صورة البروفايل يجب أن يظهر اسم صاحب
 *  الحساب، ثمّ قائمة أسفلها تسجيل الخروج ونضع فيها العروض ودعوة صديق
 *  والمفضّلة والشكاوى — بحيث نزيل هذه العناصر من التوب بار ليكون توب بار
 *  احترافيّاً بعناصر قليلة واضحة».)
 *
 * # ولماذا قائمةٌ الآن وقد رُفضت من قبل
 *
 * **رُفضت لأنّها كانت تُخفي ما هو معروضٌ بجانبها** — السلّةَ والطلباتِ
 * والإشعارات — **وتُخفي خلفها ما ليس معروضاً أصلاً.** فلا هي اختصارٌ ولا
 * ترتيب.
 *
 * **وهذه عكسُها**: ما فيها **ليس في الشريط**، وما في الشريط ليس فيها.
 * فالقائمةُ تحمل ما يُفتح مرّةً في الشهر (عرضٌ · دعوةٌ · شكوى)، **والشريطُ
 * يحمل ما يُفتح كلَّ يوم** (الجرسُ والمحفظةُ والطلبات).
 *
 * # والاسمُ ليس زينة
 *
 * **صورةٌ وحدَها لا تقول أيَّ حسابٍ مفتوح** — ومن يشارك جهازاً مع أهله
 * **يطلب باسمِ غيره ولا يعلم.** والأحرفُ الأولى في الصورة تُقرأ حين تُتأمَّل
 * لا حين تُمسح العينُ الشريطَ.
 *
 * # وثلاثةُ أبوابٍ للإغلاق
 *
 * **الهروبُ ونقرةٌ خارجَها وتبدّلُ المسار.** والثالثُ أهمُّها: من ضغط بنداً
 * انتقل، **فلو بقيت مفتوحةً غطّت الصفحةَ التي فُتحت لأجلها.**
 */
export function AccountMenu({
  Link,
  accountHref,
  accountLabel,
  avatarUrl,
  name,
  items,
  onLogout,
  logoutLabel,
  active = "",
}: {
  Link: LinkType;
  accountHref: string;
  accountLabel: string;
  avatarUrl: string | null;
  name: string;
  items: readonly AccountMenuItem[];
  onLogout: () => void;
  logoutLabel: string;
  active?: string;
}) {
  const [open, setOpen] = useState(false);
  const box = useRef<HTMLDivElement>(null);

  // **وتُغلق بتبدّل المسار** — بند القائمة ينقل، والقائمةُ الباقيةُ تغطّي
  // الوجهةَ التي فُتحت لأجلها.
  useEffect(() => setOpen(false), [active]);

  useEffect(() => {
    if (!open) return;
    const away = (e: PointerEvent) => {
      if (box.current && !box.current.contains(e.target as Node)) setOpen(false);
    };
    const esc = (e: KeyboardEvent) => e.key === "Escape" && setOpen(false);
    // `pointerdown` لا `click`: **الإغلاق يقع مع بدء اللمسة** فلا يبقى
    // المنسدلُ ظاهراً بين ضغطةٍ ورفعها على الجوّال.
    document.addEventListener("pointerdown", away);
    document.addEventListener("keydown", esc);
    return () => {
      document.removeEventListener("pointerdown", away);
      document.removeEventListener("keydown", esc);
    };
  }, [open]);

  const inMenu = items.some((it) => active.startsWith(it.href)) || active.startsWith(accountHref);

  return (
    <div className="relative" ref={box}>
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        aria-haspopup="menu"
        aria-expanded={open}
        className={`taparea flex items-center gap-2 rounded-control p-1 transition-colors ${
          open || inMenu ? "bg-page text-ink" : "text-ink-muted hover:bg-page hover:text-ink"
        }`}
      >
        <Avatar url={avatarUrl} name={name} size={TOPBAR_AVATAR} />
        {/* **والاسمُ يُقصّ ولا يمدّ الشريط**: أسماءٌ ثلاثيّةٌ تدفع ما بعدها
            خارجَ الشاشة. **ويُخفى تحت ٦٤٠** حيث الشريطُ أضيقُ ما يكون —
            والشريطُ السفليُّ يحمل «حسابي» بتسميته هناك. */}
        <span className="max-w-20 truncate text-sm font-medium sm:max-w-28">{name}</span>
        <IconChevronDown
          size={15}
          className={`shrink-0 transition-transform ${open ? "rotate-180" : ""}`}
        />
      </button>

      {open && (
        <div
          role="menu"
          aria-label={accountLabel}
          /* **تُفتح إلى الداخل لا إلى الخارج**: `end-0` منطقيّةٌ تتبع اتجاهَ
             الصفحة، **و`right-0` كانت ستدفعها خارجَ الشاشة في العربيّة.** */
          /* **وتنزل تحت الشريط لا داخلَه**: `mt-2` كانت تضعها فوق حافّته
             بخمسة بكسلات — **لأنّ `top-full` من أسفل الزرّ لا من أسفل
             الشريط**، وبينهما حشوةُ الشريط (١٢px). فقُيست: ٢٠ − ١٢ = ثمانيةٌ
             تحت الحافّة. */
          className="absolute end-0 top-full z-50 mt-5 w-56 overflow-hidden rounded-card border border-line bg-raised elev-3"
        >
          {/* **ورأسُها بابُ الحساب** — كان الصورةُ رابطاً إليه، فلمّا صارت
              زرَّ قائمةٍ **فقد «حسابي» بابَه في الشريط.** */}
          <Link
            href={accountHref}
            className="flex items-center gap-2 border-b border-line px-3 py-3 text-sm transition-colors hover:bg-page"
          >
            <Avatar url={avatarUrl} name={name} size={32} />
            <span className="min-w-0 flex-1">
              <span className="block truncate font-bold text-ink">{name}</span>
              <span className="block text-xs text-ink-muted">{accountLabel}</span>
            </span>
          </Link>

          <div className="py-1">
            {items.map((it) => {
              const on = active.startsWith(it.href);
              const Icon = it.icon;
              return (
                <Link
                  key={it.href}
                  href={it.href}
                  className={`flex items-center gap-2.5 px-3 py-2.5 text-sm transition-colors ${
                    on ? "bg-page font-medium text-ink" : "text-ink-muted hover:bg-page hover:text-ink"
                  }`}
                >
                  <Icon size={17} />
                  {it.label}
                </Link>
              );
            })}
          </div>

          {/* **والخروجُ آخرُها بحدٍّ فوقه** — أخطرُ بندٍ فيها، **وحدٌّ يفصله
              عن الروابط يمنع أن يُضغط بامتداد الإصبع.** */}
          <button
            type="button"
            role="menuitem"
            onClick={() => {
              setOpen(false);
              onLogout();
            }}
            className="flex w-full items-center gap-2.5 border-t border-line px-3 py-2.5 text-start text-sm text-danger transition-colors hover:bg-page"
          >
            <IconLogout size={17} />
            {logoutLabel}
          </button>
        </div>
      )}
    </div>
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
  menu,
  active = "",
}: {
  Link: LinkType;
  /** مكوّن الجرس الحيّ — يُمرَّر جاهزاً لأنه يحتاج api وwsUrl الخاصَّين بالتطبيق */
  notifications?: ReactNode;
  /**
   * **بنودُ قائمة الحساب — وبلاها تبقى الصورةُ رابطاً والخروجُ رقعةً.**
   *
   * القائمةُ لموقع الزبون (قرارُ المالك ٢٠٢٦-٠٨-٠٦)، **واللوحاتُ الأربعُ
   * لا تمرّرها فلا يتغيّر شريطُها حرفاً.** ولو أردناها لهنّ يوماً مُرّرت،
   * **ولا تُبنى ثانيةً.**
   */
  menu?: readonly AccountMenuItem[];
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
      {menu && menu.length > 0 ? (
        <AccountMenu
          Link={Link}
          accountHref={accountHref}
          accountLabel={accountLabel}
          avatarUrl={avatarUrl}
          name={name}
          items={menu}
          onLogout={onLogout}
          logoutLabel={logoutLabel}
          active={active}
        />
      ) : (
        /* الصورة وحدها بلا اسم: صاحبها يعرف اسمه، وإطالةُ الشريط به تزاحم ما يفيده */
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
      )}
      {/* **الخروجُ لا يُزاحم على الهاتف.**

          كان رقعةً حمراءَ مصمتةً في أضيق شريط: **أبرزُ ما في الشاشة، وهو
          أقلُّ ما يُستعمل** — ويجاور الصورةَ والجرسَ فيُضغط بالخطأ، **وإصبعٌ
          واحدةٌ على شاشةِ هاتفٍ لا تُخطئ خطأً يُتدارَك.**

          فيبقى في الشاشات الواسعة كما كان، **وعلى الهاتف يُطوى إلى صفحة
          «حسابي»** — وهي موضعُه المعتاد في كلّ تطبيق.
          (قرارُ المالك ٢٠٢٦-٠٨-٠٣.) */}
      {/* **ولا خروجَ مرّتين**: من مرّر قائمةً فالخروجُ آخرُ بندٍ فيها،
          **ورقعةٌ حمراءُ بجانبها تسأل أيُّهما الحقيقيّ.** */}
      {!(menu && menu.length > 0) && (
        <span className="hidden sm:contents">
          <TopBarChip tone="danger" onClick={onLogout} title={logoutLabel} aria-label={logoutLabel}>
            <IconLogout size={TOPBAR_ICON} />
            <span className="hidden lg:inline">{logoutLabel}</span>
          </TopBarChip>
        </span>
      )}
    </>
  );
}
