import { getMessages, defaultLocale, fmtNum } from "@rahalgo/i18n";

const m = getMessages(defaultLocale);

/**
 * **الوقتُ المتوقَّع — حسابٌ واحدٌ لا حسابان.**
 *
 * يُعرض في **بطاقة الطلب** وفي **صفحة التتبّع**. وكان مكتوباً في الصفحة وحدَها،
 * **ولو نُسخ للبطاقة لَافترقا يوماً**: يُصلَح أحدُهما ويبقى الآخر، **فتقول
 * البطاقةُ عشرين دقيقةً وتقول الصفحةُ خمساً** — والزبونُ يصدّق أيَّهما رأى
 * أوّلاً.
 *
 * **وهي عائلةُ الخلل التي تكرّرت في هذه الجولة ثلاث مرّات**: قائمةُ أنواع
 * الوسائط في Go وفي القاعدة · وسقفُ النقد في العرض وفي القبول · وشرطُ الساعات
 * في العرض وفي الإنشاء. **قاعدةٌ مكتوبةٌ مرّتين تفترق بلا صوت.**
 */
export interface EtaSource {
  accepted_at?: string | null;
  ready_at?: string | null;
  prep_minutes?: number | null;
  delivery_estimate_min?: number;
}

/**
 * **بديلٌ حين لا يقول الإعدادُ شيئاً.**
 *
 * والرقمُ هنا **لا يُكتب في نصٍّ يقرؤه الزبون** — يُحسب منه وقتٌ ويُعرض.
 * **ورقمٌ في نصٍّ يخالف رقماً في إعدادٍ أخطرُ من غياب الرقم**: غيابُه يُسأل
 * عنه، **وخلافُه يُصدَّق.**
 */
const FALLBACK_MIN = 15;

/** أللطلبِ وقتٌ متوقَّعٌ يُعرض؟ — **لا يُعرض لمن لم يُقبل بعد ولا لمن انتهى.** */
export function hasEta(o: EtaSource, status: string, closed: boolean): boolean {
  return !closed && status !== "delivered" && !!o.accepted_at && !!o.prep_minutes;
}

/** الوقتُ المتبقّي نصّاً — «خلال ١٢ دقيقة». */
export function etaText(o: EtaSource): string {
  const accepted = new Date(o.accepted_at!).getTime();
  const prepDone = o.ready_at
    ? new Date(o.ready_at).getTime()
    : accepted + (o.prep_minutes ?? 0) * 60_000;
  const est = o.delivery_estimate_min || FALLBACK_MIN;
  const left = Math.round((prepDone + est * 60_000 - Date.now()) / 60_000);
  // **ولا يُعرض صفرٌ ولا سالب**: طلبٌ تأخّر يقول «دقيقة» لا «‎−٣ دقائق» —
  // **والرقمُ السالب يُقرأ عطباً في المنصة لا تأخّراً في الطريق.**
  return m.site.orders.etaValue.replace("{n}", fmtNum(Math.max(1, left)));
}
