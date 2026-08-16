"use client";

/**
 * شكوى الزبون على طلبه.
 *
 * # لماذا وُجدت
 *
 * لم يكن للزبون بابُ شكوى: التذاكرُ تُفتح من لوحة الإدارة، فيتّصل بالمكتب
 * ويفتح موظّفٌ تذكرةً عنه. **فمن لم يجد من يردّ عليه لم يجد باباً أصلاً** —
 * وشكواه تضيع، **ونحن لا نعلم أنّها وقعت.**
 *
 * # ولماذا قائمةٌ لا صندوقُ نصّ
 *
 * **الأسبابُ من الخادم لا تُكتب هنا**: قائمةٌ في مكانين تفترق حين يُضاف سببٌ
 * في أحدهما — فيرسل التطبيقُ رمزاً لا يعرفه الخادم. **والنصُّ الحرُّ يبقى
 * بجانب السبب: القائمةُ تُصنّف والنصُّ يشرح.**
 */

import { useEffect, useState } from "react";
import { getMessages, defaultLocale, errorText } from "@rahalgo/i18n";
import { Button, Modal, Radio, Textarea } from "@rahalgo/ui";
import { api, ApiError } from "@/lib/api";

const m = getMessages(defaultLocale);
const C = m.site.complaint;
const NOTE_MAX = 500;

type Reason = { code: string; delivered_only: boolean };

export default function ComplaintModal({
  orderId,
  orderNumber,
  onClose,
  onOpened,
}: {
  orderId: string;
  orderNumber: number;
  onClose: () => void;
  onOpened: (ticketNumber: number) => void;
}) {
  const [reasons, setReasons] = useState<Reason[]>([]);
  const [reason, setReason] = useState("");
  const [note, setNote] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    api<{ reasons: Reason[] }>(`/api/v1/my/orders/${orderId}/complaint-reasons`)
      .then((r) => setReasons(r.reasons ?? []))
      .catch((err) => setError(errorText(err)));
  }, [orderId]);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    if (!reason) return setError(C.pickReason);
    // **«أخرى» بلا شرحٍ ليست شكوى** — لا يُعرف ما وقع فلا يُبدأ بشيء.
    if (reason === "other" && !note.trim()) return setError(C.needNote);
    setBusy(true);
    setError("");
    try {
      const t = await api<{ number: number }>(`/api/v1/my/orders/${orderId}/complaint`, {
        method: "POST",
        body: JSON.stringify({ reason, note: note.trim() }),
      });
      onOpened(t.number);
    } catch (err) {
      const key = err instanceof ApiError ? (err.body.message_key.split(".").pop() ?? "") : "";
      setError((m.errors as Record<string, string>)[key] ?? m.errors.internal);
      setBusy(false);
    }
  }

  return (
    /* ══════════════════════════════════════════════════════════════════
       **والنافذةُ من العُدّة لا مبنيّةً بيدها**
       ══════════════════════════════════════════════════════════════════

       (شكوى المالك ٢٠٢٦-٠٨-٠٩ بلقطة: «شوف الشكوى كيف تظهر وقت نضغط على
        شكوى» — والنافذةُ مقصوصةٌ من أعلى وأسفل.)

       **كانت طبقةً وصندوقاً مكتوبين هنا**: `fixed inset-0` و`max-w-sm`
       **بلا سقفِ ارتفاعٍ ولا تمرير.** وفيها ثمانيةُ أسبابٍ وصندوقُ نصٍّ
       وزرّان — **أطولُ من شاشة هاتف**، فيخرج رأسُها وذيلُها عن الشاشة
       **ولا سبيلَ إلى تمريرها.**

       **و`Modal` المركزيّ يحمل ما نقص**: سقفُ `90vh` وتمريرٌ عند الفيض،
       وزرُّ إغلاقٍ يُرى، و`Esc`، والإغلاقُ بالنقر خارجَها. **ومن بنى
       نافذةً بيده أعاد بناءَ أربعةٍ منها ونسي واحدة.**

       **والعنوانُ يحمل رقمَ الطلب** — كان سطراً تحته، **وسطرٌ يُقصّ مع
       الرأس حين تفيض.** */
    <Modal open onClose={onClose} title={`${C.title} · #${orderNumber}`}>
      <form onSubmit={submit} className="space-y-4">
          {/* **الاختيارُ من العُدّة لا مرتجَلاً.**

              كان `<input type="radio" className="accent-primary">` — **والرسمُ
              متروكٌ للمتصفّح**، فدائرةُ ويندوز تخالف دائرةَ أندرويد.

              **و`rounded-input` لا وجودَ له في الثيم**: صنفٌ يُكتب ولا يفعل
              شيئاً، **فالبطاقاتُ هنا مربّعةُ الأركان** وكلُّ بطاقةٍ في المنصة
              مستديرة — ولا يصرخ به بناءٌ ولا تحذير. */}
          <div className="space-y-1.5">
            {reasons.map((r) => (
              <Radio
                key={r.code}
                id={`cr-${r.code}`}
                name="reason"
                checked={reason === r.code}
                onChange={() => setReason(r.code)}
                label={C.reasons[r.code as keyof typeof C.reasons] ?? r.code}
                className={`rounded-control border p-2.5 ${
                  reason === r.code ? "border-primary bg-primary-tint" : "border-line"
                }`}
              />
            ))}
          </div>
          <Textarea
            id="cn"
            label={C.note}
            maxLength={NOTE_MAX}
            value={note}
            onChange={(e) => setNote(e.target.value)}
            placeholder={C.noteHint}
          />
          {error && <p className="text-sm text-danger">{error}</p>}
          <div className="flex gap-2">
            <Button type="submit" disabled={busy}>
              {C.submit}
            </Button>
            <Button type="button" variant="ghost" onClick={onClose}>
              {m.common.cancel}
            </Button>
          </div>
      </form>
    </Modal>
  );
}
