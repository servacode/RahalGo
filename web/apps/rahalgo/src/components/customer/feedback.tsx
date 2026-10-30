"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **التقييمُ والشكوى — كما في `OrdersScreen.kt`**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **نجومٌ للخدمة ونجومٌ للسائق** (`orders/ratings.go`) — **ولا تعليق**
 * (قرارُ المالك ٢٠٢٦-٠٨-٣١: «تقييمٌ بدون أيّ تعليقٍ بأيّ تطبيق»).
 *
 * **ويُسأل تلقائيّاً مرّةً واحدةً لكلّ طلب** — أحدثُ طلبٍ سُلّم ولم يُقيَّم
 * ولم يُسأل عنه، **ومرّةً في الجلسة** — ويُحفظ في المتصفّح (محميّاً) فلا
 * يُسأل ثانيةً ولو أُعيد فتحُ الصفحة. **و`already_rated` يُقرأ «قُيّم»**:
 * تُغلق النافذةُ ولا يُقال خطأ.
 *
 * **والشكوى سببٌ من قائمةٍ لا نصٌّ حرّ** (قرارُ المالك ٢٠٢٦-٠٨-١٥) — **والقائمةُ
 * من المحرّك تختلف بحال الطلب**، والتفصيلُ اختياريٌّ إلّا مع «أخرى».
 */

import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale, errorText, fmtNum, fmtRef } from "@rahalgo/i18n";
import { Alert, Button, LoadingState, Modal, Radio, Textarea, IconStar } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { readJSON, writeJSON } from "@/lib/customer/storage";
import { fill } from "@/lib/customer/texts";
import type { ComplaintReason, MyOrder, MyOrdersPage, RatingRow } from "@/lib/customer/types";
import { useShopToast } from "./toast";
import { markInviteDelivered } from "./InviteNudge";

const m = getMessages(defaultLocale);
const R = m.shop.rate;
const Q = m.shop.complaint;
const REASONS = m.site.complaint.reasons as Record<string, string>;
const PROMPTED_KEY = "rahalgo_rate_prompted";
const MAX_PROMPTED = 100;
const MAX_NOTE = 1000;

/** **أيُسأل عن نجوم السائق؟** — حيث أُسند سائق. */
export function hasDriver(o: MyOrder): boolean {
  return o.driver_assigned === true || !!o.driver_name;
}

/** **ما قيّمه من قبل** — `/my/ratings`، **وسقوطُه لا يُسقط شيئاً.** */
export function useRated(): { rated: Set<string>; mark: (id: string) => void; ready: boolean } {
  const { user } = useAuth();
  const [rated, setRated] = useState<Set<string>>(new Set());
  const [ready, setReady] = useState(false);
  useEffect(() => {
    if (!user) return;
    api<{ ratings: RatingRow[] }>("/api/v1/my/ratings?page=1&per_page=50")
      .then((r) => setRated(new Set((r.ratings ?? []).filter((x) => x.rated).map((x) => x.order_id))))
      .catch(() => undefined)
      .finally(() => setReady(true));
  }, [user]);
  const mark = useCallback((id: string) => setRated((s) => new Set(s).add(id)), []);
  return { rated, mark, ready };
}

function Stars({ value, onPick, label }: { value: number; onPick: (n: number) => void; label: string }) {
  return (
    <div>
      <p className="mb-1 text-sm">{label}</p>
      <div className="flex gap-1" role="radiogroup" aria-label={label}>
        {[1, 2, 3, 4, 5].map((n) => (
          <button
            key={n}
            type="button"
            role="radio"
            aria-checked={value === n}
            aria-label={fill(R.star, { n: fmtNum(n) })}
            onClick={() => onPick(n)}
            className="taparea flex h-10 w-10 items-center justify-center rounded-control hover:bg-row-hover"
          >
            <IconStar size={26} className={n <= value ? "fill-current text-accent" : "text-ink-dim"} />
          </button>
        ))}
      </div>
      {/* **وكلمةٌ تحت النجوم تقول ما اختار** — وقبل الاختيار: «اضغط على النجوم لتقيّم». */}
      <p className={`mt-1 text-xs ${value > 0 ? "font-bold text-accent" : "text-ink-muted"}`}>
        {value > 0 ? (R.words as Record<string, string>)[String(value)] : R.pick}
      </p>
    </div>
  );
}

/** **نافذةُ النجوم** — وتُغلق بالرجوع كما في التطبيق. */
export function RateDialog({
  order,
  onClose,
  onRated,
}: {
  order: MyOrder;
  onClose: () => void;
  onRated: (id: string) => void;
}) {
  const toast = useShopToast();
  const driver = hasDriver(order);
  // **والنجومُ تبدأ فارغة** (مرآةُ `RateDialog` ٢٠٢٦-١٠-٢٧) — كانت خمساً، فمن ضغط «إرسال» دغري
  // أرسل خمسَ نجومٍ وهو لم يقيّم. **فالزرُّ معطّلٌ حتّى يختار.**
  const [stars, setStars] = useState(0);
  const [driverStars, setDriverStars] = useState(0);
  const ready = stars > 0 && (!driver || driverStars > 0);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const send = async () => {
    setBusy(true);
    setError("");
    try {
      await api(`/api/v1/orders/${order.id}/rating`, {
        method: "POST",
        body: JSON.stringify({ platform_stars: stars, driver_stars: driver ? driverStars : null }),
      });
      toast(R.done, "success");
      onRated(order.id);
      onClose();
    } catch (err) {
      const code = (err as { body?: { code?: string } })?.body?.code;
      if (code === "already_rated") {
        onRated(order.id);
        onClose();
      } else {
        setError(errorText(err));
      }
    } finally {
      setBusy(false);
    }
  };

  return (
    <Modal open onClose={onClose} title={R.title}>
      <div className="space-y-4">
        <p className="text-xs text-ink-muted tabular-nums" dir="ltr">
          #{fmtRef(order.number)}
        </p>
        <Stars value={stars} onPick={setStars} label={R.service} />
        {driver && <Stars value={driverStars} onPick={setDriverStars} label={R.driver} />}
        {error && <Alert>{error}</Alert>}
        <div className="flex gap-2">
          <Button className="flex-1" onClick={send} disabled={busy || !ready}>
            {busy ? m.common.loading : R.send}
          </Button>
          <Button variant="secondary" onClick={onClose} disabled={busy}>
            {R.back}
          </Button>
        </div>
      </div>
    </Modal>
  );
}

/**
 * **السؤالُ التلقائيّ** — مرّةً في الجلسة، ومرّةً لكلّ طلب.
 *
 * **ويُثبَّت الطلبُ فورَ اختياره** — فلا يُسأل عنه ثانيةً ولو أُغلقت الصفحةُ
 * قبل الجواب (مرآةُ `autoPrompted` في `OrdersViewModel`).
 */
let askedThisSession = false;

export function RatingWatcher() {
  const { rated, mark, ready } = useRated();
  const [ask, setAsk] = useState<MyOrder | null>(null);

  useEffect(() => {
    if (!ready || askedThisSession) return;
    let alive = true;
    api<MyOrdersPage>("/api/v1/my/orders?open_only=false&page=1&per_page=30")
      .then((p) => {
        if (!alive || askedThisSession) return;
        /* **وصله طلبٌ** — فتعود بطاقةُ «عجبتك الخدمة؟ ادعُ صديقاً» مرّةً (٢٠٢٦-١٠-٣٠). */
        if ((p.orders ?? []).some((o) => o.status === "delivered")) markInviteDelivered();
        const prompted = readJSON<string[]>(PROMPTED_KEY, []);
        const seen = Array.isArray(prompted) ? prompted : [];
        const cand = (p.orders ?? []).find((o) => o.status === "delivered" && !rated.has(o.id) && !seen.includes(o.id));
        if (!cand) return;
        askedThisSession = true;
        writeJSON(PROMPTED_KEY, [...seen.filter((x) => x !== cand.id), cand.id].slice(-MAX_PROMPTED));
        setAsk(cand);
      })
      .catch(() => undefined);
    return () => {
      alive = false;
    };
  }, [ready, rated]);

  if (!ask) return null;
  return <RateDialog order={ask} onClose={() => setAsk(null)} onRated={mark} />;
}

/** **زرُّ «قيّم الطلب»** — لطلبٍ سُلّم ولم يُقيَّم. */
export function RateButton({ order }: { order: MyOrder }) {
  const { rated, mark, ready } = useRated();
  const [open, setOpen] = useState(false);
  if (order.status !== "delivered" || !ready) return null;
  if (rated.has(order.id)) return <p className="text-center text-xs text-ink-muted">{R.rated}</p>;
  return (
    <>
      <Button className="flex w-full items-center justify-center gap-2" onClick={() => setOpen(true)}>
        <IconStar size={16} />
        {R.button}
      </Button>
      {open && <RateDialog order={order} onClose={() => setOpen(false)} onRated={mark} />}
    </>
  );
}

/**
 * **الشكوى على الطلب** — وإن كانت مفتوحةً قيل رقمُها ولا يُفتح بابٌ ثانٍ.
 */
export function ComplaintBox({ order }: { order: MyOrder }) {
  const toast = useShopToast();
  const [ticket, setTicket] = useState<{ number: number } | null | undefined>(undefined);
  const [open, setOpen] = useState(false);
  const [reasons, setReasons] = useState<ComplaintReason[] | null>(null);
  const [reasonsFailed, setReasonsFailed] = useState(false);
  const [pick, setPick] = useState("");
  const [note, setNote] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    api<{ ticket: { number: number } | null }>(`/api/v1/my/orders/${order.id}/complaint`)
      .then((r) => setTicket(r.ticket ?? null))
      .catch(() => setTicket(null));
  }, [order.id]);

  /** **الأسبابُ — وإن سقط النداءُ قيل ذلك مع «إعادة»** (مرآةُ `ComplainDialog`) لا تعليقٌ على «عم نجيب». */
  const loadReasons = () => {
    setReasons(null);
    setReasonsFailed(false);
    api<{ reasons: ComplaintReason[] }>(`/api/v1/my/orders/${order.id}/complaint-reasons`)
      .then((r) => setReasons(r.reasons ?? []))
      .catch(() => {
        setReasons([]);
        setReasonsFailed(true);
      });
  };

  const openBox = () => {
    setOpen(true);
    setPick("");
    setNote("");
    setError("");
    loadReasons();
  };

  const needNote = pick === "other" && !note.trim();

  const send = async () => {
    setBusy(true);
    setError("");
    try {
      const t = await api<{ number?: number }>(`/api/v1/my/orders/${order.id}/complaint`, {
        method: "POST",
        body: JSON.stringify({ reason: pick, note: note.trim() }),
      });
      const n = typeof t?.number === "number" ? t.number : 0;
      setTicket({ number: n });
      setOpen(false);
      toast(fill(Q.sent, { n: fmtRef(n) }), "success");
    } catch (err) {
      setError(errorText(err));
    } finally {
      setBusy(false);
    }
  };

  if (ticket === undefined) return null;
  if (ticket) {
    return <Alert tone="info">{fill(Q.open, { n: fmtRef(ticket.number) })}</Alert>;
  }

  return (
    <>
      <Button variant="secondary" className="w-full" onClick={openBox}>
        {Q.button}
      </Button>
      <Modal open={open} onClose={() => setOpen(false)} title={Q.title}>
        <div className="space-y-3">
          {reasons === null ? (
            <LoadingState variant="inline" label={Q.loading} />
          ) : reasons.length === 0 && reasonsFailed ? (
            <div className="space-y-2">
              <p className="text-sm text-danger">{Q.failed}</p>
              <Button variant="secondary" onClick={loadReasons}>
                {m.common.retry}
              </Button>
            </div>
          ) : (
            <fieldset className="flex flex-col gap-2">
              {reasons.map((r) => (
                <Radio
                  key={r.code}
                  id={`cr-${r.code}`}
                  name="complaint-reason"
                  checked={pick === r.code}
                  onChange={() => setPick(r.code)}
                  label={REASONS[r.code] ?? r.code}
                />
              ))}
            </fieldset>
          )}
          {pick && (
            <Textarea
              id="complaint-note"
              label={pick === "other" ? Q.what : Q.note}
              value={note}
              onChange={(e) => setNote(e.target.value.slice(0, MAX_NOTE))}
              rows={3}
              maxLength={MAX_NOTE}
            />
          )}
          {error && <Alert>{error}</Alert>}
          <div className="flex gap-2">
            <Button className="flex-1" onClick={send} disabled={busy || !pick || needNote}>
              {busy ? m.common.loading : Q.send}
            </Button>
            <Button variant="secondary" onClick={() => setOpen(false)} disabled={busy}>
              {Q.back}
            </Button>
          </div>
        </div>
      </Modal>
    </>
  );
}
