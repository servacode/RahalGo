"use client";

/**
 * **عرضُ الصفحة الرئيسيّة — صورةُ خلفيّةٍ وفوقها ما يُقرأ ويُضغط.**
 *
 * (طلبُ المالك ٢٠٢٦-٠٨-٠٩: «النصّ اللي بالصورة بدّي ياه يُطبَّق بنفس الأسلوب
 *  في صورتَي الخلفيّة، والزرّ يكون هو نفسه زرّ التحميل بالأعلى — إذا موجود
 *  رابطٌ أو ملفٌّ يطلع وإذا مو موجود يختفي».)
 *
 * # ولماذا يُبنى النصُّ ولا يبقى مطبوعاً في الصورة
 *
 * **الحرفُ المطبوعُ صورة**: يخفت حين تُصغَّر، **ولا يُقرأ في بحثٍ ولا يسمعه
 * قارئٌ صوتيّ**، ولا يُنسخ. **وزرٌّ مرسومٌ في صورةٍ لا يُضغط** — ومن ضغطه
 * لم يقع شيء.
 *
 * **والصورةُ تبقى مشهداً** (المدينةُ والسائقُ والهاتف) — **وهو ما تُحسنه
 * الصورة**، والحرفُ فوقها حقيقيّ.
 *
 * # وزرُّ التحميل واحدٌ لا اثنان
 *
 * **من الشريط نفسِه**: `appUrl` تحمل رابطَ المتجر إن ضُبط، **وإلّا مسارَ
 * الملفّ المرفوع** — والخادمُ يختار بينهما (`appHref`). **وفارغُها يُخفي
 * الزرَّ كلَّه**: زرُّ تحميلٍ لا ينزّل شيئاً يُقرأ عطباً في المنصة لا ميزةً
 * ناقصة.
 *
 * # والنصُّ من المعجم لا من الإعدادات
 *
 * **كان أربعةَ حقولٍ في لوحة الإدارة فحُذفت بطلبه** — وهي نسخةُ تسويقٍ تُكتب
 * مرّةً، **لا شيءٌ يُضبط في كلّ موسم.** فموطنُها المعجمُ كأخواتها.
 */

import type { CSSProperties } from "react";
import { getMessages, defaultLocale, withPlatform } from "@rahalgo/i18n";
import { usePlatform } from "./platform";
import { IconLocation, IconWallet, IconMoto, IconSupport } from "./icons";

const m = getMessages(defaultLocale);
const H = m.site.home;

export interface HeroContent {
  imageUrl: string | null;
  /** **صورةُ الجوّال** — وفارغُها يسقط إلى العريضة. */
  imageMobileUrl: string | null;
}

/** **أربعُ مزايا** — أيقونةٌ فوق كلمةٍ، كما في تصميم المالك. */
const FEATURES = [
  { Icon: IconLocation, label: H.featTrack },
  { Icon: IconWallet, label: H.featCash },
  { Icon: IconMoto, label: H.featFast },
  { Icon: IconSupport, label: H.featSupport },
];

export function Hero({ content }: { content: HeroContent }) {
  const { imageUrl, imageMobileUrl } = content;
  const { name, appUrl } = usePlatform();
  const hasImage = Boolean(imageUrl || imageMobileUrl);

  return (
    /* **والحشوةُ تُلغى بسالبها**: `<main>` يعطي كلَّ صفحةٍ حشوةً — **وهي حقٌّ
       لأخواتها**، فتُطرح هنا وحدَها بدل أن تُنزع منه. **والنصُّ يستردّها
       داخلَه** فلا يلتصق بالحافّة. */
    <div className="relative isolate -mx-3 -mb-4 -mt-12 flex flex-1 items-center sm:-mx-4 sm:-mt-[4.5rem]">
      {hasImage && (
        <div
          aria-hidden
          className="home-bg-image"
          style={
            {
              ...(imageUrl ? { "--home-bg": `url(${imageUrl})` } : {}),
              ...(imageMobileUrl ? { "--home-bg-mobile": `url(${imageMobileUrl})` } : {}),
            } as CSSProperties
          }
        />
      )}

      {/* **وحجابٌ من جهة النصّ** — يعمّ الطرفَ الأيسرَ ويذوب نحو الوسط،
          **فيُقرأ الحرفُ ويبقى المشهدُ صافياً حيث لا حرفَ عليه.** */}
      <span
        aria-hidden
        className="pointer-events-none absolute inset-0 bg-gradient-to-r from-scrim to-transparent"
      />

      {/* ══════════════════════════════════════════════════════════════
          **والنصُّ إلى يسار الشاشة لا يمينها**
          ══════════════════════════════════════════════════════════════

          (قرارُ المالك ٢٠٢٦-٠٨-٠٩: «شوف النصّ وين، وحرّكه بالمكان الصحيح —
           إلى يسار الشاشة».)

          **وكان في البداية** — وهي اليمينُ في مستندٍ عربيّ، **فوقع على
          السائق والصندوق**: أزحمُ ما في المشهد. **واليسارُ مدينةٌ بعيدةٌ
          داكنة** — خلفيّةٌ يُقرأ عليها الحرف.

          **و`ms-auto` تدفعه إلى النهاية** — ولا تُكتب `left` صراحةً:
          **الخصائصُ المنطقيّةُ تنقلب مع اللغة**، وموقعٌ لاتينيٌّ يوماً يقلبها
          إلى اليمين بلا تعديل. */}
      <div className="relative z-10 ms-auto w-full max-w-2xl px-5 py-8 sm:px-12">
        {/* **وكلمتان بالنبرة من أربع** — التباينُ يصنع الإيقاع، **وسطرٌ كلُّه
            ملوّنٌ لا يُبرز شيئاً.** (كما في تصميم المالك.) */}
        <h1 className="heading-hero text-on-solid">
          <span className="block">
            {H.line1Plain} <span className="text-gradient-brand">{H.line1Accent}</span>
          </span>
          <span className="block">
            {H.line2Plain} <span className="text-gradient-brand">{H.line2Accent}</span>
          </span>
        </h1>

        <p className="hero-lead mt-5 max-w-prose text-on-solid">
          {withPlatform(H.lead, name)}
        </p>

        {/* **وبطاقتان على الهاتف وأربعٌ فوقه** — أربعٌ في عرض ٣٦٠ بكسلاً
            تصير كلُّ واحدةٍ ثمانين بكسلاً **فتُقطع الكلمةُ حرفين حرفين.** */}
        <ul className="mt-6 grid grid-cols-2 gap-2 sm:grid-cols-4 sm:gap-3">
          {FEATURES.map(({ Icon, label }) => (
            <li
              key={label}
              className="flex flex-col items-center justify-center gap-2.5 rounded-card border border-line-soft bg-field p-4 text-center"
            >
              <Icon size={30} className="text-primary" />
              <span className="text-xs font-medium leading-tight text-on-solid sm:text-sm">
                {label}
              </span>
            </li>
          ))}
        </ul>

        {/* **وزرٌّ لا ينزّل شيئاً لا يُعرض** — والوجهةُ من الإعدادات: رابطُ
            المتجر إن ضُبط، وإلّا الملفُّ المرفوع. */}
        {appUrl && (
          <a
            href={appUrl}
            target="_blank"
            rel="noopener noreferrer"
            className="btn-brand mt-7 flex w-full items-center justify-center gap-3 rounded-badge px-5 py-4 text-on-solid transition-opacity hover:opacity-90 sm:py-5"
          >
            {/* **ولا سهمَ فيه** (قرارُ المالك ٢٠٢٦-٠٨-٠٩: «السهم برأيي ألغِه»).
                **وزرٌّ كلمتُه واضحةٌ لا يحتاج سهماً يقول «اضغط»** — والسهمُ
                إشارةُ انتقالٍ لا إشارةُ تنزيل. */}
            {H.download}
          </a>
        )}
      </div>
    </div>
  );
}
