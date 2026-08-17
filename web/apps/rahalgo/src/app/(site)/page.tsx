/**
 * **الصفحةُ الرئيسيّة — واجهةُ الموقع، لا صورةٌ في خلفيّة.**
 *
 * # ما كانت
 *
 * **كانت `/` صفحةَ التسوّق نفسَها**، ثمّ أُفرغت بقرار المالك (٢٠٢٦-٠٨-٠٦)،
 * **ثمّ صارت صورةً واحدةً تُرفع من الإعدادات** (٢٠٢٦-٠٨-٠٩).
 *
 * **والصورةُ لم تُرفع قطّ**: ناداها القياسُ فردَّت `{"image":null}` —
 * **فكانت الصفحةُ بيضاءَ عند كلّ زائرٍ منذ كُتبت.** ولم تكن في اللوحة شاشةٌ
 * ترفعها أصلاً.
 *
 * # وما صارت
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-١٦: «الموقع لازم يكون الرئيسية · تسوّق · من نحن ·
 *  تواصل معنا · ميزاتنا · لماذا اسم منصّتنا… رح نلغي موضوع الصورة ونخلّيه
 *  موقعاً احترافيّاً».)
 *
 * **ثمّ حُذفت أقسامُها الخمسة** (قرارُ المالك ٢٠٢٦-٠٨-١٧: «احذف قسم
 * ميزاتنا وكيف تطلب ولماذا رحّال غو وانضمّ إلينا وتواصل معنا») —
 * **فبقيت الافتتاحيّةُ وحدَها**: سلايدرٌ ومشهدٌ يُكتب حرفاً حرفاً.
 *
 * **ونصوصُها تبقى في المعجم** — لم تُحذف: **من أعادها غداً يعيدها
 * برسمها، ومن حذفها اليوم يكتبها من جديد.**
 *
 * # ولماذا تُقدَّم من الخادم
 *
 * **لا نداءَ فيها ولا حالة** — نصٌّ وأيقونات. **وصفحةٌ تُبنى في المتصفّح
 * يقرؤها غوغل فارغةً**، وهي الصفحةُ الوحيدةُ التي يهمّنا أن تُفهرَس.
 * **واسمُ المنصّة وحدَه يأتي من الخادم** (`fetchPlatform`) كما في الغلاف.
 *
 * # والحشوةُ الجانبيّة هنا لا في السوق
 *
 * **قاعدةُ المالك «لا حشوةَ يميناً ويساراً» للسوق** — وأكّدها للموقع
 * التعريفيّ بتخصيص (٢٠٢٦-٠٨-١٦): **الأشرطةُ تبلغ الحافّة، والنصُّ داخلها
 * يُحدَّد.** **وسطرٌ يمتدّ عبر شاشةٍ عريضةٍ لا يُقرأ** — تضيع العينُ في
 * رجوعها إلى أوّل السطر التالي.
 */

import Link from "next/link";
import HeroStage from "./HeroStage";
import { getMessages, defaultLocale, withPlatform } from "@rahalgo/i18n";
import {
  fetchPlatform,
  BannerSlider,
  NetworkFx,
  IconUser,
  IconStore,
  IconMoto,
  IconUsers,
  IconList,
  IconBag,
  IconSearch,
  IconRoute,
  IconShieldX,
  IconShieldCheck,
  IconReceipt,
  IconHourglass,
  IconSteps,
  IconHandshake,
  IconWallet,
  IconApp,
  IconNote,
  IconTarget,
  IconView,
  ButtonLink,
  IconRoles,
} from "@rahalgo/ui";

const API = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";
const m = getMessages(defaultLocale);
const H = m.site.homePage;

/** **والأيقوناتُ من لوسيد بأسماءٍ موحّدة** — ولا نوعَ مصدَّراً لها، فيُوصف بما يُستعمل. */
type Icon = React.ComponentType<{ size?: number; className?: string }>;

/**
 * **شريطٌ يبلغ الحافّة ونصُّه محدَّد.**
 *
 * **و`-mx-3` تُلغي حشوةَ الغلاف** — الغلافُ يحشو كلَّ صفحاته `px-3`،
 * **وشريطٌ ملوّنٌ يقف قبل الحافّة بثلاثة أرباعِ سنتيمترٍ يُقرأ عطباً**، لا
 * تصميماً.
 */
function Band({
  children,
  flush,
  relative,
}: {
  children: React.ReactNode;
  /** **موضعٌ نسبيٌّ لِما يُطلق داخلَه** — شبكةُ الدبابيس تقع عليه. */
  relative?: boolean;
  /**
   * **بلا حشوةٍ ولا حدٍّ للعرض** — (قاعدةُ المالك، أعادها ٢٠٢٦-٠٨-١٧:
   * «ما زال هناك بادينغ على اليمين»).
   *
   * **والافتتاحيّةُ وحدَها**: الأقسامُ تحتها بطاقاتٌ في شبكة، **وشبكةٌ
   * تلامس الحافّةَ تُقرأ مقصوصة.**
   */
  flush?: boolean;
}) {
  return (
    <section
      className={`-mx-3 sm:-mx-4 ${relative ? "relative overflow-hidden" : ""} ${
        flush
          ? /* **والافتتاحيّةُ ترتفع** — (طلبُ المالك ٢٠٢٦-٠٨-١٧: «ارفع
               المحتوى للأعلى قليلاً»): **حشوةٌ علويّةٌ أقلُّ من
               السفليّة**، فتبدأ الشاشةُ بالكلام لا بالفراغ. */
            /* **وفجوةٌ بسيطةٌ فوق السلايدر.**

               (رأيُ المالك ٢٠٢٦-٠٨-١٧: «البادينغ كبيرٌ جدّاً بين التوب بار
                والسلايدر… نستفيد من البادينغ».)

               **وقِيست فكانت مئةً واثنتي عشرة**: اثنتان وسبعون من غلاف
               الموقع — **وهي فسحةُ ما يتدلّى من الشعار** ولا تُمسّ لأنّها
               لكلّ صفحات الموقع — **وأربعون من هذا الشريط.**

               **فتُخفض حشوتُه ويُسحب الشريطُ إلى أعلى** بهامشٍ سالبٍ يأكل
               نصفَ فسحة الغلاف. **والشعارُ لا يقع على السلايدر**: هو عند
               الحافّة اليمنى واللافتةُ متوسّطةٌ بسقفِ ألفٍ ومئة، **فبينهما
               فراغٌ في كلّ شاشةٍ عريضة** — ولذلك السحبُ على الكبيرة
               وحدَها. */
              "pb-14 pt-2 sm:pb-20 sm:pt-3 lg:-mt-8 lg:min-h-[36rem]"
          : "py-14 sm:py-20"
      } ${flush ? "px-6 sm:px-12" : "px-3 sm:px-4"}`}
    >
      {/* **وبلا حدٍّ للعرض في الافتتاحيّة** — الحشوةُ وحدَها تُبعده عن
          الحافّة، **وحدُّ عرضٍ على شاشةٍ عريضةٍ يترك فجوةً يمينَ النصّ**
          وهي ما شكا منها. */}
      <div className={flush ? "w-full" : "mx-auto w-full max-w-6xl"}>{children}</div>
    </section>
  );
}

/**
 * ══════════════════════════════════════════════════════════════════════
 * **لافتاتُ العرض الافتتاحيّ — من مكانها لا من مكانٍ ثانٍ**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (طلبُ المالك ٢٠٢٦-٠٨-١٧: «أوّلُ قسمٍ خلّيه سلايدر بالرئيسيّة، نعرض عليه
 *  صوراً يطلع احترافيّ أكثر».)
 *
 * **ولافتاتُها غيرُ لافتات التسوّق** — (تصحيحُ المالك ٢٠٢٦-٠٨-١٧: «بانرات
 * صفحة التسوّق مختلفة برأيي عن الرئيسيّة»): **صفحةٌ تعرّف بالمنصّة وصفحةٌ
 * تبيع، وصورةٌ تصلح لإحداهما لا تصلح للأخرى.**
 *
 * **والجدولُ والشاشةُ والسلايدرُ واحدةٌ ويفرّقها عمودُ موضع** — **وجدولان
 * بالحقول نفسِها يعنيان شاشتين ومسارين** لأجل كلمة.
 *
 * **وفشلُ الجلب يُرجع فراغاً**: الرئيسيّةُ تبقى بعنوانها ودعوتها،
 * **وشريطُ خطأٍ في أوّل ما يراه زائرٌ أسوأُ من سلايدرٍ غائب.**
 */
async function fetchBanners(): Promise<Slide[]> {
  try {
    /* ══════════════════════════════════════════════════════════════
       **ولا تُخزَّن** — اللافتاتُ تُبدَّل من اللوحة.
       ══════════════════════════════════════════════════════════════

       (شكوى المالك ٢٠٢٦-٠٨-١٧: «بدّلتُ الصورة وأضفتُ صورةً مختلفةً وما
        زال يجلب الصورةَ القديمة… وحتّى لو حذفتُ صورةَ السلايدر يأخذ
        وقتاً لتبديلها وإزالتها».)

       **وكانت `revalidate: 300`** — **فخمسُ دقائقَ بين ما يفعله في
       اللوحة وما يراه في الموقع.** ومن رفع صورةً ثمّ فتح الصفحةَ فلم
       يرها **يظنّ الرفعَ فشل فيرفع ثانيةً**، ومن حذف لافتةً يراها باقيةً
       فيحذفها مرّةً أخرى.

       **والهويّةُ تُقرأ `no-store` للسبب نفسِه** منذ كُتبت: «صفحةٌ
       مخزَّنةٌ تعرض شعاراً حُذف». **واللافتةُ أَولى**: تُبدَّل أكثرَ من
       الشعار.

       **وثمنُه نداءٌ خفيفٌ في كلّ فتحة** — نقطةٌ تردّ صفّاً أو صفّين،
       **وهي على الخادم نفسِه لا عبر الشبكة.** */
    const res = await fetch(`${API}/api/v1/public/banners?at=home`, {
      cache: "no-store",
    });
    if (!res.ok) return [];
    const j = (await res.json()) as {
      data?: { banners?: { id: string; title: string; image_url: string | null; target: string | null }[] };
    };
    return (j.data?.banners ?? [])
      .filter((b) => b.image_url)
      .map((b) => ({
        id: b.id,
        title: b.title,
        imageUrl: API + b.image_url,
        href: b.target || undefined,
      }));
  } catch {
    // @empty-ok — انظر أعلاه: الفراغُ قرارٌ لا صمت.
    return [];
  }
}

interface Slide {
  id: string;
  title: string;
  imageUrl: string;
  href?: string;
}

/**
 * ══════════════════════════════════════════════════════════════════════
 * **صفُّ المقابلة — تجربتان متقابلتان بأيقونتين من عائلةٍ واحدة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (مواصفةُ المالك ٢٠٢٦-٠٨-١٧ بستّة صفوفٍ ونصوصِها وأيقوناتها.)
 *
 * **والزوجُ من عائلةٍ واحدة**: درعٌ بخطأٍ ودرعٌ بعلامة، ساعةٌ رمليّةٌ
 * وخطواتٌ مؤشَّرة — **فيُقرأ الفرقُ قبل قراءة النصّ**، ومن عائلتين يُقرأ
 * شيئين لا علاقةَ بينهما.
 *
 * **واللونان يقولان أيُّهما أيّ**: المعتادةُ باهتةٌ بحدٍّ خفيف،
 * **وتجربتُنا بنبرة المنصّة** — والرقمُ وحدَه برتقاليٌّ، وهي «اللمسةُ»
 * التي طلبها لا لوناً ثالثاً يزاحم.
 */
function DiffRow({
  n,
  name,
  Them,
  Us,
  themTitle,
  themBody,
  usTitle,
  usBody,
}: {
  n: string;
  name: string;
  Them: React.ComponentType<{ size?: number; className?: string }>;
  Us: React.ComponentType<{ size?: number; className?: string }>;
  themTitle: string;
  themBody: string;
  usTitle: string;
  usBody: string;
}) {
  return (
    <li className="flex flex-col gap-3">
      <span className="flex items-center gap-2">
        <span className="figure text-brandmark">{n}</span>
        <span className="heading-card text-ink">{name}</span>
      </span>
      <div className="grid grid-cols-1 gap-3 md:grid-cols-2">
        {/* **المعتادةُ باهتةٌ ولا تُشتم** — نصفُ القرّاء يعيشونها اليوم. */}
        {/* **والعمودان زجاجٌ كسائر ألواح المنصّة** — (طلبُ المالك
            ٢٠٢٦-٠٨-١٧: «والكروت اجعلها شفّافة أيضاً»). **وسطحٌ مصمتٌ
            بين ألواحٍ زجاجيّةٍ يُقرأ غريباً عنها.**

            **والمعتادةُ تبقى أخفتَ حرفاً** — فيُعرف العمودان بلا رأس. */}
        <div className="site-card lift flex gap-4 p-6">
          <Them size={28} className="mt-0.5 shrink-0 text-ink-muted" />
          <div className="flex flex-col gap-1.5">
            <b className="heading-card text-ink-muted">{themTitle}</b>
            <p className="text-sm text-ink-muted">{themBody}</p>
          </div>
        </div>
        <div className="site-card lift flex gap-4 p-6">
          <Us size={28} className="mt-0.5 shrink-0 text-accent-text" />
          <div className="flex flex-col gap-1.5">
            <b className="heading-card text-accent-text">{usTitle}</b>
            <p className="text-sm text-ink">{usBody}</p>
          </div>
        </div>
      </div>
    </li>
  );
}

/**
 * ══════════════════════════════════════════════════════════════════════
 * **لماذا يختارنا كلُّ طرف — أربعةُ ألواحٍ لأربعة أدوار**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (طلبُ المالك ٢٠٢٦-٠٨-١٧: «لماذا الزبونُ يختار رحّال غو، ثمّ المندوب،
 *  ثمّ السائق، ثمّ المتجر».)
 *
 * # ولا سطرَ فيها بلا ما يقابله في المحرّك
 *
 * **قِيس كلُّ ادّعاءٍ قبل أن يُكتب**: كودُ دعوة المندوب عمودٌ في `users`،
 * وعمولتُه إعدادٌ (`sales.commission_percent`)، **وورديّةُ السائق حقلٌ
 * يُقلَب** (`on_shift`)، وزرُّ طوارئه جدولٌ يُدرَج فيه، **ومستحقُّ المتجر
 * وسحبُه طلبٌ في `payout_requests`**، ومواعيدُه `merchant_hours`،
 * وبلاغُه عن سائقٍ مسارٌ في الدعم.
 *
 * **وموقعٌ يعد بما لا يفعله يُكتشف عند أوّل مستخدم** — ولا يُنسى.
 *
 * # وأربعتُها في شبكةٍ لا في أقسامٍ أربعة
 *
 * **أربعةُ أقسامٍ بعناوينَ متتاليةٍ تُقرأ صفحةً تُعيد نفسَها** — وهو ما
 * رفضه. **والشبكةُ تجعلها مقارنةً تُمسح بنظرة**: كلٌّ يجد لوحَه ويقرأ
 * أربعةَ أسطر.
 */
function WhyCard({
  Icon,
  who,
  sub,
  lines,
}: {
  Icon: React.ComponentType<{ size?: number; className?: string }>;
  who: string;
  /** **وعدُ الدور في سطر** — (نصُّ المالك ٢٠٢٦-٠٨-١٧). */
  sub: string;
  lines: string[];
}) {
  return (
    <div className="site-card lift flex flex-col gap-3 p-6">
      {/* ══════════════════════════════════════════════════════════════
          **ورأسُ اللوح متوسّطٌ والأيقونةُ في قرص**
          ══════════════════════════════════════════════════════════════

          (طلبُ المالك ٢٠٢٦-٠٨-١٧: «الزبون والسائق والمتجر والمندوب
           خلّيهم محاذاة وسط، مع إضافة دائرةٍ ليكون داخل الدائرة».)

          **والقرصُ يُعرّف الدورَ قبل اسمه** — ومن مسح الألواحَ الأربعةَ
          بنظرةٍ رأى أربعةَ رموزٍ لا أربعةَ عناوين.

          **والبنودُ تبقى إلى اليمين**: **سطرٌ من عشرِ كلماتٍ متوسّطٌ
          تتبدّل بدايتُه في كلّ سطرٍ فتضيع العينُ في رجوعها.** */}
      <span className="flex flex-col items-center gap-2 text-center">
        <span className="site-orb flex h-16 w-16 items-center justify-center border border-accent-edge bg-accent-tint text-accent-text">
          <Icon size={30} />
        </span>
        <h3 className="heading-page text-ink">{who}</h3>
        <b className="text-base text-accent-text">{sub}</b>
      </span>
      <ul className="flex flex-col gap-2.5 text-base text-ink-muted">
        {lines.map((t) => (
          <li key={t} className="flex gap-2">
            {/* **ونقطةٌ تفصل السطور** — **وأربعةُ أسطرٍ بلا فاصلٍ تُقرأ
                فقرةً واحدة.** */}
            <span aria-hidden className="mt-2 h-1 w-1 shrink-0 rounded-full bg-accent" />
            {t}
          </li>
        ))}
      </ul>
    </div>
  );
}

/**
 * **خطوةٌ في مسار الطلب** — رقمٌ وأيقونةٌ واسم.
 *
 * (مواصفةُ المالك ٢٠٢٦-٠٨-١٧: «كيف تعمل المنظومة؟» بخمسِ خطوات.)
 *
 * **والرقمُ هنا معنًى لا زينة**: لا يُستلم قبل أن يُجهَّز، **ولا تُحتسب
 * المستحقّاتُ قبل أن يصل.**
 */
function FlowStep({
  n,
  Icon,
  label,
}: {
  n: number;
  Icon: React.ComponentType<{ size?: number; className?: string }>;
  label: string;
}) {
  return (
    <li className="relative flex flex-1 flex-col items-center gap-2 text-center">
      {/* ══════════════════════════════════════════════════════════════
          **والطريقُ يصل الدائرتين ولا يمرّ خلفَهما**
          ══════════════════════════════════════════════════════════════

          (تصحيحُ المالك ٢٠٢٦-٠٨-١٧: «الطريق اجعله احترافيّاً، لا يمرّ من
           منتصف الدائرة أو من خلفها… وكأنّه يصل الدوائرَ فقط».)

          **وكان خيطاً واحداً يمتدّ تحت الخمس** — **فيُرى داخلَ كلّ دائرةٍ
          لأنّها شفّافة**، ويُقرأ خطّاً مرسوماً عليها لا طريقاً بينها.

          **فصار قطعةً لكلّ خطوةٍ بعد الأولى**، طرفاها حافّتا الدائرتين
          بالضبط: **نصفُ عرض الخطوة ناقصَ نصفَ قطر الدائرة** من الجهتين —
          **يُحسبان من المقاس نفسِه فلا ينكسران إذا تبدّل عددُ الخطوات.**

          **ومنطقيّان لا يمينٌ ويسار** — ينقلبان مع اللغة بأنفسهما. */}
      {n > 1 && (
        <>
          <span
            aria-hidden
            className="flow-path pointer-events-none absolute top-8 hidden h-px sm:block"
            style={{
              insetInlineStart: "calc(-50% + 0.5rem)",
              insetInlineEnd: "calc(50% + 2rem)",
            }}
          />
          {/* **وعلى الجوّالِ الفجوةُ وحدَها** — الخطواتُ عمودٌ، **والفجوةُ
              بين دائرتين هي كلُّ ما يُوصَل.** */}
          <span
            aria-hidden
            className="flow-path-y pointer-events-none absolute -top-6 h-6 w-px sm:hidden"
            style={{ insetInlineStart: "calc(50% - 0.5px)" }}
          />
        </>
      )}
      <span className="lift relative flex h-16 w-16 items-center justify-center rounded-full border border-accent-edge bg-accent-tint text-accent-text">
        <Icon size={26} />
        <span className="figure absolute -top-2 -end-2 flex h-6 w-6 items-center justify-center rounded-full bg-accent text-2xs text-on-bright">
          {n}
        </span>
      </span>
      <span className="text-sm font-bold text-ink">{label}</span>
    </li>
  );
}

/**
 * **لوحُ نصٍّ أو قرصُ قيمة.**
 *
 * (طلبُ المالك ٢٠٢٦-٠٨-١٧: «اجعلها دوائرَ بدل المربّعات».)
 *
 * **والقرصُ للكلمة الواحدة** — «الثقة» و«السرعة»: **دائرةٌ فيها كلمةٌ
 * تُقرأ وسماً، ومربّعٌ فيه كلمةٌ يُقرأ بطاقةً ناقصة.**
 *
 * **واللوحُ للفقرة** — قصّةٌ في قرصٍ تخرج عن حدّه أو تصغّر حرفَها.
 */
function NoteCard({
  Icon,
  title,
  tone = "text-ink",
  body,
  orb,
}: {
  Icon?: React.ComponentType<{ size?: number; className?: string }>;
  title: string;
  /** **ولكلّ عنوانٍ لونُه** — (طلبُ المالك ٢٠٢٦-٠٨-١٧). */
  tone?: string;
  body?: string;
  /** **قرصٌ لا لوح** — للكلمة الواحدة. */
  orb?: boolean;
}) {
  return (
    <div
      className={
        orb
          ? "site-card site-orb lift mx-auto flex aspect-square w-full max-w-[11rem] flex-col items-center justify-center gap-2 p-4 text-center"
          : "site-card lift flex flex-col items-center gap-3 p-6 text-center"
      }
    >
      {/* **والأيقونةُ في قرصٍ كألواح الأدوار** — (طلبُ المالك ٢٠٢٦-٠٨-١٧:
          «طبّقها أيضاً على مهمّتنا وقصّتنا ورؤيتنا»). **وفي القرص لا
          تُحاط**: هو قرصٌ بنفسه. */}
      {Icon &&
        (orb ? (
          <Icon size={30} className="text-accent-text" />
        ) : (
          <span className="site-orb flex h-16 w-16 items-center justify-center border border-accent-edge bg-accent-tint text-accent-text">
            <Icon size={30} />
          </span>
        ))}
      <h3 className={`${orb ? "heading-card" : "heading-page"} ${tone}`}>{title}</h3>
      {body && <p className="text-base text-ink-muted">{body}</p>}
    </div>
  );
}

export default async function HomePage() {
  const [brand, slides] = await Promise.all([fetchPlatform(API), fetchBanners()]);
  const name = brand.name;


  return (
    <>
      {/* ══════════════════════════════════════════════════════════════
          **العرضُ الافتتاحيّ — جملةٌ تقول ما نفعل وزرٌّ يبدأ**
          ══════════════════════════════════════════════════════════════

          **ولا زرَّ «حمّل التطبيق» بعد**: التطبيقاتُ لم تُنشر على غوغل بلاي
          **وزرٌّ يعد بما لا يوجد يُفقد الثقةَ في أوّل شاشة.** يُضاف يومَ
          النشر. */}
      <Band flush>
        {/* **والصورةُ فوق الكلام** — **وعنوانٌ يُكتب فوق صورةٍ يرفعها
            صاحبُها لا يُضمَن أن يُقرأ**: صورةٌ فاتحةٌ تبتلع الحرفَ الأبيضَ
            وداكنةٌ تبتلع الأسود. **فالسلايدرُ يعلو والكلامُ تحته على أرضِ
            الصفحة** — كلٌّ منهما يُقرأ على حدة.

            **ولا يُرسم إن لم تُرفع لافتة** — إطارٌ فارغٌ يُقرأ عطباً. */}
        {/* **ويتخطّى حشوةَ الشريط ليبلغ الحافّتين** — (شكوى المالك
            ٢٠٢٦-٠٨-١٧: «بحجم الصفحة من اليمين إلى اليسار»). **والحشوةُ
            للنصّ لا للصورة.** */}
        {/* ══════════════════════════════════════════════════════════
            **وحاويةٌ مركزيّةٌ لا حافّةٌ إلى حافّة**
            ══════════════════════════════════════════════════════════

            (مواصفةُ المالك ٢٠٢٦-٠٨-١٧: «داخل Container مركزيّ أنيق مع
             مساحةٍ واضحةٍ يمين ويسار… وليس بانراً ملتصقاً بحواف
             الشاشة».)

            **وحشوةُ الشريط تُلغى ثمّ تُعطى الفجوةُ الخاصّة** — الشريطُ
            يحشو نصَّه بمقدارٍ ثابت، **واللافتةُ فجوتُها تتدرّج مع
            العرض.** */}
        {slides.length > 0 && (
          <div className="banner-gutter -mx-6 mb-10 sm:-mx-12">
            <BannerSlider
              className="banner-shell w-full"
              items={slides}
              Link={Link}
              hero
            />
          </div>
        )}
        {/* **والنصُّ إلى اليمين لا في الوسط** — (طلبُ المالك ٢٠٢٦-٠٨-١٧:
            «النصّ كامل يصبح محاذاة إلى اليمين»).

            **و`items-start` لا `items-end`**: البدايةُ منطقيّةٌ تنقلب مع
            اللغة — **ومن كتب «يمين» بالحرف كسر صفحتَه يومَ تُقرأ
            بالإنجليزيّة.** */}
        {/* ══════════════════════════════════════════════════════════
            **والشبكةُ تتوسّط المحتوى لا الشريطَ كلَّه**
            ══════════════════════════════════════════════════════════

            (شكوى المالك ٢٠٢٦-٠٨-١٧: «إذا وُجد سلايدر المفروض المحتوى
             ينزل تحته ولا يصبح مخفيّاً تحت السلايدر».)

            **وكانت مُطلَقةً على الشريط** — والسلايدرُ جزءٌ منه، **فيقع
            نصفُها العلويُّ خلفَه** كلَّما رُفعت لافتة. **وتوسّطُها ما
            حولَها لا ما فوقَها** — فهي زخرفةُ الكلام لا زخرفةُ الصورة. */}
        {/* ══════════════════════════════════════════════════════════
            **ولوحةُ الشبكة يسارَ الكلام**
            ══════════════════════════════════════════════════════════

            (مواصفةُ المالك ٢٠٢٦-٠٨-١٧.)

            **وعرضُها يتدرّج ولا يقفز** — والرسمُ داخلَها نسبيٌّ بـ
            `viewBox`، **فيبقى على نسبته في كلّ شاشةٍ بلا كسر.**

            **وتُخفى دون اللابتوب**: الافتتاحيّةُ هناك عمودٌ يملؤه
            الكلام، **ولوحةٌ تحته تدفع ما يُقرأ خارجَ الشاشة.** */}
        <div className="relative">
          <div
            aria-hidden
            /* **وتنزل عن منتصف الكلام** — (طلبُ المالك ٢٠٢٦-٠٨-١٧:
               «والشكلُ كاملاً أنزله إلى الأسفل بشكلٍ ملحوظ»).

               **وبالنسبة لا بالبكسل**: المحتوى يطول ويقصر بمقاس الشاشة،
               **ورقمٌ ثابتٌ يصلح لواحدةٍ ويخرج عن الباقي.** */
            className="pointer-events-none absolute top-[82%] hidden -translate-y-1/2 lg:block"
            style={{ insetInlineEnd: "4%", width: "min(30vw, 26rem)" }}
          >
            <NetworkFx />
          </div>
          <HeroStage name={name} />
        </div>
      </Band>

      {/* **وقسمُ الفرق بعد الافتتاحيّة** — (سؤالُ المالك ٢٠٢٦-٠٨-١٧). */}
      {/* **وقسمُ المقابلة** — (مواصفةُ المالك ٢٠٢٦-٠٨-١٧). */}
      <Band>
        <div className="mx-auto max-w-5xl">
          <div className="mb-8 text-center">
            <h2 className="heading-display text-accent">{H.diffTitle}</h2>
            <p className="mt-3 text-sm text-ink-muted">{H.diffLead}</p>
          </div>
          {/* **ورأسان يقولان أيُّ عمودٍ لمن** — على الحاسوب وحدَه:
              **والعمودان يصيران صفّين على الجوّال فيُقرأ كلُّ زوجٍ معاً.** */}
          <div className="mb-3 hidden grid-cols-2 gap-3 text-2xs md:grid">
            <span className="px-4 text-ink-muted">{H.diffThem}</span>
            <span className="px-4 font-bold text-accent-text">{H.diffUs}</span>
          </div>
          <ul className="flex flex-col gap-6">
            <DiffRow
              n="01"
              name={H.r1n}
              Them={IconList}
              Us={IconBag}
              themTitle={H.r1at}
              themBody={H.r1ab}
              usTitle={H.r1bt}
              usBody={H.r1bb}
            />
            <DiffRow
              n="02"
              name={H.r2n}
              Them={IconSearch}
              Us={IconRoute}
              themTitle={H.r2at}
              themBody={H.r2ab}
              usTitle={H.r2bt}
              usBody={H.r2bb}
            />
            <DiffRow
              n="03"
              name={H.r3n}
              Them={IconShieldX}
              Us={IconShieldCheck}
              themTitle={H.r3at}
              themBody={H.r3ab}
              usTitle={H.r3bt}
              usBody={H.r3bb}
            />
            <DiffRow
              n="04"
              name={H.r4n}
              Them={IconReceipt}
              Us={IconWallet}
              themTitle={H.r4at}
              themBody={H.r4ab}
              usTitle={H.r4bt}
              usBody={H.r4bb}
            />
            <DiffRow
              n="05"
              name={H.r5n}
              Them={IconHourglass}
              Us={IconSteps}
              themTitle={H.r5at}
              themBody={H.r5ab}
              usTitle={H.r5bt}
              usBody={H.r5bb}
            />
            <DiffRow
              n="06"
              name={H.r6n}
              Them={IconUsers}
              Us={IconHandshake}
              themTitle={H.r6at}
              themBody={H.r6ab}
              usTitle={H.r6bt}
              usBody={H.r6bb}
            />
          </ul>
        </div>
      </Band>
      {/* **ولماذا يختارنا كلُّ طرف** — (طلبُ المالك ٢٠٢٦-٠٨-١٧). */}
      <Band>
        <div className="mx-auto max-w-6xl">
          <div className="mb-8 text-center">
            <h2 className="heading-display text-primary">{H.whyTitle}</h2>
            <p className="mt-2 text-sm text-ink-muted">{H.whyLead}</p>
          </div>
          <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
            <WhyCard Icon={IconUser} who={H.whyCustomer} sub={H.whyCustomerSub} lines={[H.wc1, H.wc2, H.wc3, H.wc4]} />
            <WhyCard Icon={IconStore} who={H.whyStore} sub={H.whyStoreSub} lines={[H.ws1, H.ws2, H.ws3, H.ws4]} />
            <WhyCard Icon={IconMoto} who={H.whyDriver} sub={H.whyDriverSub} lines={[H.wd1, H.wd2, H.wd3, H.wd4]} />
            <WhyCard Icon={IconUsers} who={H.whyRep} sub={H.whyRepSub} lines={[H.wr1, H.wr2, H.wr3, H.wr4]} />
          </div>
        </div>
      </Band>
      {/* **كيف تعمل المنظومة** — (مواصفةُ المالك ٢٠٢٦-٠٨-١٧). */}
      <Band>
        <div className="mx-auto max-w-5xl">
          <h2 className="heading-display mb-8 text-center text-violet">{H.flowTitle}</h2>
          {/* **وتصير عموداً على الجوّال** — خمسُ خطواتٍ في صفٍّ على
              ثلاثمئةٍ وستّين تُقرأ حروفاً متراكمة. */}
          <ol className="relative flex flex-col items-center gap-6 sm:flex-row sm:items-start">
            <FlowStep n={1} Icon={IconApp} label={H.f1} />
            <FlowStep n={2} Icon={IconSteps} label={H.f2} />
            <FlowStep n={3} Icon={IconMoto} label={H.f3} />
            <FlowStep n={4} Icon={IconRoute} label={H.f4} />
            <FlowStep n={5} Icon={IconWallet} label={H.f5} />
          </ol>
        </div>
      </Band>

      {/* **قيمنا** — أربعُ كلماتٍ لا شرحَ لها: **الشرحُ يُضعفها.** */}
      <Band>
        <div className="mx-auto max-w-4xl">
          <h2 className="heading-display mb-8 text-center text-info">{H.valuesTitle}</h2>
          <div className="grid grid-cols-2 gap-4 md:grid-cols-4">
            <NoteCard Icon={IconHandshake} title={H.v1} orb />
            <NoteCard Icon={IconMoto} title={H.v2} orb />
            <NoteCard Icon={IconSearch} title={H.v3} orb />
            <NoteCard Icon={IconShieldCheck} title={H.v4} orb />
          </div>
        </div>
      </Band>

      {/* **قصّتنا ومهمّتنا ورؤيتنا** — ثلاثةٌ في صفٍّ واحد. */}
      <Band>
        <div className="mx-auto grid max-w-5xl grid-cols-1 gap-4 md:grid-cols-3">
          <NoteCard Icon={IconNote} title={H.storyTitle} tone="text-accent" body={H.storyBody} />
          <NoteCard Icon={IconTarget} title={H.missionTitle} tone="text-primary" body={H.missionBody} />
          <NoteCard Icon={IconView} title={H.visionTitle} tone="text-violet" body={H.visionBody} />
        </div>
      </Band>
    </>
  );
}
