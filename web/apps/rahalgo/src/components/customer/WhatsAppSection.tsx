"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **توثيقُ واتساب من «حسابي»** (مرآةُ `WhatsAppVerify` في `AccountScreen.kt`)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **على رقم الحساب نفسِه لا على رقمٍ ثانٍ** (قرارُ المالك ٢٠٢٦-٠٨-١٢) —
 * فلا يُسأل عن رقم. **والطريقُ مقلوبٌ كالاستعادة**: يُفتح واتساب برسالةٍ
 * جاهزة (`/auth/wa/ticket`، `purpose=verify`) **فيصله الرمزُ ردّاً**، ثمّ
 * يُكتب هنا (`/auth/whatsapp/confirm`).
 *
 * **وحالُه تُقال قبل الزرّ** — والموثَّقُ لا يُدعى إلى فعلٍ تمّ. **والمحرّكُ
 * يردّ الطلبَ بـ`whatsapp_required` إن أوجبته اللوحة** — فيُساق إلى هنا.
 */

import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale, errorText } from "@rahalgo/i18n";
import { Alert, Button, FormSection, LoadingState, OtpInput, IconVerified } from "@rahalgo/ui";
import { api } from "@/lib/api";
import type { MeSummary } from "@/lib/customer/types";
import { useShopToast } from "./toast";

const m = getMessages(defaultLocale);
const A = m.shop.account;
const T = m.shop.auth;

export function WhatsAppSection() {
  const toast = useShopToast();
  const [me, setMe] = useState<MeSummary | null>(null);
  const [waUrl, setWaUrl] = useState("");
  const [waiting, setWaiting] = useState(false);
  const [code, setCode] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const load = useCallback(() => {
    api<MeSummary>("/api/v1/me/summary")
      .then(setMe)
      .catch((err) => setError(errorText(err)));
  }, []);

  useEffect(load, [load]);

  const ask = async () => {
    if (!me) return;
    setBusy(true);
    setError("");
    try {
      const t = await api<{ wa_url: string }>("/api/v1/auth/wa/ticket", {
        method: "POST",
        body: JSON.stringify({ phone: me.phone, purpose: "verify" }),
      });
      setWaUrl(t.wa_url);
      setWaiting(true);
      setCode("");
      try {
        window.open(t.wa_url, "_blank", "noopener");
      } catch {
        /* **وسفاري يمنع نافذةً بعد انتظار** — الرابطُ ظاهرٌ تحت يده. */
      }
    } catch (err) {
      setError(errorText(err));
    } finally {
      setBusy(false);
    }
  };

  const confirmCode = async (c: string = code) => {
    if (!me || !c.trim()) return;
    setBusy(true);
    setError("");
    try {
      await api("/api/v1/auth/whatsapp/confirm", {
        method: "POST",
        body: JSON.stringify({ phone: me.phone, code: c.trim() }),
      });
      setWaiting(false);
      setCode("");
      toast(A.waDone, "success");
      load();
    } catch (err) {
      setError(errorText(err));
    } finally {
      setBusy(false);
    }
  };

  const verified = me?.whatsapp_verified === true;

  return (
    <div id="whatsapp" className="scroll-mt-20">
      <FormSection title={A.waTitle} icon={<IconVerified />}>
        {!me ? (
          error ? <Alert>{error}</Alert> : <LoadingState variant="inline" />
        ) : (
          <div className="flex flex-col gap-3">
            <p className={`text-sm font-bold ${verified ? "text-success" : "text-danger"}`}>
              {verified ? A.waVerified : A.waUnverified}
            </p>
            {!verified && !waiting && (
              <Button onClick={ask} disabled={busy} className="w-full sm:w-fit">
                {busy ? T.busy : A.waVerify}
              </Button>
            )}
            {waiting && (
              <form
                className="flex flex-col gap-3"
                onSubmit={(e) => {
                  e.preventDefault();
                  void confirmCode();
                }}
              >
                {waUrl && (
                  <p className="text-sm text-ink-muted">
                    {T.resetWaHint}{" "}
                    <a href={waUrl} target="_blank" rel="noreferrer" className="font-bold text-accent-text underline">
                      {T.resetWaLink}
                    </a>
                  </p>
                )}
                <p className="text-xs text-ink-muted">{A.waCode}</p>
                <OtpInput value={code} onChange={setCode} boxLabel={m.auth.otpBoxLabel} onComplete={(v) => void confirmCode(v)} />
                <div className="flex gap-2">
                  <Button type="submit" disabled={busy || !code.trim()} className="flex-1">
                    {busy ? T.busy : A.waConfirm}
                  </Button>
                  <Button type="button" variant="secondary" onClick={() => setWaiting(false)} disabled={busy}>
                    {A.waCancel}
                  </Button>
                </div>
              </form>
            )}
            {error && <Alert>{error}</Alert>}
          </div>
        )}
      </FormSection>
    </div>
  );
}
