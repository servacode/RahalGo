/**
 * **التذييل** — بابُ الصفحات التي لا تُطلب كلَّ يومٍ وتُطلب حين تُطلب.
 *
 * # ولماذا لا في الشريط
 *
 * الشريطُ العلويُّ لما يُستعمل في كلّ زيارة: السلّةُ والطلباتُ والمحفظة.
 * **والشروطُ تُقرأ مرّةً والمساعدةُ عند حيرة** — ووضعُهما في الشريط يزاحم ما
 * يُضغط يوميّاً، **وهو ضيّقٌ على الجوّال أصلاً.**
 *
 * # وأسفلَ الصفحة حيث يُبحث عنها
 *
 * **من فرغ من الصفحة ولم يجد جواباً ينزل** — وهي العادةُ المتّفق عليها في
 * كلّ موقع، **ومخالفتُها تجعل من يبحث لا يجد.**
 */

import Link from "next/link";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import {
  IconFacebook,
  IconInstagram,
  IconTelegram,
  IconWhatsApp,
} from "@rahalgo/ui";
const m = getMessages(defaultLocale);
const L = m.site.legal;

const LINKS = [
  { href: "/help", label: L.helpTitle },
  { href: "/contact", label: L.contactTitle },
  { href: "/terms", label: L.termsTitle },
  { href: "/privacy", label: L.privacyTitle },
];

type Social = { facebook: string; instagram: string; telegram: string; whatsapp: string };
const NO_SOCIAL: Social = { facebook: "", instagram: "", telegram: "", whatsapp: "" };

export default function Footer({
  /** **اسمُ المنصة من الإعدادات** — وفارغٌ يعني «خذ من المعجم». */
  name = "",
  /* **والحساباتُ تُمرَّر ولا تُقرأ هنا**: التخطيطُ ينادي `fetchPlatform`
     أصلاً، **ونداءٌ ثانٍ من التذييل رحلةٌ في كلّ صفحة** لِما بين يديه. */
  social = NO_SOCIAL,
}: { name?: string; social?: Social } = {}) {
  const brand = name || m.common.appName;

  const ACCOUNTS = [
    { href: social.facebook, Icon: IconFacebook, label: "Facebook" },
    { href: social.instagram, Icon: IconInstagram, label: "Instagram" },
    { href: social.telegram, Icon: IconTelegram, label: "Telegram" },
    {
      /* **وواتساب رقمٌ يُبنى منه رابط** — لا صفحةَ تُنسخ. */
      href: social.whatsapp ? `https://wa.me/${social.whatsapp.replace(/\D/g, "")}` : "",
      Icon: IconWhatsApp,
      label: "WhatsApp",
    },
  ].filter((a) => a.href);

  return (
    /* **وشريطٌ كالعلويّ لا كتلةٌ سائبة.**

       كان بلا حدٍّ ولا خلفيّة — **فيطفو في أسفل الصفحة كنصٍّ نُسي.** فصار
       له سطحٌ كالشريط العلويّ: **موقعٌ يبدأ بشريطٍ وينتهي بشريط.**

       **ولا حدَّ علويّاً** — (قرارُ المالك ٢٠٢٦-٠٨-٠٦): **السطحُ الزجاجيُّ
       يفصل بكثافته**، وخطٌّ فوقه يقطع الصورةَ التي يمرّ منها. */
    /* ══════════════════════════════════════════════════════════════════
       **وسطرٌ واحدٌ رفيعٌ لا كتلةٌ من ثلاثة**
       ══════════════════════════════════════════════════════════════════

       (قرارُ المالك ٢٠٢٦-٠٨-٠٩: «وزّع العناصر على سطرٍ واحد — صفحات على
        اليسار، حقوق الملكيّة بالوسط، وسائل التواصل على اليمين، لنعمل فوتر
        رفيع وليس عريضاً ضخماً».)

       **كان ثلاثةَ صفوفٍ متراكمة** فأخذ ربعَ الشاشة — **وهو ذيلٌ لا محتوى.**
       وارتفاعُه يقتطع من الصورة التي فوقه.

       **والترتيبُ في الشجرة يقلبه الاتّجاه**: الأوّلُ يقع يميناً في مستندٍ
       عربيّ — **فالحساباتُ أوّلاً لتقع يميناً، والصفحاتُ آخراً لتقع يساراً.**

       **وثلاثةُ أعمدةٍ متساويةٍ لا `justify-between`**: الأخيرةُ تجعل موضعَ
       الوسط يتحرّك بطول جاريه — **فيزيح اسمُ المنصة الطويلُ الحقوقَ عن
       المنتصف.** والشبكةُ تُثبّته مهما طال ما حولَه.

       **وعلى الضيّق تتراصّ** — ثلاثةُ أقسامٍ في عرض هاتفٍ تُقرأ حرفاً حرفاً. */
    <footer className="surface-lit grid grid-cols-1 items-center gap-2 bg-surface px-4 py-2.5 text-xs text-ink-muted sm:grid-cols-3">
      <div className="flex items-center justify-center gap-2 sm:justify-start">
        {ACCOUNTS.map(({ href, Icon, label }) => (
          <a
            key={label}
            href={href}
            target="_blank"
            rel="noreferrer noopener"
            aria-label={label}
            title={label}
            className="taparea flex h-7 w-7 items-center justify-center rounded-control text-ink-muted transition-colors hover:text-accent-text"
          >
            <Icon size={16} />
          </a>
        ))}
      </div>

      {/* **والسنةُ تُحسب في الخادم** — وهذا مكوّنُ خادم، فلا يختلف عليها
          متصفّحٌ ضُبط على سنةٍ أخرى في أوّل رسم.

          **والاسمُ يُكتب هنا بقرار المالك** (٢٠٢٦-٠٨-٠٩: «قصدتُ بالفوتر حقوق
          الملكيّة محفوظة») — **وكان محذوفاً بقراره السابق** (٢٠٢٦-٠٨-٠٦: «لا
          أريد أن تكتب اسم المنصة بأيّ مكانٍ أبداً… أنا سوف أخبرك أين يُكتب»).
          **وقد أخبر.** والاسمُ من الإعدادات لا من نصٍّ مكتوب. */}
      <p className="text-center">
        © {new Date().getFullYear()} {brand} — {L.rights}
      </p>

      <div className="flex flex-wrap items-center justify-center gap-x-4 gap-y-1 sm:justify-end">
        {LINKS.map((l) => (
          <Link key={l.href} href={l.href} className="transition-colors hover:text-accent-text">
            {l.label}
          </Link>
        ))}
      </div>
    </footer>
  );
}
