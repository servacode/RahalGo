"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **«فعّل الإشعارات» — زرٌّ واحدٌ بحالات الجهاز كلِّها** (٢٠٢٦-١٠-١٧)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **ثلاثةُ مواضع**: حسابُ الزبون (`card`)، ومرّةً بعد أوّل طلب (`prompt`
 * يُغلق ولا يعود)، **ولوحةُ المتجر بجانب زرّ الصوت** (`banner` كبير).
 *
 * **والإذنُ من الضغطة** — ولا يُطلب عند الفتح. **وكلُّ حالٍ تقول نفسَها**:
 * آيفون من سفاري يُقال له أن يضيف الأيقونة أوّلاً، **والممنوعُ يُقال له أين
 * يفتحه**، **وما لا يدعمه المتصفّحُ يُقال بصريحه** — لا زرٌّ لا يفعل شيئاً.
 */

import { useEffect, useState } from "react";
import { getMessages, defaultLocale, errorText } from "@rahalgo/i18n";
import { Button, IconBell, IconBellOff, IconClose } from "@rahalgo/ui";
import { readJSON, writeJSON } from "@/lib/customer/storage";
import {
  disableWebPush,
  enableWebPush,
  webPushState,
  WebPushDisabledError,
  type WebPushApp,
  type WebPushState,
} from "@/lib/webpush";

const m = getMessages(defaultLocale);
const W = m.shared.webpush;

const PROMPT_SEEN = "rahalgo_webpush_prompt_seen";

export default function WebPushCard({
  app,
  variant = "card",
  notes = true,
}: {
  app: WebPushApp;
  variant?: "card" | "prompt" | "banner";
  /** **في اللافتة: أتُعرض حالُ المنع وآيفون؟** — في صفحة المتجر الأولى وحدَها، لا في كلّ صفحة. */
  notes?: boolean;
}) {
  const [state, setState] = useState<WebPushState | null>(null);
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState("");
  const [hidden, setHidden] = useState(variant === "prompt");

  useEffect(() => {
    let alive = true;
    if (variant === "prompt" && readJSON<boolean>(PROMPT_SEEN, false)) return;
    void webPushState(app).then((s) => {
      if (!alive) return;
      setState(s);
      if (variant === "prompt") setHidden(s === "on" || s === "unsupported");
    });
    return () => {
      alive = false;
    };
  }, [app, variant]);

  if (hidden || state === null) return null;
  // **واللافتةُ تختفي حين تعمل** — لا تُلحّ على من فعّلها.
  if (variant === "banner" && state === "on") return null;
  if (variant === "banner" && (state === "unsupported" || (state !== "off" && !notes))) return null;

  const title = app === "merchant" ? W.merchantTitle : W.customerTitle;
  const body = app === "merchant" ? W.merchantBody : W.customerBody;

  const dismiss = () => {
    writeJSON(PROMPT_SEEN, true);
    setHidden(true);
  };

  const enable = async () => {
    setBusy(true);
    setErr("");
    try {
      const s = await enableWebPush(app);
      setState(s);
      if (variant === "prompt") writeJSON(PROMPT_SEEN, true);
    } catch (e) {
      setErr(e instanceof WebPushDisabledError ? W.serverOff : errorText(e) || W.failed);
    } finally {
      setBusy(false);
    }
  };

  const disable = async () => {
    setBusy(true);
    try {
      await disableWebPush(app);
      setState("off");
    } finally {
      setBusy(false);
    }
  };

  const note =
    state === "ios-home"
      ? W.iosHome
      : state === "ios-old"
        ? W.iosOld
        : state === "denied"
          ? W.denied
          : state === "unsupported"
            ? W.unsupported
            : "";

  if (variant === "banner" && state === "off") {
    return (
      <div className="flex flex-col gap-1">
        <button
          type="button"
          onClick={() => void enable()}
          disabled={busy}
          data-testid="enable-webpush"
          className="flex w-full items-center justify-center gap-3 rounded-card bg-accent px-4 py-5 text-on-bright shadow-e3 hover:opacity-90 disabled:opacity-60"
        >
          <IconBell size={28} className="shrink-0" />
          <span className="flex flex-col items-start text-start">
            <span className="heading-card">{busy ? W.enabling : title}</span>
            <span className="text-xs">{body}</span>
          </span>
        </button>
        {err && <p className="text-xs text-danger">{err}</p>}
      </div>
    );
  }

  return (
    <div
      className="surface flex flex-col gap-2 rounded-card p-3"
      data-testid={`webpush-${variant}`}
    >
      <div className="flex items-start gap-2">
        {state === "on" ? (
          <IconBell size={20} className="shrink-0 text-success" />
        ) : (
          <IconBellOff size={20} className="shrink-0 text-ink-muted" />
        )}
        <div className="min-w-0 flex-1">
          <p className="font-bold">{variant === "prompt" ? W.afterOrderTitle : title}</p>
          <p className="text-xs text-ink-muted">
            {state === "on" ? W.enabled : variant === "prompt" ? W.afterOrderBody : body}
          </p>
        </div>
        {variant === "prompt" && (
          <button type="button" onClick={dismiss} aria-label={W.later} className="shrink-0 p-1 text-ink-muted">
            <IconClose size={16} />
          </button>
        )}
      </div>
      {note && <p className="text-xs text-warning">{note}</p>}
      {err && <p className="text-xs text-danger">{err}</p>}
      <div className="flex flex-wrap gap-2">
        {state === "off" && (
          <Button onClick={() => void enable()} disabled={busy}>
            {busy ? W.enabling : W.enable}
          </Button>
        )}
        {state === "on" && variant === "card" && (
          <Button variant="secondary" onClick={() => void disable()} disabled={busy}>
            {W.disable}
          </Button>
        )}
        {variant === "prompt" && state !== "on" && (
          <Button variant="secondary" onClick={dismiss}>
            {W.later}
          </Button>
        )}
      </div>
    </div>
  );
}
