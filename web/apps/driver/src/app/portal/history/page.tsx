"use client";

/**
 * **سجلُّ السائق** — ما نفّذه، نجح أم فشل.
 *
 * # لماذا يلزم
 *
 * شاشتُه كانت «الآن» وحدَه: مهامٌّ جارية وطابورٌ قادم. **وما انتهى اختفى.**
 * فلا يعرف كم سلّم أمس، **ولا يجد طلباً يتذكّره ليقول فيه شيئاً.**
 *
 * # وزرُّ البلاغ هنا لا في مكانٍ آخر
 *
 * **البلاغُ يقع على طلبٍ بعينه** — «متجرٌ أوقفني نصفَ ساعة» بلا رقم طلبٍ لا
 * يُحقَّق. **ونموذجٌ في صفحةٍ منفصلةٍ يجعله يكتب من ذاكرته**، فيُقال «مطعمٌ
 * ما» ولا يُعرف أيّهما.
 */

import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale, fmtNum, fmtRef, fmtDateTime } from "@rahalgo/i18n";
import {
  Alert,
  Badge,
  Button,
  Card,
  EmptyState,
  LoadingState,
  Modal,
  Input,
  useLiveRefresh,
  IconOrder,
  IconStore,
  IconLocation,
  IconWarning,
  IconSupport,
  IconStar,
  PageHeader,
} from "@rahalgo/ui";
import { api, ApiError } from "@/lib/api";

const m = getMessages(defaultLocale);
const D = m.driver;

interface HistoryOrder {
  id: string;
  number: number;
  status: string;
  merchant_name: string;
  customer_name: string;
  address_text: string;
  total: number;
  cash_due: number;
  fail_reason: string;
  created_at: string;
  /** أقيّمتُ متجرَه — **ومن قيّم لا يُعرض عليه الزرُّ ثانيةً.** */
  merchant_rated: boolean;
  /** **وقف عند بابه فعلاً** — ومن لم يقف لا رأيَ له فيه. */
  can_rate_merchant: boolean;
}

type ReportReason = { code: string; against: string };

function errText(e: unknown): string {
  const key = e instanceof ApiError ? ((e.body.message_key ?? "").split(".").pop() ?? "") : "";
  return (m.errors as Record<string, string>)[key] ?? m.errors.internal;
}

export default function DriverHistoryPage() {
  const [rows, setRows] = useState<HistoryOrder[] | null>(null);
  const [error, setError] = useState("");
  const [reporting, setReporting] = useState<HistoryOrder | null>(null);
  const [reasons, setReasons] = useState<ReportReason[] | null>(null);
  const [reasonsErr, setReasonsErr] = useState(false);
  const [reason, setReason] = useState("");
  const [note, setNote] = useState("");
  const [busy, setBusy] = useState(false);
  const [done, setDone] = useState("");
  /** **وتقييمُ المتجر — والزبونُ يرى الطعامَ ولا يرى المطبخ.** */
  const [rating, setRating] = useState<HistoryOrder | null>(null);
  const [speed, setSpeed] = useState(0);
  const [conduct, setConduct] = useState(0);
  const [rateNote, setRateNote] = useState("");

  const load = useCallback(() => {
    api<{ orders: HistoryOrder[] }>("/api/v1/driver/orders/history")
      .then((r) => setRows(r.orders ?? []))
      .catch((e) => {
        setRows([]);
        setError(errText(e));
      });
  }, []);

  useEffect(load, [load]);
  useLiveRefresh(["order"], load);

  /**
   * **الأسبابُ تُجلَب مع الشاشة لا مع الضغطة.**
   *
   * وهي القاعدةُ نفسُها التي أنقذت أسبابَ التعذّر: نافذةٌ تُفتح قبل أن يصل
   * الجواب **تعرض «لا أسباب» فتُقرأ حكماً على الإعدادات**، والحقيقةُ أنّها
   * لم تُسأل بعد.
   */
  const loadReasons = useCallback(async () => {
    setReasonsErr(false);
    try {
      const r = await api<{ reasons: ReportReason[] }>("/api/v1/driver/orders/report-reasons");
      setReasons(r.reasons ?? []);
    } catch {
      setReasonsErr(true);
    }
  }, []);

  useEffect(() => {
    void loadReasons();
  }, [loadReasons]);

  async function submit() {
    const o = reporting;
    if (!o || !reason) return;
    setBusy(true);
    setError("");
    try {
      await api(`/api/v1/driver/orders/${o.id}/report`, {
        method: "POST",
        body: JSON.stringify({ reason, note: note.trim() }),
      });
      setReporting(null);
      setDone(D.history.reportSent);
    } catch (e) {
      setError(errText(e));
    } finally {
      setBusy(false);
    }
  }

  async function sendRating() {
    const o = rating;
    if (!o || speed < 1 || conduct < 1) return;
    setBusy(true);
    setError("");
    try {
      await api(`/api/v1/driver/orders/${o.id}/rate-merchant`, {
        method: "POST",
        body: JSON.stringify({
          speed_stars: speed,
          conduct_stars: conduct,
          comment: rateNote.trim(),
        }),
      });
      setRating(null);
      setDone(D.history.rateSent);
      load();
    } catch (e) {
      setError(errText(e));
    } finally {
      setBusy(false);
    }
  }

  if (rows === null) return <LoadingState />;

  return (
    <div className="space-y-4">
      {/* **وترويسةٌ مركزيّةٌ لا مبنيّةٌ باليد.**

          (كشفه جردُ السائق ٢٠٢٦-٠٨-٠٦: وصفُها عند **٣٫٨٢** — لأنّها كانت
           `<div>` عارياً على الخلفيّة المتدرّجة.)

          **وكانت تنسخ `PageHeader` حرفاً بحرف**: أيقونةٌ وعنوانٌ ووصفٌ
          بالمقاسات نفسِها. **ونسخةٌ لا ترث إصلاحاً**: لمّا صار الأصلُ زجاجاً
          بقيت هذه على الخلفيّة. */}
      <PageHeader icon={IconOrder} title={D.history.title} subtitle={D.history.subtitle} />

      {error && <Alert>{error}</Alert>}
      {done && (
        <Alert tone="success">{done}</Alert>
      )}

      {rows.length === 0 ? (
        <EmptyState icon={IconOrder} title={D.history.empty} />
      ) : (
        <div className="grid gap-3 xl:grid-cols-2">
          {rows.map((o) => {
            const failed = o.status === "failed";
            return (
              <Card key={o.id}>
                <div className="flex items-start justify-between gap-3">
                  <div className="min-w-0">
                    <p className="flex items-center gap-2 font-bold">
                      #{fmtRef(o.number)}
                      <Badge variant={failed ? "danger" : "success"}>
                        {m.orders.status[o.status as keyof typeof m.orders.status] ?? o.status}
                      </Badge>
                    </p>
                    <p className="mt-1 flex items-center gap-1.5 truncate text-sm text-ink-muted">
                      <IconStore size={14} />
                      {o.merchant_name}
                    </p>
                    <p className="flex items-center gap-1.5 truncate text-sm text-ink-muted">
                      <IconLocation size={14} />
                      {o.address_text}
                    </p>
                  </div>
                  <div className="shrink-0 text-end">
                    <p className="font-bold" dir="ltr">
                      {fmtNum(o.total)} {m.common.currency}
                    </p>
                    <p className="text-xs text-ink-muted">{fmtDateTime(o.created_at)}</p>
                  </div>
                </div>

                {/* **وسببُ التعذّر يُعرض** — من فشل طلبُه يعرف بماذا سُجّل عليه. */}
                {failed && o.fail_reason && (
                  <p className="mt-2 flex items-center gap-1.5 text-sm text-danger">
                    <IconWarning size={14} />
                    {m.common.failReasons[o.fail_reason as keyof typeof m.common.failReasons] ??
                      o.fail_reason}
                  </p>
                )}

                <div className="mt-3 flex items-center justify-end gap-1">
                  {/* **ومن وقف عند بابه يقيّمه** — والزرُّ لا يُعرض على من
                      لم يقف، **ولا على من قيّم**: زرٌّ يُضغط فيُردّ يُقرأ
                      عطباً لا قاعدة. */}
                  {o.can_rate_merchant &&
                    (o.merchant_rated ? (
                      <span className="px-2 text-xs text-ink-muted">{D.history.rated}</span>
                    ) : (
                      <Button
                        variant="ghost"
                        onClick={() => {
                          setSpeed(0);
                          setConduct(0);
                          setRateNote("");
                          setDone("");
                          setRating(o);
                        }}
                      >
                        <span className="flex items-center gap-1.5">
                          <IconStar size={15} />
                          {D.history.rate}
                        </span>
                      </Button>
                    ))}
                  <Button
                    variant="ghost"
                    onClick={() => {
                      setReason("");
                      setNote("");
                      setDone("");
                      if (reasonsErr) void loadReasons();
                      setReporting(o);
                    }}
                  >
                    <span className="flex items-center gap-1.5">
                      <IconSupport size={15} />
                      {D.history.report}
                    </span>
                  </Button>
                </div>
              </Card>
            );
          })}
        </div>
      )}

      <Modal
        open={reporting !== null}
        onClose={() => setReporting(null)}
        title={`${D.history.reportTitle} #${reporting ? fmtRef(reporting.number) : ""}`}
      >
        <div className="space-y-3">
          <p className="text-sm font-medium">{D.history.reportAgainst}</p>

          {reasonsErr && !reasons ? (
            <div className="space-y-2 rounded-control border border-danger/40 bg-danger/5 px-3 py-2">
              <Alert>{D.history.reasonsError}</Alert>
              <Button variant="secondary" onClick={() => void loadReasons()}>
                {m.common.retry}
              </Button>
            </div>
          ) : !reasons ? (
            <p className="text-sm text-ink-muted">{m.common.loading}</p>
          ) : (
            <div className="space-y-1.5">
              {reasons.map((x) => (
                <label
                  key={x.code}
                  className={`flex cursor-pointer items-center gap-2.5 rounded-control border px-3 py-2 text-sm transition-colors ${
                    reason === x.code
                      ? "border-accent bg-accent/10 font-medium"
                      : "border-line hover:border-accent/60"
                  }`}
                >
                  <input
                    type="radio"
                    name="report-reason"
                    className="accent-accent"
                    checked={reason === x.code}
                    onChange={() => setReason(x.code)}
                  />
                  {/* **ورمزٌ بلا ترجمةٍ يُعرض كما هو** — فمن أضاف سبباً في
                      الخادم ونسي القاموسَ يرى نقصَه في الشاشة. */}
                  {D.history.reportReasons[
                    x.code as keyof typeof D.history.reportReasons
                  ] ?? x.code}
                </label>
              ))}
            </div>
          )}

          <Input
            id="report-note"
            label={D.history.reportNote}
            value={note}
            onChange={(e) => setNote(e.target.value)}
          />
          <p className="text-xs text-ink-muted">{D.history.reportHint}</p>

          <div className="flex justify-end gap-2">
            <Button variant="secondary" onClick={() => setReporting(null)}>
              {m.common.cancel}
            </Button>
            <Button variant="danger" disabled={!reason || busy} onClick={() => void submit()}>
              {D.history.reportSend}
            </Button>
          </div>
        </div>
      </Modal>
      {/* **نجمتان لا واحدة.**

          «المتجر» و«التعامل» شيئان يفترقان: مطعمٌ سريعٌ فظّ، وآخرُ بطيءٌ
          مهذّب. **ونجمةٌ واحدةٌ تجمعهما تُخفي أيَّهما المشكلة** — فلا يُعرف
          أنُكلّم المطبخَ أم صاحبَ المحلّ. */}
      <Modal
        open={rating !== null}
        onClose={() => setRating(null)}
        title={`${D.history.rateTitle} — ${rating?.merchant_name ?? ""}`}
      >
        <div className="space-y-3">
          <StarRow label={D.history.rateSpeed} value={speed} onPick={setSpeed} />
          <StarRow label={D.history.rateConduct} value={conduct} onPick={setConduct} />
          <Input
            id="rate-note"
            label={D.history.rateComment}
            value={rateNote}
            onChange={(e) => setRateNote(e.target.value)}
          />
          <div className="flex justify-end gap-2">
            <Button variant="secondary" onClick={() => setRating(null)}>
              {m.common.cancel}
            </Button>
            <Button
              disabled={busy || speed < 1 || conduct < 1}
              onClick={() => void sendRating()}
            >
              {D.history.rateSend}
            </Button>
          </div>
        </div>
      </Modal>
    </div>
  );
}

/** صفُّ نجومٍ يُضغط — **وخمسٌ لا عشر**: مقياسٌ يعرفه الناسُ بلا شرح. */
function StarRow({
  label,
  value,
  onPick,
}: {
  label: string;
  value: number;
  onPick: (n: number) => void;
}) {
  return (
    <div className="flex items-center justify-between gap-3">
      <span className="text-sm font-medium">{label}</span>
      <span className="flex gap-1">
        {[1, 2, 3, 4, 5].map((n) => (
          <button
            key={n}
            type="button"
            aria-label={`${label} ${n}`}
            onClick={() => onPick(n)}
            className={n <= value ? "text-accent-text" : "text-line"}
          >
            <IconStar size={22} />
          </button>
        ))}
      </span>
    </div>
  );
}
