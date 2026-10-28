"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **‏/loc/<رمز> — «أرسل موقعي»** (قرارُ المالك ٢٠٢٦-١٠-٢٩)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **يصل المستلِمَ رابطُها بالواتساب** حين يضغط المتجرُ «اطلب الموقع من
 * المستلم» — فلا نسخَ رابطٍ ولا لصق. **صفحةٌ عامّةٌ بلا حساب**، فيها اسمُ
 * المتجر وحدَه **وزرٌّ واحدٌ كبير.**
 *
 * **والموقعُ من الجهاز بدقّةٍ عالية** وتُعرض دقّتُه. **ومن منع الإذنَ أو
 * أراد غيرَ مكانه** يضغط على الخريطة ثمّ «أرسل هذا الموقع».
 *
 * **والرابطُ لمرّةٍ واحدة** — ينتهي بعد ٢٤ ساعة أو عند أوّل إرسال
 * (`backend/internal/server/recipient_location.go`).
 */

import { lazy, Suspense, useCallback, useEffect, useState } from "react";
import { useParams } from "next/navigation";
import { getMessages, defaultLocale, errorText, fmtNum } from "@rahalgo/i18n";
import { Alert, Button, IconLocation, LoadingState } from "@rahalgo/ui";
import { api, ApiError } from "@/lib/api";
import { fill } from "@/lib/customer/texts";

const m = getMessages(defaultLocale);
const L = m.locShare;

const PickMap = lazy(() => import("@rahalgo/ui/map").then((mod) => ({ default: mod.PickMap })));

type Info = { store_name: string; mode: "send" | "fetch"; status: "pending" | "received" | "expired" };
type Phase = "loading" | "ready" | "done" | "closed" | "missing";

export default function LocationSharePage() {
  const { token } = useParams<{ token: string }>();
  const [info, setInfo] = useState<Info | null>(null);
  const [phase, setPhase] = useState<Phase>("loading");
  const [loadError, setLoadError] = useState("");
  const [locating, setLocating] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [accuracy, setAccuracy] = useState<number | null>(null);
  const [mapOpen, setMapOpen] = useState(false);
  const [pin, setPin] = useState<{ lat: number; lng: number } | null>(null);

  const load = useCallback(() => {
    setLoadError("");
    api<Info>(`/api/v1/public/location-requests/${encodeURIComponent(token)}`)
      .then((r) => {
        setInfo(r);
        setPhase(r.status === "pending" ? "ready" : r.status === "received" ? "done" : "closed");
      })
      .catch((e) => {
        if (e instanceof ApiError && e.status === 404) setPhase("missing");
        else setLoadError(errorText(e));
      });
  }, [token]);

  useEffect(load, [load]);

  async function submit(lat: number, lng: number, acc: number | null) {
    setBusy(true);
    setError("");
    try {
      await api(`/api/v1/public/location-requests/${encodeURIComponent(token)}`, {
        method: "POST",
        body: JSON.stringify({ lat, lng, accuracy_m: acc }),
      });
      setPhase("done");
    } catch (e) {
      if (e instanceof ApiError && e.status === 410) setPhase("closed");
      else setError(errorText(e));
    } finally {
      setBusy(false);
    }
  }

  function locate() {
    setError("");
    if (!("geolocation" in navigator)) {
      setMapOpen(true);
      setError(L.denied);
      return;
    }
    setLocating(true);
    navigator.geolocation.getCurrentPosition(
      (p) => {
        setLocating(false);
        const acc = Math.round(p.coords.accuracy);
        setAccuracy(acc);
        void submit(p.coords.latitude, p.coords.longitude, acc);
      },
      () => {
        setLocating(false);
        setMapOpen(true);
        setError(L.denied);
      },
      { enableHighAccuracy: true, timeout: 20000, maximumAge: 0 },
    );
  }

  const store = info?.store_name ?? "";

  return (
    <main className="flex min-h-screen items-center justify-center px-4 py-8">
      <div className="surface flex w-full max-w-md flex-col items-center gap-4 rounded-card p-6 text-center">
        <span className="flex h-16 w-16 items-center justify-center rounded-full bg-accent-tint text-accent">
          <IconLocation size={32} />
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

        {phase === "done" && (
          <div className="flex flex-col gap-2">
            <p className="heading-card text-success">{fill(L.done, { store })}</p>
            {accuracy != null && <p className="text-sm text-ink-muted">{fill(L.accuracy, { m: fmtNum(accuracy) })}</p>}
          </div>
        )}

        {phase === "ready" && (
          <>
            <h1 className="heading-page">{fill(info?.mode === "fetch" ? L.titleFetch : L.title, { store })}</h1>
            <p className="text-sm text-ink-muted">{L.hint}</p>
            <Button size="lg" className="w-full" disabled={locating || busy} onClick={locate}>
              {locating ? L.locating : busy && !mapOpen ? L.sending : L.send}
            </Button>
            {accuracy != null && <p className="text-xs text-ink-muted">{fill(L.accuracy, { m: fmtNum(accuracy) })}</p>}
            {error && <Alert>{error}</Alert>}
            {!mapOpen ? (
              <button
                type="button"
                className="text-sm font-bold text-accent underline-offset-4 hover:underline"
                onClick={() => setMapOpen(true)}
              >
                {L.useMap}
              </button>
            ) : (
              <div className="flex w-full flex-col gap-2">
                <p className="text-xs text-ink-muted">{L.pinHint}</p>
                <Suspense fallback={<div className="h-72 w-full animate-pulse rounded-card bg-field" />}>
                  <PickMap
                    lat={pin?.lat ?? null}
                    lng={pin?.lng ?? null}
                    height="h-72"
                    onPick={(la: number, ln: number) => setPin({ lat: la, lng: ln })}
                    onLocated={(la: number, ln: number) => setPin({ lat: la, lng: ln })}
                  />
                </Suspense>
                <Button
                  variant="secondary"
                  className="w-full"
                  disabled={!pin || busy}
                  onClick={() => pin && void submit(pin.lat, pin.lng, null)}
                >
                  {busy ? L.sending : L.sendPin}
                </Button>
              </div>
            )}
          </>
        )}
      </div>
    </main>
  );
}
