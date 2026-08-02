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
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { Button } from "@rahalgo/ui";
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
      .catch(() => setError(m.errors.internal));
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
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-ink/40 p-4"
      onClick={onClose}
    >
      <div
        className="w-full max-w-sm rounded-card border border-line bg-surface p-5 shadow-xl"
        onClick={(e) => e.stopPropagation()}
      >
        <h2 className="mb-1 text-lg font-bold">{C.title}</h2>
        <p className="mb-4 text-sm text-ink-muted">#{orderNumber}</p>
        <form onSubmit={submit} className="space-y-4">
          <div className="space-y-1.5">
            {reasons.map((r) => (
              <label
                key={r.code}
                className={`flex cursor-pointer items-center gap-2 rounded-input border p-2.5 text-sm ${
                  reason === r.code ? "border-primary bg-primary-light" : "border-line"
                }`}
              >
                <input
                  type="radio"
                  name="reason"
                  className="accent-primary"
                  checked={reason === r.code}
                  onChange={() => setReason(r.code)}
                />
                {C.reasons[r.code as keyof typeof C.reasons] ?? r.code}
              </label>
            ))}
          </div>
          <div>
            <label htmlFor="cn" className="mb-1.5 block text-sm font-medium">
              {C.note}
            </label>
            <textarea
              id="cn"
              rows={3}
              maxLength={NOTE_MAX}
              value={note}
              onChange={(e) => setNote(e.target.value)}
              placeholder={C.noteHint}
              className="w-full rounded-input border border-line bg-surface p-2 text-sm"
            />
          </div>
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
      </div>
    </div>
  );
}
