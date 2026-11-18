"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **‏/dashboard/store-message — «رسالة للمتاجر»** (قرارُ المالك ٢٠٢٦-١١-١٨)
 * ══════════════════════════════════════════════════════════════════════
 *
 * رسالةٌ واحدةٌ (تذكيرُ تحديث الأسعار) تصل أصحابَ المتاجر الفعّالة على واتساب وفي تطبيق المتجر —
 * بزرّ «ابعت هلّق»، أو «تجربةً» لمتجرٍ واحد، أو آلياً (يومياً/أسبوعياً بساعةٍ بتوقيت دمشق).
 *
 * **والعددُ يُعرض ويُطلب عليه تأكيدٌ داخلَ الصفحة** (لا `window.confirm`): ما أُرسل لا يُسحب.
 * **والإرسالُ في الخادم لا هنا**: متسلسلٌ بفاصلٍ عشوائيّ حمايةً لرقم البوت، فالصفحةُ تقرأ الحالَ دورياً.
 */

import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale, errorText, fmtDateTime, fmtNum } from "@rahalgo/i18n";
import {
  AdminPage, Alert, Badge, Button, Checkbox, FormSection, IconWhatsApp, Input, Modal,
  FormActions, Select, Textarea,
} from "@rahalgo/ui";
import { api } from "@/lib/api";

const m = getMessages(defaultLocale);
const S = m.admin.storeMessage;

type Config = {
  title: string;
  text: string;
  auto_enabled: boolean;
  auto_frequency: "daily" | "weekly";
  auto_weekday: number;
  auto_hour: number;
};
type Run = {
  id: string;
  trigger: "manual" | "auto" | "test";
  status: "running" | "done" | "stopped";
  stop_reason: string;
  target: number;
  sent: number;
  failed: number;
  started_at: string;
  finished_at: string | null;
};
type State = {
  config: Config;
  eligible: number;
  already_today: number;
  bot_ready: boolean;
  runs: Run[];
};
type Store = { id: string; name: string };

const STATUS_TONE = { running: "info", done: "success", stopped: "danger" } as const;

function fill(t: string, v: Record<string, string>) {
  return Object.entries(v).reduce((s, [k, x]) => s.replace(`{${k}}`, x), t);
}

export default function StoreMessagePage() {
  const [st, setSt] = useState<State | null>(null);
  const [cfg, setCfg] = useState<Config | null>(null);
  const [dirty, setDirty] = useState(false);
  const [stores, setStores] = useState<Store[]>([]);
  const [testId, setTestId] = useState("");
  const [confirming, setConfirming] = useState(false);
  const [busy, setBusy] = useState(false);
  const [ok, setOk] = useState("");
  const [error, setError] = useState("");

  const load = useCallback(async () => {
    try {
      const res = await api<State>("/api/v1/admin/store-message");
      setSt(res);
      setCfg((cur) => cur ?? res.config);
    } catch (e) {
      setError(errorText(e));
    }
  }, []);

  useEffect(() => {
    void load();
    api<{ stores: Store[] }>("/api/v1/admin/store-message/stores")
      .then((r) => setStores(r.stores))
      .catch((e) => setError(errorText(e)));
  }, [load]);

  // **الإرسالُ يمشي بالخلفيّة** — تُقرأ حالُ السجلّ كلَّ ١٠ ثوانٍ ما دامت تشغيلةٌ جارية.
  const running = st?.runs.some((r) => r.status === "running") ?? false;
  useEffect(() => {
    if (!running) return;
    const t = setInterval(() => void load(), 10000);
    return () => clearInterval(t);
  }, [running, load]);

  function edit(patch: Partial<Config>) {
    setCfg((c) => (c ? { ...c, ...patch } : c));
    setDirty(true);
    setOk("");
  }

  async function save() {
    if (!cfg) return;
    setBusy(true);
    setError("");
    try {
      await api("/api/v1/admin/store-message/config", {
        method: "PUT",
        body: JSON.stringify(cfg),
      });
      setDirty(false);
      setOk(S.saved);
      await load();
    } catch (e) {
      setError(errorText(e));
    } finally {
      setBusy(false);
    }
  }

  async function sendNow() {
    setBusy(true);
    setError("");
    try {
      const res = await api<{ target: number }>("/api/v1/admin/store-message/send", {
        method: "POST",
        body: "{}",
      });
      setOk(res.target > 0 ? fill(S.sentStarted, { n: fmtNum(res.target) }) : S.nothingToSend);
      setConfirming(false);
      await load();
    } catch (e) {
      setError(errorText(e));
      setConfirming(false);
    } finally {
      setBusy(false);
    }
  }

  async function sendTest() {
    setBusy(true);
    setError("");
    try {
      await api("/api/v1/admin/store-message/test", {
        method: "POST",
        body: JSON.stringify({ merchant_id: testId }),
      });
      setOk(S.testStarted);
      await load();
    } catch (e) {
      setError(errorText(e));
    } finally {
      setBusy(false);
    }
  }

  const stopped = st?.runs[0]?.status === "stopped" ? st.runs[0] : null;
  const toSend = st ? Math.max(0, st.eligible - st.already_today) : 0;
  const days = S.weekdays as unknown as string[];

  return (
    <AdminPage
      title={S.title}
      icon={IconWhatsApp}
      notices={
        <>
          {st && !st.bot_ready && <Alert tone="error">{S.botDown}</Alert>}
          {stopped && (
            <Alert tone="error">
              {fill(S.stoppedAlert, { sent: fmtNum(stopped.sent), target: fmtNum(stopped.target) })}
            </Alert>
          )}
          {ok && <Alert tone="success">{ok}</Alert>}
          {error && <Alert tone="error">{error}</Alert>}
        </>
      }
    >
      <p className="mb-4 text-sm text-ink-muted">{S.hint}</p>

      {cfg && (
        <div className="grid grid-cols-1 gap-4 xl:grid-cols-2">
          <FormSection title={S.messageTitle} icon={<IconWhatsApp />}>
            <div className="space-y-3">
              <Input
                id="sm-title"
                label={S.titleField}
                value={cfg.title}
                maxLength={80}
                onChange={(e) => edit({ title: e.target.value })}
              />
              <Textarea
                id="sm-text"
                label={S.textField}
                rows={9}
                value={cfg.text}
                onChange={(e) => edit({ text: e.target.value })}
              />

              {/* **الجدولةُ الآليّة** — مطفأةٌ افتراضاً، بتوقيت دمشق. */}
              <div className="rounded-control border border-line p-3">
                <p className="mb-2 text-sm font-bold">{S.autoTitle}</p>
                <Checkbox
                  id="sm-auto"
                  checked={cfg.auto_enabled}
                  onChange={(e) => edit({ auto_enabled: e.target.checked })}
                  label={cfg.auto_enabled ? S.autoEnabled : S.autoOff}
                />
                {cfg.auto_enabled && (
                  <div className="mt-3 grid grid-cols-1 gap-3 sm:grid-cols-3">
                    <Select
                      id="sm-freq"
                      label={S.frequency}
                      value={cfg.auto_frequency}
                      onChange={(e) => edit({ auto_frequency: e.target.value as Config["auto_frequency"] })}
                    >
                      <option value="daily">{S.daily}</option>
                      <option value="weekly">{S.weekly}</option>
                    </Select>
                    {cfg.auto_frequency === "weekly" && (
                      <Select
                        id="sm-day"
                        label={S.weekday}
                        value={String(cfg.auto_weekday)}
                        onChange={(e) => edit({ auto_weekday: Number(e.target.value) })}
                      >
                        {days.map((d, i) => (
                          <option key={d} value={i}>{d}</option>
                        ))}
                      </Select>
                    )}
                    <Select
                      id="sm-hour"
                      label={S.hour}
                      value={String(cfg.auto_hour)}
                      onChange={(e) => edit({ auto_hour: Number(e.target.value) })}
                    >
                      {Array.from({ length: 24 }, (_, h) => (
                        <option key={h} value={h}>{String(h).padStart(2, "0")}:00</option>
                      ))}
                    </Select>
                  </div>
                )}
              </div>

              <div className="flex items-center justify-end gap-3">
                {dirty && <span className="text-xs text-warning">{S.dirty}</span>}
                <Button disabled={busy || !dirty} onClick={() => void save()}>{S.save}</Button>
              </div>
            </div>
          </FormSection>

          <div className="space-y-4">
            <FormSection title={S.sendTitle} icon={<IconWhatsApp />}>
              {st && (
                <p className="mb-3 rounded-control bg-ink-faint px-3 py-2 text-sm text-ink-muted">
                  {st.eligible === 0
                    ? S.noOne
                    : fill(S.willReach, { n: fmtNum(toSend), already: fmtNum(st.already_today) })}
                </p>
              )}
              {/* **ولا يُرسل الحقلُ غيرَ المحفوظ**: الخادمُ يقرأ المحفوظَ — فتُحفظ التعديلاتُ أوّلاً. */}
              <div className="flex justify-end">
                <Button
                  disabled={busy || dirty || !st?.bot_ready || toSend === 0}
                  onClick={() => setConfirming(true)}
                >
                  {S.sendNow}
                </Button>
              </div>
            </FormSection>

            <FormSection title={S.testTitle} icon={<IconWhatsApp />}>
              <div className="flex flex-wrap items-end gap-3">
                <div className="min-w-[14rem] flex-1">
                  <Select
                    id="sm-test"
                    label={S.testPick}
                    value={testId}
                    onChange={(e) => setTestId(e.target.value)}
                  >
                    <option value="">—</option>
                    {stores.map((x) => (
                      <option key={x.id} value={x.id}>{x.name}</option>
                    ))}
                  </Select>
                </div>
                <Button
                  disabled={busy || dirty || !testId || !st?.bot_ready}
                  onClick={() => void sendTest()}
                >
                  {S.testSend}
                </Button>
              </div>
            </FormSection>
          </div>
        </div>
      )}

      <div className="mt-4">
      <FormSection title={S.historyTitle} icon={<IconWhatsApp />}>
        {st && st.runs.length === 0 ? (
          <p className="text-sm text-ink-muted">{S.noRuns}</p>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead>
                <tr className="text-start text-ink-muted">
                  <th className="p-2 text-start">{S.colTime}</th>
                  <th className="p-2 text-start">{S.colTrigger}</th>
                  <th className="p-2 text-start">{S.colTarget}</th>
                  <th className="p-2 text-start">{S.colSent}</th>
                  <th className="p-2 text-start">{S.colFailed}</th>
                  <th className="p-2 text-start">{S.colStatus}</th>
                </tr>
              </thead>
              <tbody>
                {st?.runs.map((r) => (
                  <tr key={r.id} className="border-t border-line-soft">
                    <td className="p-2">{fmtDateTime(r.started_at)}</td>
                    <td className="p-2">{S.trigger[r.trigger]}</td>
                    <td className="p-2">{fmtNum(r.target)}</td>
                    <td className="p-2">{fmtNum(r.sent)}</td>
                    <td className="p-2">{fmtNum(r.failed)}</td>
                    <td className="p-2">
                      <Badge variant={STATUS_TONE[r.status]}>{S.status[r.status]}</Badge>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </FormSection>
      </div>

      {/* **تأكيدٌ بالرقم قبل الإرسال** — ما أُرسل لا يُسحب. */}
      <Modal open={confirming} onClose={() => setConfirming(false)} title={S.confirmTitle}>
        <p className="mb-4 text-sm">{fill(S.confirmBody, { n: fmtNum(toSend) })}</p>
        <FormActions
          onSave={() => void sendNow()}
          onCancel={() => setConfirming(false)}
          busy={busy}
          saveLabel={S.confirmSend}
          tone="danger"
        />
      </Modal>
    </AdminPage>
  );
}
