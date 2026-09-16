"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **حالة التطبيق — ثلاثُ حالاتٍ فوق أربعِ رايات** (`PL`، ٢٠٢٦-٠٩-١٦)
 * ══════════════════════════════════════════════════════════════════════
 *
 * # ولمَ حالٌ لا رايات
 *
 * **والمالكُ لا يفكّر بأربعِ رايات** — **بل بثلاثِ حالات**: ما قبل
 * الافتتاح · واستعراضٌ قبله · ومفتوحٌ للعمل.
 *
 * **ومن قلّب أربعاً بيده نسي واحدةً** — **ففُتح التسجيلُ والسوقُ مقفلٌ**،
 * أو **فُتح الطلبُ ولا متجرَ فيه.**
 *
 * # ولا سلطانَ هنا
 *
 * **والرايةُ في `app_settings` هي الحقُّ المخزَّن** — **وهذه الشاشةُ
 * تكتبها بنداءٍ واحدٍ يكتب الأربعَ معاً أو لا يكتب شيئاً.**
 * **والمنعُ في المحرّك على كلّ حال.**
 *
 * # ويُعرَض ما سيتبدّل قبل أن يتبدّل
 *
 * **وزرٌّ يفتح المنصّةَ للناس لا يُضغط على ظنّ** — **فيُقال بالحرف:
 * هذا يُفتح وهذا يُغلق**، **وأبوابُ العمل لا يمسّها.**
 */

import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale, errorText } from "@rahalgo/i18n";
import {
  Alert,
  Confirm,
  PageHeader,
  Button,
  Badge,
  Card,
  Textarea,
  LoadingState,
  IconSettings,
} from "@rahalgo/ui";
import { api } from "@/lib/api";

const m = getMessages(defaultLocale);
const S = m.admin.appStatus;

/** **الرايةُ ومفتاحُها** — والاسمُ العربيُّ للعرض وحدَه. */
const CUSTOMER_KEYS = [
  "launch.customer_signup",
  "launch.customer_browse",
  "launch.customer_orders",
  "launch.customer_custom_orders",
] as const;

const PRESETS = ["pre_launch", "browse_only", "open"] as const;
type PresetName = (typeof PRESETS)[number];

interface PresetInfo {
  flags: Record<string, boolean>;
  enable: string[];
  disable: string[];
}

interface LaunchState {
  current: Record<string, boolean>;
  active: string;
  notice: string;
  presets: Record<string, PresetInfo>;
  untouched: Record<string, boolean>;
}

function label(key: string): string {
  return (S.flags as Record<string, string>)[key] ?? key;
}

export default function AppStatusPanel() {
  const [state, setState] = useState<LaunchState | null>(null);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const [busy, setBusy] = useState(false);
  const [asking, setAsking] = useState<PresetName | null>(null);
  const [noticeText, setNoticeText] = useState("");

  const load = useCallback(async () => {
    try {
      const r = await api<LaunchState>("/api/v1/admin/launch");
      setState(r);
      setNoticeText(r.notice ?? "");
      setError("");
    } catch (e) {
      setError(errorText(e));
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  async function apply(preset: PresetName) {
    setAsking(null);
    setBusy(true);
    try {
      await api("/api/v1/admin/launch/preset", {
        method: "POST",
        body: JSON.stringify({ preset }),
      });
      setNotice(S.applied);
      await load();
    } catch (e) {
      setError(errorText(e));
    } finally {
      setBusy(false);
    }
  }

  async function saveNotice() {
    setBusy(true);
    try {
      await api("/api/v1/admin/settings/launch.notice", {
        method: "PUT",
        body: JSON.stringify({ value: noticeText }),
      });
      setNotice(S.applied);
      await load();
    } catch (e) {
      setError(errorText(e));
    } finally {
      setBusy(false);
    }
  }

  if (!state) {
    return error ? <Alert>{error}</Alert> : <LoadingState />;
  }

  const activeLabel =
    state.active === ""
      ? S.custom
      : (S.presets as Record<string, string>)[state.active] ?? state.active;

  const pending = asking ? state.presets[asking] : null;

  return (
    <div>
      <PageHeader icon={IconSettings} title={S.title} />
      <p className="mb-4 text-sm text-ink-muted">{S.hint}</p>

      {error && <Alert className="mb-4">{error}</Alert>}
      {notice && <Alert className="mb-4">{notice}</Alert>}

      {/* **والحالُ الآن أوّلُ ما يُقرأ** — **ومن لا يعرف أين هو لا يختار.** */}
      <Card className="mb-4 p-4">
        <div className="mb-3 flex items-center gap-2">
          <span className="text-sm text-ink-muted">{S.current}</span>
          <Badge>{activeLabel}</Badge>
        </div>
        <ul className="grid gap-1 text-sm">
          {CUSTOMER_KEYS.map((k) => (
            <li key={k} className="flex items-center gap-2">
              <Badge>{state.current[k] ? S.on : S.off}</Badge>
              <span>{label(k)}</span>
            </li>
          ))}
        </ul>
      </Card>

      {/* **وأبوابُ العمل تُعرَض لتُطمئن** — **لا يمسّها النمط.** */}
      <Card className="mb-4 p-4">
        <p className="mb-2 text-sm text-ink-muted">{S.untouched}</p>
        <ul className="grid gap-1 text-sm">
          {Object.keys(state.untouched).map((k) => (
            <li key={k} className="flex items-center gap-2">
              <Badge>{state.untouched[k] ? S.on : S.off}</Badge>
              <span>{label(k)}</span>
            </li>
          ))}
        </ul>
      </Card>

      <div className="mb-4 grid gap-3">
        {PRESETS.map((p) => {
          const info = state.presets[p];
          const nothing =
            !info || (info.enable.length === 0 && info.disable.length === 0);
          return (
            <Card key={p} className="p-4">
              <div className="mb-1 font-medium">
                {(S.presets as Record<string, string>)[p]}
              </div>
              <p className="mb-3 text-sm text-ink-muted">
                {(S.presetHints as Record<string, string>)[p]}
              </p>
              <Button
                disabled={busy || nothing}
                onClick={() => setAsking(p)}
              >
                {nothing ? S.noChange : S.apply}
              </Button>
            </Card>
          );
        })}
      </div>

      {/* **ونصُّ الشاشة يُبدَّل من هنا** — **بلا نشرٍ في المتجر.** */}
      <Card className="p-4">
        <div className="mb-1 font-medium">{S.notice}</div>
        <p className="mb-3 text-sm text-ink-muted">{S.noticeHint}</p>
        <Textarea
          id="launch-notice"
          label=""
          value={noticeText}
          onChange={(e) => setNoticeText(e.target.value)}
        />
        <div className="mt-3">
          <Button disabled={busy} onClick={saveNotice}>
            {S.apply}
          </Button>
        </div>
      </Card>

      {/* ══════════════════════════════════════════════════════════════
          **ولا يُفتح شيءٌ على ظنّ** — **يُقال بالحرف ما سيتبدّل**
          ══════════════════════════════════════════════════════════ */}
      {asking && pending && (
        <Confirm
          open
          title={S.confirmTitle}
          confirmLabel={S.apply}
          busy={busy}
          onCancel={() => setAsking(null)}
          onConfirm={() => void apply(asking)}
          body={
            <div className="grid gap-2 text-sm">
              {pending.enable.length > 0 && (
                <div>
                  <span className="text-ink-muted">{S.willEnable}: </span>
                  {pending.enable.map(label).join(" · ")}
                </div>
              )}
              {pending.disable.length > 0 && (
                <div>
                  <span className="text-ink-muted">{S.willDisable}: </span>
                  {pending.disable.map(label).join(" · ")}
                </div>
              )}
              <div className="text-ink-muted">
                {S.untouched}:{" "}
                {Object.keys(state.untouched).map(label).join(" · ")}
              </div>
            </div>
          }
        />
      )}
    </div>
  );
}
