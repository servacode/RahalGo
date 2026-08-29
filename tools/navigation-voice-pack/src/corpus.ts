/**
 * ══════════════════════════════════════════════════════════════════════
 * **مدوّنةُ التعليمات — تُولَّد من قوالبَ وقواميس، ولا تُكتب بيد**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (مواصفةُ المالك ٢٠٢٦-٠٨-٢٣، البند ١٥: «لا تضع مئاتِ الجمل يدويّاً إذا
 *  كان بالإمكان توليدُها بأمانٍ من Templates + Lexicons».)
 *
 * **والناتجُ يحمل النصَّ الكاملَ لكلّ ملفّ** قبل أن يُرسَل — فلا
 * يُولَّد شيءٌ في الظلام.
 *
 * # والجملةُ تُولَّد كاملةً لا مقاطع
 *
 * (البند ٨.) **ولا تُقطَّع إلى «بَعْد» + «ثَلاثمِئَة مِتْر» + «اِنْعَطِف
 * يَمينًا»** ثمّ تُركَّب عند التشغيل: **الوصلُ يكسر النبرَ والفواصل**،
 * فتُسمع ثلاثَ جملٍ لا جملةً واحدة.
 *
 * # والنصُّ المرئيُّ غيرُ المنطوق
 *
 * (البند ٥.) `displayText` بلا تشكيلٍ للشاشة، و`ttsText` مشكولٌ
 * للمحرّك. **ولا يُعرض التشكيلُ لمستخدمٍ أبداً.**
 */

import { spokenDistance, spokenOrdinal, distanceSlug, ORDINALS } from './lexicon.js';
import { applyOverrides } from './overrides.js';

/** **نوعُ المناورة كما يسمّيه OSRM** — البند ١٤. */
export type ManeuverType =
  | 'turn'
  | 'continue'
  | 'merge'
  | 'on ramp'
  | 'off ramp'
  | 'fork'
  | 'end of road'
  | 'new name'
  | 'depart'
  | 'arrive'
  | 'roundabout'
  | 'rotary'
  | 'exit roundabout'
  | 'exit rotary'
  | null;

export type ManeuverModifier =
  | 'straight'
  | 'slight right'
  | 'right'
  | 'sharp right'
  | 'uturn'
  | 'sharp left'
  | 'left'
  | 'slight left'
  | null;

export type Category =
  | 'turn'
  | 'lane'
  | 'merge'
  | 'fork'
  | 'ramp'
  | 'end_of_road'
  | 'roundabout'
  | 'destination'
  | 'status';

export interface Instruction {
  readonly id: string;
  readonly category: Category;
  readonly displayText: string;
  readonly ttsText: string;
  readonly file: string;
  readonly distanceMeters?: number;
  readonly roundaboutExit?: number;
  readonly maneuverType: ManeuverType;
  readonly maneuverModifier: ManeuverModifier;
}

interface Stem {
  readonly id: string;
  readonly category: Category;
  readonly display: string;
  readonly tts: string;
  readonly file: string;
  readonly type: ManeuverType;
  readonly modifier: ManeuverModifier;
  /** **وبعضُها لا مسافةَ له** — «بدأت الملاحة» لا تُقال بعد ثلاثمئة متر. */
  readonly withDistance: boolean;
}

/**
 * **مسافاتُ الإعلان** — لا كلُّ ما في القاموس.
 *
 * **وستّةَ عشرَ مسافةً لكلّ تعليمةٍ تعني مئاتِ الملفّات بلا فائدة** —
 * **والسائقُ لا يُنبَّه على ثمانمئةٍ وتسعمئةٍ معاً.** فيُختار ما
 * يُستعمل فعلاً في الإعلان.
 */
export const ANNOUNCE_DISTANCES: readonly number[] = [
  50, 100, 150, 200, 250, 300, 400, 500, 700, 1000, 1500, 2000,
];

const STEMS: readonly Stem[] = [
  // ── الانعطافُ الأساسيّ — البند ٩ ────────────────────────────────────
  { id: 'turn_right', category: 'turn', display: 'انعطف يمينًا.', tts: 'اِنْعَطِف يَمينًا.', file: 'turn_right', type: 'turn', modifier: 'right', withDistance: true },
  { id: 'turn_left', category: 'turn', display: 'انعطف يسارًا.', tts: 'اِنْعَطِف يَسارًا.', file: 'turn_left', type: 'turn', modifier: 'left', withDistance: true },
  { id: 'slight_right', category: 'turn', display: 'انحرف قليلًا نحو اليمين.', tts: 'اِنْحَرِف قَليلًا نَحوَ اليَمين.', file: 'slight_right', type: 'turn', modifier: 'slight right', withDistance: true },
  { id: 'slight_left', category: 'turn', display: 'انحرف قليلًا نحو اليسار.', tts: 'اِنْحَرِف قَليلًا نَحوَ اليَسار.', file: 'slight_left', type: 'turn', modifier: 'slight left', withDistance: true },
  { id: 'sharp_right', category: 'turn', display: 'انعطف بحدّة نحو اليمين.', tts: 'اِنْعَطِف بِحِدَّة نَحوَ اليَمين.', file: 'sharp_right', type: 'turn', modifier: 'sharp right', withDistance: true },
  { id: 'sharp_left', category: 'turn', display: 'انعطف بحدّة نحو اليسار.', tts: 'اِنْعَطِف بِحِدَّة نَحوَ اليَسار.', file: 'sharp_left', type: 'turn', modifier: 'sharp left', withDistance: true },
  { id: 'continue_straight', category: 'turn', display: 'استمرّ مباشرةً.', tts: 'اِستَمِرّ مُباشَرَةً.', file: 'continue_straight', type: 'continue', modifier: 'straight', withDistance: true },
  { id: 'uturn', category: 'turn', display: 'انعطف للعودة في الاتّجاه المعاكس.', tts: 'اِنعَطِف لِلعَودَة في الاِتِّجاه المُعاكِس.', file: 'uturn', type: 'turn', modifier: 'uturn', withDistance: true },

  // ── المسار — البند ٩ ────────────────────────────────────────────────
  { id: 'keep_right', category: 'lane', display: 'التزم المسار الأيمن.', tts: 'اِلتَزِم المَسار الأَيمَن.', file: 'keep_right', type: 'fork', modifier: 'slight right', withDistance: true },
  { id: 'keep_left', category: 'lane', display: 'التزم المسار الأيسر.', tts: 'اِلتَزِم المَسار الأَيسَر.', file: 'keep_left', type: 'fork', modifier: 'slight left', withDistance: true },

  // ── الاندماجُ والتفرّع — البند ١٠ ───────────────────────────────────
  { id: 'merge_right', category: 'merge', display: 'اندمج في المسار نحو اليمين.', tts: 'اِندَمِج في المَسار نَحوَ اليَمين.', file: 'merge_right', type: 'merge', modifier: 'right', withDistance: true },
  { id: 'merge_left', category: 'merge', display: 'اندمج في المسار نحو اليسار.', tts: 'اِندَمِج في المَسار نَحوَ اليَسار.', file: 'merge_left', type: 'merge', modifier: 'left', withDistance: true },
  { id: 'fork_right', category: 'fork', display: 'عند تفرّع الطريق، التزم اليمين.', tts: 'عِند تَفَرُّع الطَّريق، اِلتَزِم اليَمين.', file: 'fork_right', type: 'fork', modifier: 'right', withDistance: true },
  { id: 'fork_left', category: 'fork', display: 'عند تفرّع الطريق، التزم اليسار.', tts: 'عِند تَفَرُّع الطَّريق، اِلتَزِم اليَسار.', file: 'fork_left', type: 'fork', modifier: 'left', withDistance: true },

  // ── نهايةُ الطريق والمخارج — البند ١٠ ───────────────────────────────
  { id: 'end_of_road_right', category: 'end_of_road', display: 'عند نهاية الطريق، انعطف يمينًا.', tts: 'عِند نِهايَة الطَّريق، اِنْعَطِف يَمينًا.', file: 'end_of_road_right', type: 'end of road', modifier: 'right', withDistance: true },
  { id: 'end_of_road_left', category: 'end_of_road', display: 'عند نهاية الطريق، انعطف يسارًا.', tts: 'عِند نِهايَة الطَّريق، اِنْعَطِف يَسارًا.', file: 'end_of_road_left', type: 'end of road', modifier: 'left', withDistance: true },
  { id: 'ramp_right', category: 'ramp', display: 'خذ المخرج نحو اليمين.', tts: 'خُذ المَخرَج نَحوَ اليَمين.', file: 'ramp_right', type: 'off ramp', modifier: 'right', withDistance: true },
  { id: 'ramp_left', category: 'ramp', display: 'خذ المخرج نحو اليسار.', tts: 'خُذ المَخرَج نَحوَ اليَسار.', file: 'ramp_left', type: 'off ramp', modifier: 'left', withDistance: true },

  // ── الوجهة — البند ١٢ ───────────────────────────────────────────────
  { id: 'destination_right', category: 'destination', display: 'وجهتك على اليمين.', tts: 'وِجهَتُك عَلَى اليَمين.', file: 'destination_right', type: 'arrive', modifier: 'right', withDistance: true },
  { id: 'destination_left', category: 'destination', display: 'وجهتك على اليسار.', tts: 'وِجهَتُك عَلَى اليَسار.', file: 'destination_left', type: 'arrive', modifier: 'left', withDistance: true },
  { id: 'approaching_destination', category: 'destination', display: 'أنت تقترب من وجهتك.', tts: 'أَنتَ تَقتَرِب مِن وِجهَتِك.', file: 'approaching_destination', type: 'arrive', modifier: null, withDistance: false },
  { id: 'arrived', category: 'destination', display: 'تمّ الوصول إلى الوجهة.', tts: 'تَمَّ الوُصول إِلى الوِجهَة.', file: 'arrived', type: 'arrive', modifier: null, withDistance: false },

  // ── حالُ الملاحة — البند ١٣ ─────────────────────────────────────────
  { id: 'recalculating_route', category: 'status', display: 'جاري إعادة حساب المسار.', tts: 'جارِي إِعادَة حِساب المَسار.', file: 'recalculating_route', type: null, modifier: null, withDistance: false },
  { id: 'route_updated', category: 'status', display: 'تمّ تحديث المسار.', tts: 'تَمَّ تَحديث المَسار.', file: 'route_updated', type: null, modifier: null, withDistance: false },
  { id: 'gps_weak', category: 'status', display: 'إشارة تحديد الموقع ضعيفة.', tts: 'إِشارَة تَحديد المَوقِع ضَعيفَة.', file: 'gps_weak', type: null, modifier: null, withDistance: false },
  // **و«انقطعت» لا «فُقدت»** — **الثانيةُ تحتاج ضمّةً لتُقرأ مبنيّةً
  // للمجهول**، وبلاها تُقرأ «فَقَدت» فيصير للإشارة فاعلٌ لا وجودَ له.
  { id: 'gps_lost', category: 'status', display: 'انقطعت إشارة تحديد الموقع.', tts: 'اِنقَطَعَت إِشارَة تَحديد المَوقِع.', file: 'gps_lost', type: null, modifier: null, withDistance: false },
  // **و«عادت» تقابل «انقطعت»** — والأولى كانت «تمّ استعادة» بتذكيرٍ خطأ.
  { id: 'gps_restored', category: 'status', display: 'عادت إشارة تحديد الموقع.', tts: 'عادَت إِشارَة تَحديد المَوقِع.', file: 'gps_restored', type: null, modifier: null, withDistance: false },
  { id: 'navigation_started', category: 'status', display: 'بدأت الملاحة.', tts: 'بَدَأَت المِلاحَة.', file: 'navigation_started', type: 'depart', modifier: null, withDistance: false },

  // ── ما ينطقه التطبيقُ وليس في مواصفة ٢٠٢٦-٠٨-٢٣ ────────────────────
  //
  // **قِيست `VoicePhrases` في تطبيق السائق (٢٠٢٦-٠٨-٢٤)** فبان أنّ
  // ثلاثَ عشرةَ جملةً يقولها المحرّكُ ولا مقطعَ لها. **ومقطعٌ ناقصٌ
  // يعني صمتاً عند مناورة** — وذاك أسوأُ من صوتٍ رديء.
  //
  // **والمعدِّلُ قد يغيب**: OSRM يعطي `fork` بلا يمينٍ ولا يسار،
  // **ومن اخترع له جهةً أضلَّ سائقاً.**
  { id: 'follow_route', category: 'status', display: 'تابع المسار.', tts: 'تابِع المَسار.', file: 'follow_route', type: null, modifier: null, withDistance: false },
  { id: 'depart', category: 'turn', display: 'ابدأ السير.', tts: 'اِبدَأ السَّير.', file: 'depart', type: 'depart', modifier: null, withDistance: false },
  { id: 'merge', category: 'merge', display: 'اندمج مع الطريق.', tts: 'اِندَمِج مَع الطَّريق.', file: 'merge', type: 'merge', modifier: null, withDistance: true },
  { id: 'ramp', category: 'ramp', display: 'اسلك المخرج.', tts: 'اِسلُك المَخرَج.', file: 'ramp', type: 'off ramp', modifier: null, withDistance: true },
  { id: 'fork', category: 'fork', display: 'انتبه، الطريق يتفرّع.', tts: 'اِنتَبِه، الطَّريق يَتَفَرَّع.', file: 'fork', type: 'fork', modifier: null, withDistance: true },
  { id: 'exit_roundabout', category: 'roundabout', display: 'اخرج من الدوّار.', tts: 'اُخرُج مِن الدَّوّار.', file: 'exit_roundabout', type: 'exit roundabout', modifier: null, withDistance: true },
  { id: 'roundabout_continue', category: 'roundabout', display: 'عند الدوّار، تابع الاتّجاه.', tts: 'عِند الدَّوّار، تابِع الاِتِّجاه.', file: 'roundabout_continue', type: 'roundabout', modifier: null, withDistance: true },

  // ── جملُ الأحداث ────────────────────────────────────────────────────
  //
  // **ولفظُ الاتّجاه المعاكس كما قرّره المالك ٢٠٢٦-٠٨-٢١**: وصفٌ لا
  // أمرٌ بمناورة — **«غيّر اتّجاه السير» تُقرأ أمراً بدورانٍ فوريّ وقد
  // يكون محرَّماً أو خطرا.**
  { id: 'wrong_way', category: 'status', display: 'أنت تسير بعكس اتّجاه المسار.', tts: 'أَنتَ تَسير بِعَكس اِتِّجاه المَسار.', file: 'wrong_way', type: null, modifier: null, withDistance: false },
  { id: 'reroute_failed', category: 'status', display: 'تعذّرت إعادة حساب المسار.', tts: 'تَعَذَّرَت إِعادَة حِساب المَسار.', file: 'reroute_failed', type: null, modifier: null, withDistance: false },
  // **وطرفا الرحلة لفظان لا لفظٌ واحد** — «وصلت» وحدَها لا تقول
  // للسائق أعند المتجر هو أم عند الزبون.
  { id: 'arrived_pickup', category: 'destination', display: 'وصلت إلى نقطة الاستلام.', tts: 'وَصَلْتَ إِلى نُقطَة الاِستِلام.', file: 'arrived_pickup', type: 'arrive', modifier: null, withDistance: false },
  { id: 'arrived_dropoff', category: 'destination', display: 'وصلت إلى وجهة التوصيل.', tts: 'وَصَلْتَ إِلى وِجهَة التَّوصيل.', file: 'arrived_dropoff', type: 'arrive', modifier: null, withDistance: false },
  // **ولا تدّعي وصولاً ولا قُرباً** — قرارُ المالك ٢٠٢٦-٠٨-٢١، البند ٩.
  { id: 'route_end', category: 'status', display: 'انتهى المسار المرسوم، تابع نحو الموقع.', tts: 'اِنتَهى المَسار المَرسوم، تابِع نَحوَ المَوقِع.', file: 'route_end', type: null, modifier: null, withDistance: false },
];

/**
 * **علاماتُ التشكيل الكامل** — وهي وحدَها ما يُرفض في النصّ المرئيّ.
 *
 * **والشدّةُ وتنوينُ الفتح ليسا منها**: «تمّ» و«يمينًا» هكذا تُكتبان في
 * كلّ نصٍّ عربيٍّ عاديّ، **ونزعُهما يجعل المكتوبَ أغربَ لا أوضح** —
 * «تم» و«يمينا». والبندُ ٥ أراد ألّا يظهر التشكيلُ الكاملُ على شاشة
 * السائق، لا أن تُكتب العربيّةُ ناقصةً.
 *
 * وهي: تنوينُ الضمّ والكسر · الفتحةُ والضمّةُ والكسرة · السكون ·
 * الألفُ الخنجريّة.
 */
export const VOCALIZATION = /[ٌ-ِْٰ]/;

/** **ويُنزع التشكيلُ كلُّه** — يُستعمل لاشتقاق المرئيّ من المنطوق. */
export function stripDiacritics(s: string): string {
  return s.replace(/[ً-ْٰـ]/g, '');
}

/** **وسابقةُ المسافة** — البند ٧: `بَعْد {distance}`. */
export function distancePrefix(meters: number): { display: string; tts: string } {
  return {
    display: `بعد ${stripDiacritics(spokenDistance(meters))}، `,
    tts: `بَعْد ${spokenDistance(meters)}، `,
  };
}

/** **وجملةُ الدوّار** — البند ١١. */
export function roundaboutSentence(exit: number): { display: string; tts: string } {
  return {
    display: `عند الدوّار، خذ المخرج ${stripDiacritics(spokenOrdinal(exit))}.`,
    tts: `عِند الدَّوّار، خُذ المَخرَج ${spokenOrdinal(exit)}.`,
  };
}

/**
 * **ويُبنى الجدولُ كلُّه حتميّاً** — **الترتيبُ نفسُه في كلّ تشغيل**،
 * فتُقارَن خطّتان وتُعرف الزيادةُ من النقص.
 */
/**
 * **صيغةُ التنفيذ** — «اِنْعَطِف يَمينًا **الآن**.»
 *
 * **وطورُ `NOW` جملةٌ أخرى لا الجملةَ نفسَها بلا مسافة**: من سمع
 * «انعطف يميناً» وهو عند المنعطف لا يعرف أهي له الآن أم بعد مئتَي متر.
 * **و«الآن» هي الفارق كلُّه.**
 */
function nowForm(sentence: string): string {
  return `${sentence.replace(/[.؟!]\s*$/, '')} الآن.`;
}

/**
 * **صيغةُ اللاحقة** — «**ثُمَّ** اِنْعَطِف يَسارًا **مُباشَرَةً**.»
 *
 * # ولماذا وُجدت
 *
 * (بلاغُ المالك ٢٠٢٦-٠٨-٢٤: «الصوتُ متأخّر… خصوصاً إذا كان هناك أكثرُ
 *  من انعطافٍ قريباتٍ على بعض».)
 *
 * **والمحرّكُ يدمج المتقاربتين في جملةٍ واحدة** — «انعطف يميناً، ثمّ
 * انعطف يساراً مباشرةً». **وربطُ المقاطع أسقط شطرَها الثاني**، لأنّ
 * عشرين مناورةً في عشرين تعني أربعمئةِ تركيبٍ لا تُسجَّل.
 *
 * **فتُسجَّل اللاحقةُ وحدَها** — عشرون جملةً لا أربعمئة، **وتُوصَل بعد
 * الأولى فتُسمع جملتين متتابعتين لا شطراً مبتوراً.**
 *
 * **وهذا وصلُ جملتين تامّتين لا تركيبُ جملةٍ من قِطَع** — والبندُ ٨ منع
 * الثاني لأنّه يكسر النبر، **ووقفةٌ بين جملتين طبيعيّةٌ يفعلها المتكلّم.**
 */
function thenForm(sentence: string): string {
  return `ثُمَّ ${sentence.replace(/^[اأإ]/, (c) => c).replace(/[.؟!]\s*$/, '')} مُباشَرَةً.`;
}

function thenDisplay(sentence: string): string {
  return `ثمّ ${sentence.replace(/[.؟!]\s*$/, '')} مباشرةً.`;
}

export function buildCorpus(): readonly Instruction[] {
  const out: Instruction[] = [];

  for (const s of STEMS) {
    out.push({
      id: s.id,
      category: s.category,
      displayText: s.display,
      ttsText: applyOverrides(s.tts),
      file: `${s.file}.mp3`,
      maneuverType: s.type,
      maneuverModifier: s.modifier,
    });
    if (!s.withDistance) continue;

    // **ولا «الآن» للوصول** — «وجهتك على اليمين الآن» عربيّةٌ عرجاء،
    // والمحرّكُ نفسُه لا يلحقها بها (`VoicePhrases.maneuver`).
    if (s.category !== 'destination') {
      out.push({
        id: `${s.id}_now`,
        category: s.category,
        displayText: nowForm(s.display),
        ttsText: applyOverrides(nowForm(s.tts)),
        file: `${s.file}_now.mp3`,
        maneuverType: s.type,
        maneuverModifier: s.modifier,
      });
    }

    // **واللاحقةُ للمناورة الثانية المتقاربة** — انظر [thenForm].
    out.push({
      id: `then_${s.id}`,
      category: s.category,
      displayText: thenDisplay(s.display),
      ttsText: applyOverrides(thenForm(s.tts)),
      file: `then_${s.file}.mp3`,
      maneuverType: s.type,
      maneuverModifier: s.modifier,
    });
    for (const m of ANNOUNCE_DISTANCES) {
      const p = distancePrefix(m);
      out.push({
        id: `${s.id}_in_${m}m`,
        category: s.category,
        displayText: p.display + s.display,
        ttsText: applyOverrides(p.tts + s.tts),
        file: `${s.file}_in_${distanceSlug(m)}.mp3`,
        distanceMeters: m,
        maneuverType: s.type,
        maneuverModifier: s.modifier,
      });
    }
  }

  for (const exit of ORDINALS.keys()) {
    const r = roundaboutSentence(exit);
    out.push({
      id: `roundabout_exit_${exit}`,
      category: 'roundabout',
      displayText: r.display,
      ttsText: applyOverrides(r.tts),
      file: `roundabout_exit_${exit}.mp3`,
      roundaboutExit: exit,
      maneuverType: 'roundabout',
      maneuverModifier: null,
    });
    out.push({
      id: `then_roundabout_exit_${exit}`,
      category: 'roundabout',
      displayText: thenDisplay(r.display),
      ttsText: applyOverrides(thenForm(r.tts)),
      file: `then_roundabout_exit_${exit}.mp3`,
      roundaboutExit: exit,
      maneuverType: 'roundabout',
      maneuverModifier: null,
    });
    out.push({
      id: `roundabout_exit_${exit}_now`,
      category: 'roundabout',
      displayText: nowForm(r.display),
      ttsText: applyOverrides(nowForm(r.tts)),
      file: `roundabout_exit_${exit}_now.mp3`,
      roundaboutExit: exit,
      maneuverType: 'roundabout',
      maneuverModifier: null,
    });
    for (const m of ANNOUNCE_DISTANCES) {
      const p = distancePrefix(m);
      out.push({
        id: `roundabout_exit_${exit}_in_${m}m`,
        category: 'roundabout',
        displayText: p.display + r.display,
        ttsText: applyOverrides(p.tts + r.tts),
        file: `roundabout_exit_${exit}_in_${distanceSlug(m)}.mp3`,
        distanceMeters: m,
        roundaboutExit: exit,
        maneuverType: 'roundabout',
        maneuverModifier: null,
      });
    }
  }

  validateCorpus(out);
  return out;
}

export class CorpusError extends Error {}

/** **ولا معرّفَ يتكرّر ولا اسمَ ملفّ** — البند ٢٩. */
export function validateCorpus(list: readonly Instruction[]): void {
  const ids = new Set<string>();
  const files = new Set<string>();
  for (const i of list) {
    if (ids.has(i.id)) throw new CorpusError(`معرّفٌ مكرّر: ${i.id}`);
    if (files.has(i.file)) throw new CorpusError(`اسمُ ملفٍّ مكرّر: ${i.file}`);
    ids.add(i.id);
    files.add(i.file);
    // **واسمُ الملفّ صالحٌ لـ`res/raw`** — البند ١٦: **حروفٌ صغيرةٌ
    // وشرطةٌ سفليّةٌ ولا شيءَ سواهما**، ولا يبدأ برقم.
    if (!/^[a-z][a-z0-9_]*\.mp3$/.test(i.file)) {
      throw new CorpusError(`اسمٌ غيرُ صالحٍ لأندرويد: ${i.file}`);
    }
    if (i.ttsText.trim() === '' || i.displayText.trim() === '') {
      throw new CorpusError(`نصٌّ فارغ: ${i.id}`);
    }
    // **ولا رقمَ رقميٌّ في المنطوق** — البند ٦: **يقرؤه المحرّكُ كما
    // يشاء**، وقد يقرؤه بالإنكليزيّة.
    if (/[0-9٠-٩]/.test(i.ttsText)) {
      throw new CorpusError(`رقمٌ في النصّ المنطوق: ${i.id} — ${i.ttsText}`);
    }
    // **والمرئيُّ بلا تشكيلٍ كامل** — البند ٥.
    if (VOCALIZATION.test(i.displayText)) {
      throw new CorpusError(`تشكيلٌ في النصّ المرئيّ: ${i.id}`);
    }
  }
}
