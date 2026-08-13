/**
 * **هويّةُ المنصة ونصوصُ الصفحات — تُقرأ في الخادم لا في المتصفّح.**
 *
 * # لماذا هنا لا في المكوّن
 *
 * كانت تُجلب بـ`useEffect` بعد أن تُرسَم الصفحة، **فيخرج من الخادم اسمُ
 * العلامة ثمّ يُبدَّل بعد جزءٍ من الثانية.**
 *
 * **ووثيقةٌ قانونيةٌ يُقرأ نصُّها من فهارس البحث**: من بحث عن اسم الشركة لم
 * يجد صفحةَ شروطها، **ومن فتحها بشبكةٍ بطيئةٍ قرأ اسمَ العلامة وظنّه المتعاقد.**
 *
 * # ولا تُخزَّن أبداً
 *
 * `no-store` — **الاسمُ والرقمُ والنصوصُ تُبدَّل من لوحة الادمن**، ولو خُزّنت
 * لَبقي القديمُ معروضاً بعد تبديله.
 *
 * # وتعثّرُها لا يُسقط الصفحة
 *
 * **الشروطُ تُقرأ وإن غاب رقمُ الهاتف** — فيُعاد `null` ويُعرض نصُّ المعجم.
 * **وصفحةٌ قانونيةٌ لا تُفتح أسوأُ من صفحةٍ بلا رقم.**
 */

const API = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

export interface Contact {
  legal_name: string;
  support_phone: string;
  address: string;
  /** **نصوصُ الصفحات المحرَّرةُ من اللوحة** — وفارغُها يعني «خذ من المعجم». */
  help_text?: string;
  terms_text?: string;
  privacy_text?: string;
  /** **من نحن** — صفحةٌ في الموقع والتطبيق معاً (٢٠٢٦-٠٨-١٣). */
  about_text?: string;
  /** **تعليماتُ السائق** — غيرُ تعليمات الزبون. */
  driver_help_text?: string;
}

export async function getContact(): Promise<Contact | null> {
  try {
    const r = await fetch(`${API}/api/v1/public/contact`, { cache: "no-store" });
    if (!r.ok) return null;
    const body = (await r.json()) as { data?: Contact };
    return body.data ?? null;
  } catch {
    // @empty-ok — انظر أعلاه: تعثّرُها لا يُسقط الصفحة.
    return null;
  }
}

/**
 * **يقرأ نصّاً محرَّراً إلى كروت.**
 *
 * (طلبُ المالك ٢٠٢٦-٠٨-٠٩: «جهّز الصفحات لتكون ديناميكيّةً أتحكّم بها من لوحة
 *  التحكّم، أعدّل النصوص الموجودة إذا أردتُ ذلك».)
 *
 * **والاتّفاقُ أبسطُ ما يمكن**: سطرٌ فارغٌ يفصل كرتاً عن كرت، **وأوّلُ سطرٍ
 * عنوانٌ** وما بعده فقرات.
 *
 * **ولا Markdown ولا محرّرٌ غنيّ** — من يكتب سياسةَ خصوصيّةٍ يكتب فقرات،
 * **ومحرّرٌ بعشرة أزرارٍ يُخطئ فيه** ويُخرج ترميزاً لا يُقرأ.
 *
 * **وكرتٌ بسطرٍ واحدٍ لا عنوانَ له** — فقرةٌ وحدَها، **ولا يُجعل أوّلُ سطرٍ
 * عنواناً حين لا شيءَ بعده** فيبقى الكرتُ عنواناً بلا متن.
 */
export function parseBlocks(text: string): { h?: string; p: string[] }[] {
  const chunks = text.split(BLANK_LINE);
  const out: { h?: string; p: string[] }[] = [];
  for (const chunk of chunks) {
    const lines = chunk
      .split(NEW_LINE)
      .map((x) => x.trim())
      .filter(Boolean);
    if (lines.length === 0) continue;
    if (lines.length === 1) out.push({ p: lines });
    else out.push({ h: lines[0], p: lines.slice(1) });
  }
  return out;
}

/** **سطرٌ فارغٌ يفصل الكروت** — وقد يحمل مسافاتٍ فيُتسامح فيها. */
const BLANK_LINE = new RegExp("\\n\\s*\\n");
const NEW_LINE = new RegExp("\\n");
