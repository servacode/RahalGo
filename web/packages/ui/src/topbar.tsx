"use client";

/**
 * الشريط العلوي الموحّد — كرت عائم واحد بروح واحدة في كل التطبيقات: اللوحات
 * (إدارة/متجر/مندوب) وموقع الزبون. كان لكل واحد شريطه المرتجل بارتفاعات وحواف
 * مختلفة، فصار المصدر هنا وكلٌّ يمرّر محتواه فقط.
 */

import {
  useCallback,
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
  type ComponentType,
  type ReactNode,
} from "react";
import { createPortal } from "react-dom";
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
      className={`surface-lit flex items-center gap-2 bg-surface px-3 py-4 sm:gap-3 ${
        shape === "card"
          ? /* **لوحٌ كلوح المحتوى تحته** — `surface-lit` لا حدٌّ: الأرضُ
               والبطاقةُ لونٌ واحدٌ منذ ٢٠٢٦-٠٨-٠٦، **وخيطُ الضوء أعلى اللوح
               هو ما يرفعه عمّا تحته.** (وهو عينُ ما يفعله `<main>`، فلو
               اختلفا لَقُرئا شيئين.) */
            "rounded-card"
          : /*
              **شريطٌ يمتدّ لا كرتٌ يطفو.** (قرارُ المالك ٢٠٢٦-٠٨-٠٦.)

              كان `rounded-card` بهوامشَ من الجوانب الثلاثة — **فيُقرأ صندوقاً
              أوّلَ في شاشةٍ من ثلاثة صناديق**، وشكلُ لوحةِ تحكّمٍ لا شكلُ موقع.

              **ولا فاصلَ أصلاً** — (قرارُ المالك ٢٠٢٦-٠٨-٠٦: «الخطُّ الأبيضُ
              من التوب بار والفوتر ألغِه ليندمج التوب بار والفوتر مع الخلفيّة
              والمحتوى، يعني بدون فاصل»).

              **كان حدٌّ سفليٌّ يقول «هنا ينتهي الشريط»** — وذلك يصحّ حين
              يكون الشريطُ لوحاً مصمتاً. **ولمّا صار زجاجاً يمرّ منه ما تحته
              صار الخطُّ هو الشيءَ الوحيدَ الذي يقطع الصورة.**

              **والزجاجُ يفصل بكثافته لا بخطّه.**
            */
            ""
      } ${
        /* **ويلتصق عند الصفر**: موضعُ الالتصاق كان مشتقّاً من الزوايا — لوحٌ
           مدوَّرٌ يطفو بعيداً عن الحافّة، **وشريطٌ ممتدٌّ يلتصق بها.** */
        /* **والتضبيبُ من `surface-lit` لا من هنا** — كان `backdrop-blur-xl`
           يُكتب في حالة الالتصاق وحدَها، **فالشريطُ زجاجٌ حين يمرّر المستخدمُ
           ولوحٌ مصمتٌ حين يقف.** (قرارُ المالك ٢٠٢٦-٠٨-٠٦.) */
        sticky ? "sticky top-0 z-40" : ""
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
/* **والأيقونةُ تتبع الحرف**: كبُر حرفُ الحبّة من ١٤ إلى ١٦ **فأيقونةٌ
   بعشرين تُقرأ ضامرةً بجانبه.** (طلبُ المالك ٢٠٢٦-٠٨-٠٧: «كبّر التوب بار
   قليلاً وكلمةَ التسوّق مع الأيقونة لتكون واضحة».) */
export const TOPBAR_ICON = 22;
export const TOPBAR_AVATAR = 36;

/** ارتفاع وحواف موحّدة لكل عناصر الشريط — لا يقرّر كل عنصر مقاسه بنفسه.
 *
 * **و`shrink-0` لأنّ الصفَّ ينزلق**: بدونها يضغط `flex` الحبّاتِ حتّى تتداخل
 * أيقوناتُها، **فيصير الشريطُ صفّاً من رموزٍ مقصوصة** بدل أن ينزلق سليماً.
 *
 * **وحشوةٌ أضيقُ على الجوّال** (`px-2`) — ثمانيةُ اختصاراتٍ × ثمانية بكسلات
 * تكسب اختصاراً كاملاً في العرض. */
const chipBase =
  "flex shrink-0 items-center gap-2 rounded-control px-2 py-2 text-base transition-colors sm:gap-2 sm:px-3";

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
      className="flex shrink-0 items-center justify-center overflow-hidden rounded-full bg-primary-tint text-sm font-bold text-primary"
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
  const [at, setAt] = useState({ top: 0, left: 0 });
  const btn = useRef<HTMLButtonElement>(null);
  const panel = useRef<HTMLDivElement>(null);

  /* ══════════════════════════════════════════════════════════════════════
     **موضعٌ يُحسب، وبوّابةٌ إلى `body` — لأنّ الشريط يقصّ**
     ══════════════════════════════════════════════════════════════════════

     (عطبٌ شهده المالك ٢٠٢٦-٠٨-٠٦: «القائمة لا تفتح، لا يوجد أيّ شيء
      بداخلها».)

     **صفُّ الأدوات فيه `overflow-x-auto`** — يمنع الشريطَ أن يجرّ الصفحةَ
     أفقيّاً حين تكثر الأيقونات. **وأيُّ محورٍ غيرِ `visible` يجعل الآخرَ
     `auto` بحكم المواصفة** — فصار الصفُّ يقصّ رأسيّاً أيضاً.

     **فالقائمةُ كانت تُرسم داخلَه فتُقصّ عند حافّته**: موجودةٌ في الشجرة،
     مقيسةُ الأبعاد، **ولا يُرى منها شيء.**

     **ولا يكفي `position: fixed`**: الشريطُ عليه `backdrop-blur` —
     **و`backdrop-filter` تُنشئ كتلةَ احتواءٍ للثابت** فيعود أسيرَ الشريط.

     **فالبوّابةُ إلى `body` وحدَها تخرج من كلّ قاصّ** — ومعها يُحسب الموضعُ
     بجافاسكربت لأنّ العنصرَ لم يعد جارَ زرّه في الشجرة.
     ══════════════════════════════════════════════════════════════════════ */
  const place = useCallback(() => {
    const r = btn.current?.getBoundingClientRect();
    if (!r) return;
    const W = 224; // w-56
    const vw = document.documentElement.clientWidth;
    // **تحت الزرّ ومحصورةٌ في الشاشة**: تُوسَّط تحته ثمّ تُقصّ إلى الحافّتين
    // بثمانيةٍ — **فلا تخرج يميناً في العربيّة ولا يساراً في اللاتينيّة.**
    const left = Math.min(Math.max(r.left + r.width / 2 - W / 2, 8), vw - W - 8);
    // **وتُعلَّق من حافّة الشريط لا من حافّة الزرّ**: بينهما حشوةُ الشريط
    // (١٢px)، **فثمانيةٌ من الزرّ تضعها فوق الحافّة بأربعة** — تُقرأ ملتصقةً
    // بالبار لا منسدلةً منه. (وقِيس: ‎−٤ في اللوحات و‎−٥ في الموقع.)
    const bar = btn.current?.closest("header")?.getBoundingClientRect();
    setAt({ top: (bar?.bottom ?? r.bottom) + 8, left });
  }, []);

  // **يُقاس قبل الطلاء** — `useEffect` يترك القائمةَ تُرسم عند (٠،٠) إطاراً
  // ثمّ تقفز إلى موضعها، **وهي قفزةٌ تُرى.**
  useLayoutEffect(() => {
    if (open) place();
  }, [open, place]);

  // **وتُغلق بتبدّل المسار** — بند القائمة ينقل، والقائمةُ الباقيةُ تغطّي
  // الوجهةَ التي فُتحت لأجلها.
  useEffect(() => setOpen(false), [active]);

  useEffect(() => {
    if (!open) return;
    const away = (e: PointerEvent) => {
      const t = e.target as Node;
      if (btn.current?.contains(t) || panel.current?.contains(t)) return;
      setOpen(false);
    };
    const esc = (e: KeyboardEvent) => e.key === "Escape" && setOpen(false);
    // `pointerdown` لا `click`: **الإغلاق يقع مع بدء اللمسة** فلا يبقى
    // المنسدلُ ظاهراً بين ضغطةٍ ورفعها على الجوّال.
    document.addEventListener("pointerdown", away);
    document.addEventListener("keydown", esc);
    // **والشريطُ لاصقٌ والقائمةُ ثابتة** — فلو مُرِّرت الصفحةُ بقيت معلّقةً
    // في الهواء بلا زرّها. (`capture` يلتقط تمريرَ أيّ حاوية.)
    window.addEventListener("scroll", place, true);
    window.addEventListener("resize", place);
    return () => {
      document.removeEventListener("pointerdown", away);
      document.removeEventListener("keydown", esc);
      window.removeEventListener("scroll", place, true);
      window.removeEventListener("resize", place);
    };
  }, [open, place]);

  const inMenu = items.some((it) => active.startsWith(it.href)) || active.startsWith(accountHref);

  const menu = (
    <div
      ref={panel}
      role="menu"
      aria-label={accountLabel}
      style={{ top: at.top, left: at.left }}
      className="fixed z-[80] w-56 overflow-hidden surface-raised elev-3"
    >
      {/* **ورأسُها بابُ الحساب** — كانت الصورةُ رابطاً إليه، فلمّا صارت
          زرَّ قائمةٍ **فقد «حسابي» بابَه في الشريط.** */}
      <Link
        href={accountHref}
        className="flex items-center gap-2 border-b border-line-soft px-3 py-3 text-sm transition-colors hover:bg-row-hover"
      >
        <Avatar url={avatarUrl} name={name} size={32} />
        <span className="min-w-0 flex-1">
          <span className="block truncate font-bold text-ink">{name}</span>
          <span className="block text-xs text-ink-muted">{accountLabel}</span>
        </span>
      </Link>

      {/* **وقائمةٌ بلا بنودٍ لا فراغَ فيها** — اللوحاتُ تمرّر مصفوفةً فارغةً
          عمداً (قرارُ المالك: «أضِف فقط تسجيل الخروج للقائمة»)، **فلا يُرسم
          حاوٍ فارغٌ بحشوته.** */}
      {items.length > 0 && (
        <div className="py-1">
          {items.map((it) => {
            const on = active.startsWith(it.href);
            const Icon = it.icon;
            return (
              <Link
                key={it.href}
                href={it.href}
                className={`flex items-center gap-2.5 px-3 py-2.5 text-sm transition-colors ${
                  on ? "bg-primary-tint font-medium text-ink" : "text-ink-muted hover:bg-row-hover hover:text-ink"
                }`}
              >
                <Icon size={17} />
                {it.label}
              </Link>
            );
          })}
        </div>
      )}

      {/* **والخروجُ آخرُها بحدٍّ فوقه** — أخطرُ بندٍ فيها، **وحدٌّ يفصله
          عن الروابط يمنع أن يُضغط بامتداد الإصبع.** */}
      <button
        type="button"
        role="menuitem"
        onClick={() => {
          setOpen(false);
          onLogout();
        }}
        className="flex w-full items-center gap-2.5 border-t border-line-soft px-3 py-2.5 text-start text-sm text-danger transition-colors hover:bg-row-hover"
      >
        <IconLogout size={17} />
        {logoutLabel}
      </button>
    </div>
  );

  return (
    <>
      <button
        ref={btn}
        type="button"
        onClick={() => setOpen((v) => !v)}
        aria-haspopup="menu"
        aria-expanded={open}
        className={`taparea flex shrink-0 items-center gap-2 rounded-control p-1 transition-colors ${
          open || inMenu ? "bg-row-hover text-ink" : "text-ink-muted hover:bg-row-hover hover:text-ink"
        }`}
      >
        <Avatar url={avatarUrl} name={name} size={TOPBAR_AVATAR} />
        {/* **والاسمُ يُقصّ ولا يمدّ الشريط**: أسماءٌ ثلاثيّةٌ تدفع ما بعدها
            خارجَ الشاشة. */}
        <span className="max-w-20 truncate text-sm font-medium sm:max-w-28">{name}</span>
        <IconChevronDown
          size={15}
          className={`shrink-0 transition-transform ${open ? "rotate-180" : ""}`}
        />
      </button>
      {open && createPortal(menu, document.body)}
    </>
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
   * **ووجودُها لا طولُها هو ما يبدّل الشكل**: اللوحاتُ الأربعُ تمرّر مصفوفةً
   * **فارغةً** عمداً (قرارُ المالك ٢٠٢٦-٠٨-٠٦: «طبّق الاسم والقائمة مع باقي
   * اللوحات وأضِف فقط تسجيل الخروج للقائمة») — **فتأخذ الاسمَ والقائمةَ
   * وفيها الحسابُ والخروج، ولا بنودَ زائدة.**
   *
   * **ولو فُحص الطولُ لَسقطت اللوحاتُ إلى الشكل القديم صامتةً** — وهي عائلةُ
   * الخلل التي تتكرّر هنا: **شرطٌ يقيس الحجمَ وهو يريد الوجود.**
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
      {menu ? (
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
      {!menu && (
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
