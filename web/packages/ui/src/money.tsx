import { fmtNum, getMessages, defaultLocale } from "@rahalgo/i18n";

const m = getMessages(defaultLocale);

/**
 * ══════════════════════════════════════════════════════════════════════
 * **مبلغٌ وعملتُه — بترتيبٍ واحدٍ في المنصّة كلِّها**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-١٢: «مكتوب ل.س ٣٠٠ وهذا غلط، لازم ٣٠٠ ل.س»
 *  · «حاول تحتفظ بالملاحظات وتطبّقها بشكلٍ مركزيّ، ما يصير نستخدم أكثر
 *  من مصلّحٍ لنفس الشيء».)
 *
 * # لماذا كان ينقلب
 *
 * **الرقمُ يُلَفّ بـ`dir="ltr"`** لأنّ فاصلةَ الآلاف والكسرَ يقعان بغير
 * موضعهما في نصٍّ عربيّ. **فإذا وقع الرمزُ داخل اللفّة نفسِها** صار
 * ترتيبُهما من اليسار: الرقمُ أوّلاً ثمّ الرمز — **وعينُ القارئ تمشي من
 * اليمين**، فتقع على «ل.س» قبل «٣٠٠».
 *
 * **والصوابُ لفّةٌ للرقم وحدَه**: يبقى الرقمُ سليماً داخلها، **ويبقى
 * ترتيبُه مع الرمز عربيّاً خارجَها.**
 *
 * # ولماذا مركّبةٌ لا سطرٌ يُكرَّر
 *
 * **سبعةٌ وثمانون موضعاً تكتب `{fmtNum(x)} {currency}` بيدها** — ومن
 * صحّح واحداً ترك ستّةً وثمانين. **وأوّلُ ما يفترق هو ما يُكتب مرّتين.**
 *
 * **ويومَ تُضاف عملةٌ ثانية** — أو يُبدَّل الرمز، أو يُطلب موضعُه قبل
 * الرقم — **يُبدَّل هذا الملفُّ وحدَه.**
 */
export function Money({
  value,
  /** **حجمُ الرمز** — أصغرُ من الرقم في الإجماليات، ومثلُه في السطور. */
  small = false,
  className = "",
}: {
  value: number;
  small?: boolean;
  className?: string;
}) {
  return (
    <span className={className}>
      {/* **واللفّةُ للرقم وحدَه** — فاصلتُه وكسرُه يقعان في موضعهما. */}
      <span dir="ltr" className="tabular-nums">
        {fmtNum(value)}
      </span>{" "}
      <span className={small ? "text-xs font-normal text-ink-muted" : ""}>
        {m.common.currency}
      </span>
    </span>
  );
}

/**
 * **الدولارُ للعرض** — «$1.8» لا «$1.80» (مرآةُ `dollars` في التطبيق).
 */
export function usdLabel(v: number): string {
  const cents = Math.round(v * 100);
  const text = cents % 100 === 0 ? String(cents / 100) : String(cents / 100).replace(/0+$/, "");
  return `$${text}`;
}

/**
 * ══════════════════════════════════════════════════════════════════════
 * **السعرُ بالعملتين — قطعةٌ واحدةٌ لكلّ موضعٍ يظهر فيه الدولارُ والليرةُ معاً**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (ملاحظاتُ المالك ٢٠٢٦-١١-١١: «$0.13 · 15 ل.س» تُقرأ رقماً واحداً.) **الدولارُ عريضٌ بلون العلامة، والليرةُ
 * أصغرُ باهتةٌ بعد «≈».** **و«$» لا ينقلب** (لفّةُ `dir="ltr"`)، **ولا «·» يتيم**: لا فاصلَ، والليرةُ وحدةٌ
 * لا تنكسر (`whitespace-nowrap`) تنزل كلُّها تحت الدولار إن ضاق السطر، أو دائماً بـ`stacked`.
 * **مرآةُ `DualPrice` في `mobile/ui`** — ومن غيّر شكلاً غيّره في الآخر. وبلا دولارٍ تُعرض الليرةُ وحدَها.
 */
export function DualPrice({
  usd,
  syp,
  stacked = false,
  muted = false,
  inherit = false,
  className = "",
}: {
  usd?: number | null;
  syp: number;
  /** **سطران دائماً** — لخانةٍ ضيّقة. */
  stacked?: boolean;
  /** **بلا لونٍ ولا ثِقَل** — لسطور الخيارات والمشروبات. */
  muted?: boolean;
  /** **يرث لونَ الأب** — داخل زرٍّ ملوّن. */
  inherit?: boolean;
  className?: string;
}) {
  const strong = inherit ? "font-bold" : muted ? "" : "font-bold text-accent-text";
  if (usd == null) return <Money value={syp} small={muted} className={`${strong} ${className}`} />;
  const sypTone = inherit ? "opacity-90" : "text-ink-muted";
  return (
    <span className={`inline-flex ${stacked ? "flex-col items-start" : "flex-wrap items-baseline gap-x-1"} ${className}`}>
      <span dir="ltr" className={`tabular-nums ${strong}`}>
        {usdLabel(usd)}
      </span>
      <span className={`whitespace-nowrap ${muted ? "" : "text-xs"} ${sypTone}`}>
        {"≈ "}
        <Money value={syp} small className={inherit ? "[&_span]:text-current!" : ""} />
      </span>
    </span>
  );
}

/**
 * **رقمُ هاتفٍ للعرض داخل نصٍّ عربيّ** (D4: «963900555001+») — معزولٌ يساراً (`LRI … PDI`) فلا تقفز «+» إلى
 * الطرف الآخر. **مرآةُ `phoneText` في `mobile/ui`**، والنصُّ المخزَّن لا يتغيّر والفارغُ يبقى فارغاً.
 */
export function phoneText(phone: string | null | undefined): string {
  const p = (phone ?? "").trim();
  return p ? `\u2066${p}\u2069` : "";
}

/**
 * **سهمُ «جديد ← قديم» مقلوبٌ في السطر العربيّ** (D12: «1x ← 2x») — الخادمُ يكتب الجديدَ أوّلاً فيقع على اليمين
 * والسهمُ يشير إلى القديم. **فيُبدَّل الطرفان للعرض** ليقرأ العينُ من القديم (يميناً) إلى الجديد (يساراً)،
 * وكلُّ رقمٍ معزول. **مرآةُ `BidiText.fixArrows` في `mobile/ui`**؛ والنصُّ المخزَّن لا يتغيّر.
 */
export function fixArrows(text: string): string {
  const cur = String.fromCharCode(0x644, 0x2e, 0x633).replace(".", String.raw`\.`);
  const tok = String.raw`[+\-−]?#?\d[\d.,:/]*(?:\s?${cur})?`;
  return text.replace(new RegExp(String.raw`(${tok})\s*←\s*(${tok})`, "g"), (_m, a: string, b: string) => `⁨${b}⁩ ← ⁨${a}⁩`);
}
