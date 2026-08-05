/**
 * **هويّةُ المنصة — تُقرأ في الخادم لا في المتصفّح.**
 *
 * # لماذا هنا لا في المكوّن
 *
 * كانت تُجلب بـ`useEffect` بعد أن تُرسَم الصفحة، **فيخرج من الخادم اسمُ
 * العلامة ثمّ يُبدَّل بالاسم المسجَّل بعد جزءٍ من الثانية** — و«تواصل معنا»
 * لا يظهر في المصدر أصلاً.
 *
 * **ووثيقةٌ قانونيةٌ يُقرأ نصُّها من فهارس البحث**: من بحث عن اسم الشركة لم
 * يجد صفحةَ شروطها، **ومن فتحها بشبكةٍ بطيئةٍ قرأ اسمَ العلامة وظنّه المتعاقد.**
 *
 * # ولا تُخزَّن أبداً
 *
 * `no-store` — **الاسمُ والرقمُ يُبدَّلان من لوحة الادمن**، ولو خُزّنت الصفحةُ
 * لَبقي الرقمُ القديمُ معروضاً بعد تبديله. **ومن اتّصل بالقديم لم يجد أحداً.**
 *
 * # وتعثّرُها لا يُسقط الصفحة
 *
 * **الشروطُ تُقرأ وإن غاب رقمُ الهاتف** — فيُعاد `null` ويُعرض اسمُ العلامة،
 * ويُخفى قسمُ التواصل. **وصفحةٌ قانونيةٌ لا تُفتح أسوأُ من صفحةٍ بلا رقم.**
 */

const API = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

export interface Contact {
  legal_name: string;
  support_phone: string;
  address: string;
}

export async function getContact(): Promise<Contact | null> {
  try {
    const r = await fetch(`${API}/api/v1/public/contact`, { cache: "no-store" });
    if (!r.ok) return null;
    const body = (await r.json()) as { data?: Contact };
    return body.data ?? null;
  } catch {
    return null;
  }
}
