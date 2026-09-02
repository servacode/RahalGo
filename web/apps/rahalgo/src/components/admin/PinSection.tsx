"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **الرمزُ السرّيُّ — قدرةٌ كانت مبنيّةً ولا بابَ إليها**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **(قِيس ٢٠٢٦-٠٩-٠٢:** فُحص مئتان وأربعةٌ وثمانون مساراً في المحرّك
 * مقابل مئةٍ وثمانيةٍ وخمسين نداءً في اللوحة. **وأربعةُ مساراتٍ للرمز
 * السرّيّ لا يناديها أحد.)**
 *
 * # وما كان يقع
 *
 * **يُنشئ الأدمنُ رمزَه مرّةً واحدةً عند أوّل دخول، ثمّ لا سبيلَ إليه
 * أبداً**: لا تغييراً ولا استعادة.
 *
 * **والخطرُ ملموسٌ لا نظريّ**: من رآه أحدٌ فوق كتفه **لا يملك أن
 * يبدّله**، ومن نسيه **أُقفل عليه بابُ اللوحة** — وهو البابُ الثاني
 * بعد كلمة المرور، فلا دخولَ بدونه.
 *
 * **والمحرّكُ كان جاهزاً طوال الوقت** — `‎/pin/state` و`‎/pin/change`
 * و`‎/pin/reset/request` و`‎/pin/reset/confirm`. **بل إنّ تعليقَ
 * `handlePinState` يقول نصّاً: «تقرؤها شاشةُ حسابي».** فكُتب الخادمُ
 * لشاشةٍ لم تُكتب.
 *
 * # ولماذا حالان لا شاشتان
 *
 * **التغييرُ والاستعادةُ فعلٌ واحدٌ في عين صاحبه**: «أريد رمزاً
 * جديداً». **والفرقُ أنّه يذكر القديمَ أو لا يذكره.**
 *
 * **فبابٌ واحدٌ وطريقان** — ومن نسي ضغط «نسيت الرمز؟» فانتقل، ولم
 * يبحث عن صفحةٍ أخرى.
 */

import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale, fmtDateTime } from "@rahalgo/i18n";
import { Button, FormSection, Input, Alert, IconStatus } from "@rahalgo/ui";
import { api, ApiError } from "@/lib/api";

const m = getMessages(defaultLocale);
const P = m.admin.myAccount.pin;

interface PinState {
  required: boolean;
  set: boolean;
  set_at?: string;
}

/** **أربعةُ أرقامٍ بقرار المالك** — و`pinLen` في المحرّك تقول الشيءَ نفسَه. */
const PIN_LEN = 4;

const isPin = (v: string) => v.length === PIN_LEN && /^[0-9]+$/.test(v);

export default function PinSection() {
  const [state, setState] = useState<PinState | null>(null);
  const [mode, setMode] = useState<"change" | "reset">("change");
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState("");
  const [note, setNote] = useState("");

  const [current, setCurrent] = useState("");
  const [next, setNext] = useState("");
  const [again, setAgain] = useState("");
  const [code, setCode] = useState("");
  const [codeSent, setCodeSent] = useState(false);

  const load = useCallback(() => {
    api<PinState>("/api/v1/pin/state")
      .then(setState)
      .catch(() => setState(null));
  }, []);

  useEffect(load, [load]);

  /** **ورسالةُ المحرّك تُترجم** — ومفتاحُها أدقُّ من «خطأ داخليّ». */
  function fail(e: unknown) {
    if (e instanceof ApiError) {
      const key = e.body?.message_key;
      const parts = typeof key === "string" ? key.split(".") : [];
      let node: unknown = m;
      for (const part of parts) {
        if (typeof node !== "object" || node === null) {
          node = null;
          break;
        }
        node = (node as Record<string, unknown>)[part];
      }
      setErr(typeof node === "string" ? node : m.errors.internal);
      return;
    }
    setErr(m.errors.internal);
  }

  function reset() {
    setCurrent("");
    setNext("");
    setAgain("");
    setCode("");
  }

  async function change(e: React.FormEvent) {
    e.preventDefault();
    setErr("");
    setNote("");
    if (!isPin(next)) return setErr(P.format);
    if (next !== again) return setErr(P.mismatch);
    setBusy(true);
    try {
      await api("/api/v1/pin/change", {
        method: "POST",
        body: JSON.stringify({ current, pin: next }),
      });
      setNote(P.changed);
      reset();
      load();
    } catch (e) {
      fail(e);
    } finally {
      setBusy(false);
    }
  }

  async function askCode() {
    setErr("");
    setNote("");
    setBusy(true);
    try {
      await api("/api/v1/pin/reset/request", { method: "POST" });
      setCodeSent(true);
      setNote(P.resetSent);
    } catch (e) {
      fail(e);
    } finally {
      setBusy(false);
    }
  }

  async function confirmReset(e: React.FormEvent) {
    e.preventDefault();
    setErr("");
    setNote("");
    if (!isPin(next)) return setErr(P.format);
    if (next !== again) return setErr(P.mismatch);
    setBusy(true);
    try {
      await api("/api/v1/pin/reset/confirm", {
        method: "POST",
        body: JSON.stringify({ code, pin: next }),
      });
      setNote(P.resetDone);
      setCodeSent(false);
      setMode("change");
      reset();
      load();
    } catch (e) {
      fail(e);
    } finally {
      setBusy(false);
    }
  }

  /** **ولا يُعرض لمن لا يلزمه رمز** — الحقلُ `required` من المحرّك. */
  if (!state?.required) return null;

  return (
    <FormSection title={P.title} icon={<IconStatus />}>
      <p className="mb-3 text-xs text-ink-muted">{P.hint}</p>

      <p className="mb-3 text-xs text-ink-muted">
        {state.set
          ? state.set_at
            ? P.setAt.replace("{d}", fmtDateTime(state.set_at))
            : ""
          : P.notSet}
      </p>

      {err && <Alert className="mb-3">{err}</Alert>}
      {note && (
        <p className="mb-3 rounded-control bg-success-tint px-2.5 py-1 text-xs text-success">
          {note}
        </p>
      )}

      {mode === "change" ? (
        <form onSubmit={change} className="space-y-3">
          <Input
            label={P.current}
            type="password"
            inputMode="numeric"
            maxLength={PIN_LEN}
            value={current}
            onChange={(e) => setCurrent(e.target.value)}
            autoComplete="current-password"
          />
          <Input
            label={P.next}
            type="password"
            inputMode="numeric"
            maxLength={PIN_LEN}
            value={next}
            onChange={(e) => setNext(e.target.value)}
            autoComplete="new-password"
          />
          <Input
            label={P.confirm}
            type="password"
            inputMode="numeric"
            maxLength={PIN_LEN}
            value={again}
            onChange={(e) => setAgain(e.target.value)}
            autoComplete="new-password"
          />
          <div className="flex items-center gap-3">
            <Button type="submit" disabled={busy}>
              {P.change}
            </Button>
            <button
              type="button"
              className="text-xs text-ink-muted underline"
              onClick={() => {
                setMode("reset");
                setErr("");
                setNote("");
                reset();
              }}
            >
              {P.forgot}
            </button>
          </div>
        </form>
      ) : (
        <form onSubmit={confirmReset} className="space-y-3">
          <p className="text-xs text-ink-muted">{P.resetHint}</p>

          {!codeSent ? (
            <div className="flex items-center gap-3">
              <Button type="button" disabled={busy} onClick={() => void askCode()}>
                {P.resetSend}
              </Button>
              <button
                type="button"
                className="text-xs text-ink-muted underline"
                onClick={() => {
                  setMode("change");
                  setErr("");
                  setNote("");
                  reset();
                }}
              >
                {P.back}
              </button>
            </div>
          ) : (
            <>
              <Input
                label={P.resetCode}
                inputMode="numeric"
                value={code}
                onChange={(e) => setCode(e.target.value)}
              />
              <Input
                label={P.next}
                type="password"
                inputMode="numeric"
                maxLength={PIN_LEN}
                value={next}
                onChange={(e) => setNext(e.target.value)}
                autoComplete="new-password"
              />
              <Input
                label={P.confirm}
                type="password"
                inputMode="numeric"
                maxLength={PIN_LEN}
                value={again}
                onChange={(e) => setAgain(e.target.value)}
                autoComplete="new-password"
              />
              <div className="flex items-center gap-3">
                <Button type="submit" disabled={busy}>
                  {P.resetConfirm}
                </Button>
                <button
                  type="button"
                  className="text-xs text-ink-muted underline"
                  onClick={() => {
                    setMode("change");
                    setCodeSent(false);
                    setErr("");
                    setNote("");
                    reset();
                  }}
                >
                  {P.back}
                </button>
              </div>
            </>
          )}
        </form>
      )}
    </FormSection>
  );
}
