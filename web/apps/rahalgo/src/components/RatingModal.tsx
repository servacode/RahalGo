"use client";

/**
 * نافذة تقييم طلب مُسلَّم: نجوم الخدمة (إلزامي) + السائق (إن وُجد) + تعليق.
 *
 * **والنجمةُ للمنصة لا للمتجر.** الزبونُ لا يرى اسمَ متجرٍ ولا يختاره: يطلب
 * من «رحّال غو» ونحن نختار من أين نشتري. **فنجمةٌ تُنسب إلى متجرٍ لم يعرفه
 * نجمةٌ بلا معنى** — وهو يحكم على طعامٍ ووقتٍ ومعاملة، **وثلاثتُها من
 * عندنا.** وأسوأُ من انعدام المعنى أثرُه: **متجرٌ يُحاسَب على تأخيرٍ سببُه
 * سائقُنا.**
 */

import { useState } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import {
  Alert, Button, Modal, Stars, Textarea } from "@rahalgo/ui";
import { api, ApiError } from "@/lib/api";

const m = getMessages(defaultLocale);
const R = m.site.rating;
const COMMENT_MAX = 200;

export default function RatingModal({
  order,
  onClose,
  onRated,
}: {
  /** **ولا اسمَ متجرٍ هنا** — الأصنافُ تُعرّف الطلبَ، والمصدرُ محجوب. */
  order: { order_id: string; number: number; items_preview: string; has_driver: boolean };
  onClose: () => void;
  onRated: () => void;
}) {
  const [platformStars, setPlatformStars] = useState(0);
  const [driverStars, setDriverStars] = useState(0);
  const [comment, setComment] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    if (platformStars < 1) return setError(R.pickStars);
    setBusy(true);
    setError("");
    try {
      await api(`/api/v1/orders/${order.order_id}/rating`, {
        method: "POST",
        body: JSON.stringify({
          platform_stars: platformStars,
          driver_stars: order.has_driver && driverStars > 0 ? driverStars : null,
          comment: comment.trim(),
        }),
      });
      onRated();
    } catch (err) {
      const key = err instanceof ApiError ? err.body.message_key.split(".").pop() ?? "" : "";
      setError((m.errors as Record<string, string>)[key] ?? m.errors.internal);
      setBusy(false);
    }
  }

  return (
    /* **ومن العُدّة لا مبنيّةً بيدها** — كنافذة الشكوى: طبقةٌ وصندوقٌ
       مكتوبان هنا **بلا سقفِ ارتفاعٍ ولا تمرير**، فتُقصّ على شاشةٍ قصيرة.
       (شكوى المالك ٢٠٢٦-٠٨-٠٩ على أختِها، **والعلّةُ نسخةٌ منها**.) */
    <Modal open onClose={onClose} title={`${R.title} · #${order.number}`}>
      <p className="mb-4 text-sm text-ink-muted">{order.items_preview}</p>
        <form onSubmit={submit} className="space-y-4">
          <div>
            <p className="mb-1.5 text-sm font-medium">{R.platform}</p>
            <Stars value={platformStars} onChange={setPlatformStars} size="lg" />
          </div>
          {order.has_driver && (
            <div>
              <p className="mb-1.5 text-sm font-medium">{R.driver}</p>
              <Stars value={driverStars} onChange={setDriverStars} size="lg" />
            </div>
          )}
          <div>
            {/* **والعدّادُ من العُدّة** — كان مكتوباً هنا وفي نافذةِ الشكوى
                بصيغتين، **ورقمٌ لاتينيٌّ في واجهةٍ عربية** لأنّه لم يمرّ
                بـ`fmtNum`. */}
            <Textarea
              id="rc"
              label={R.comment}
              value={comment}
              onChange={(e) => setComment(e.target.value.slice(0, COMMENT_MAX))}
              rows={2}
              maxLength={COMMENT_MAX}
              placeholder={R.commentHint}
            />
          </div>
          {error && (
            <Alert>{error}</Alert>
          )}
          <div className="flex justify-end gap-2">
            <Button type="button" variant="secondary" onClick={onClose}>
              {m.common.cancel}
            </Button>
            <Button type="submit" disabled={busy}>
              {busy ? m.common.loading : R.submit}
            </Button>
          </div>
      </form>
    </Modal>
  );
}
