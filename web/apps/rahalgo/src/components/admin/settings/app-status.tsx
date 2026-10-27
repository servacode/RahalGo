"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **حالة التطبيق — ثلاثُ حالاتٍ فوق أربعةِ أبواب** (`PL`، ٢٠٢٦-٠٩-١٦)
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
  Button,
  Badge,
  Textarea,
  LoadingState,
  useLiveRefresh,
} from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";

const m = getMessages(defaultLocale);
const S = m.admin.appStatus;

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

/**
 * **بطاقةٌ واحدة** (تدقيقُ ٢٠٢٦-١٠-٢٧): الحالُ الآن وثلاثُ حبوبٍ للحالات، **والأبوابُ السبعةُ
 * مفاتيحُ تحتها في الصفحة** — كانت تُعرض هنا مرّةً ثانيةً شاراتٍ رماديّةً بأسماءٍ أخرى.
 * **والتبديلُ وحفظُ النصّ لمن يملك الإعداداتِ العامّة** — غيرُه كان يكبس فيُردّ ٤٠٣.
 */
export default function AppStatusPanel() {
  const { can } = useAuth();
  const may = can("settings.general.manage");
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
  // **ويتحدّث مع أيّ تغييرٍ في الإعدادات** — **ولا يمسّ نصَّ الإشعار وهو يُكتب**.
  useLiveRefresh(["settings"], () => {
    api<LaunchState>("/api/v1/admin/launch").then(setState).catch(() => undefined);
  });

  async function apply(preset: PresetName) {
    setAsking(null);
    setBusy(true);
    setNotice("");
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
    setNotice("");
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
    <section className="surface space-y-4 p-4">
      <div className="flex flex-wrap items-start justify-between gap-2">
        <div className="min-w-0">
          <h2 className="heading-card">{S.title}</h2>
          <p className="text-sm text-ink-muted">{S.hint}</p>
        </div>
        <Badge variant={state.active === "open" ? "success" : "warning"}>
          {S.current}: {activeLabel}
        </Badge>
      </div>

      {error && <Alert>{error}</Alert>}
      {notice && <Alert tone="success">{notice}</Alert>}

      {/* **تلات حبوب** — الحالُ الحاليّةُ ملوّنة، والكبسُ على غيرها يسأل ويقول شو بيتبدّل. */}
      <div className="space-y-1.5">
        <span className="block text-xs font-bold text-ink-muted">{S.choose}</span>
        <div className="grid grid-cols-1 gap-2 sm:grid-cols-3">
          {PRESETS.map((p) => {
            const on = state.active === p;
            const info = state.presets[p];
            const nothing = !info || (info.enable.length === 0 && info.disable.length === 0);
            return (
              <button
                key={p}
                type="button"
                aria-pressed={on}
                disabled={!may || busy || on || nothing}
                onClick={() => setAsking(p)}
                className={`rounded-card border p-3 text-start transition-colors disabled:cursor-default ${
                  on ? "border-accent bg-accent text-on-bright" : "border-line-soft hover:bg-row-hover"
                }`}
              >
                <span className="block font-bold">{(S.presets as Record<string, string>)[p]}</span>
                <span className={`mt-0.5 block text-xs ${on ? "" : "text-ink-muted"}`}>
                  {(S.presetHints as Record<string, string>)[p]}
                </span>
              </button>
            );
          })}
        </div>
        {!may && <p className="text-xs text-ink-muted">{S.viewOnly}</p>}
      </div>

      {/* **ونصُّ الشاشة يُبدَّل من هنا** — **بلا نشرٍ في المتجر.** */}
      <div className="space-y-2 border-t border-line-soft pt-3">
        <span className="block font-bold">{S.notice}</span>
        <p className="text-sm text-ink-muted">{S.noticeHint}</p>
        <Textarea
          id="launch-notice"
          label=""
          value={noticeText}
          disabled={!may}
          onChange={(e) => setNoticeText(e.target.value)}
        />
        {may && (
          <Button
            className="w-full sm:w-auto"
            disabled={busy || noticeText === (state.notice ?? "")}
            onClick={saveNotice}
          >
            {m.common.save}
          </Button>
        )}
      </div>

      {/* **ولا يُفتح شيءٌ على ظنّ** — **يُقال بالحرف ما سيتبدّل** */}
      {asking && pending && (
        <Confirm
          open
          title={S.confirmTitle}
          confirmLabel={S.apply}
          tone="primary"
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
    </section>
  );
}
