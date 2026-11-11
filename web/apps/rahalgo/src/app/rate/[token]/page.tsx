"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **‏/rate/<رمز> — «كيف كانت الخدمة؟»** (قرارُ المالك ٢٠٢٦-١١-٠٤)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **يصل الزبونَ رابطُها بالواتساب بعد تسليم طلبه بعشر دقائق.** صفحةٌ عامّةٌ بلا
 * حساب: نجومُ الخدمة (لازمة)، ونجومُ السائق (اختياريّة، إن كان للطلب سائق)،
 * وكلمةٌ عن التطبيق (اختياريّة). **والرابطُ لمرّةٍ واحدة** — ينتهي بعد ٧ أيّام
 * أو عند أوّل إرسال (`backend/internal/server/order_feedback.go`).
 */

import { useCallback, useEffect, useState } from "react";
import { useParams } from "next/navigation";
import { getMessages, defaultLocale, errorText, fmtRef } from "@rahalgo/i18n";
import { Alert, Button, IconStar, LoadingState, Stars, Textarea } from "@rahalgo/ui";
import { api, ApiError } from "@/lib/api";
import { fill } from "@/lib/customer/texts";

const m = getMessages(defaultLocale);
const L = m.rateFeedback;

type Info = { order_number: number; has_driver: boolean; status: "pending" | "answered" | "expired" };
type Phase = "loading" | "ready" | "done" | "closed" | "missing";

export default function RateFeedbackPage() {
  const { token } = useParams<{ token: string }>();
  const [info, setInfo] = useState<Info | null>(null);
  const [phase, setPhase] = useState<Phase>("loading");
  const [loadError, setLoadError] = useState("");
  // **خمسُ نجومٍ افتراضاً** (قرارُ المالك ٢٠٢٦-١١-١١) — كنافذة التطبيق والموقع.
  const [service, setService] = useState(5);
  const [driver, setDriver] = useState(5);
  const [feedback, setFeedback] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const load = useCallback(() => {
    setLoadError("");
    api<Info>(`/api/v1/public/feedback-requests/${encodeURIComponent(token)}`)
      .then((r) => {
        setInfo(r);
        setPhase(r.status === "pending" ? "ready" : "closed");
      })
      .catch((e) => {
        if (e instanceof ApiError && e.status === 404) setPhase("missing");
        else setLoadError(errorText(e));
      });
  }, [token]);

  useEffect(load, [load]);

  async function submit() {
    if (service < 1) {
      setError(L.serviceRequired);
      return;
    }
    setBusy(true);
    setError("");
    try {
      await api(`/api/v1/public/feedback-requests/${encodeURIComponent(token)}`, {
        method: "POST",
        body: JSON.stringify({
          service_stars: service,
          driver_stars: info?.has_driver && driver > 0 ? driver : null,
          feedback: feedback.trim(),
        }),
      });
      setPhase("done");
    } catch (e) {
      if (e instanceof ApiError && e.status === 410) setPhase("closed");
      else setError(errorText(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <main className="flex min-h-screen items-center justify-center px-4 py-8">
      <div className="surface flex w-full max-w-md flex-col items-center gap-4 rounded-card p-6 text-center">
        <span className="flex h-16 w-16 items-center justify-center rounded-full bg-accent-tint text-accent">
          <IconStar size={32} />
        </span>

        {phase === "loading" &&
          (loadError ? (
            <>
              <Alert>{loadError}</Alert>
              <Button variant="secondary" onClick={load}>
                {m.common.retry}
              </Button>
            </>
          ) : (
            <LoadingState variant="inline" label={L.loading} />
          ))}

        {phase === "missing" && <p className="heading-card">{L.notFound}</p>}
        {phase === "closed" && <p className="heading-card">{L.closed}</p>}
        {phase === "done" && <p className="heading-card text-success">{L.thanks}</p>}

        {phase === "ready" && info && (
          <>
            <h1 className="heading-page">{fill(L.title, { n: fmtRef(info.order_number) })}</h1>

            <div className="flex w-full flex-col items-center gap-1">
              <p className="text-sm font-bold">{L.service}</p>
              <Stars value={service} size="lg" onChange={setService} />
            </div>

            {info.has_driver && (
              <div className="flex w-full flex-col items-center gap-1">
                <p className="text-sm font-bold">{L.driver}</p>
                <Stars value={driver} size="lg" onChange={setDriver} />
              </div>
            )}

            <div className="w-full text-start">
              <Textarea
                label={L.appLabel}
                placeholder={L.appPlaceholder}
                value={feedback}
                maxLength={1000}
                rows={3}
                onChange={(e) => setFeedback(e.target.value)}
              />
            </div>

            {error && <Alert>{error}</Alert>}
            <Button size="lg" className="w-full" disabled={busy} onClick={() => void submit()}>
              {busy ? L.sending : L.send}
            </Button>
          </>
        )}
      </div>
    </main>
  );
}
