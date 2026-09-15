"use client";

/**
 * **مركزُ الإشعارات — يُعايَن ثمّ يُرسَل أو يُجدوَل.**
 *
 * # وما الفرقُ عن «إعلان المنصّة»
 *
 * **والإعلانُ يُرسل في الحال ولا يحفظ شيئاً** — **ولا جدولةَ ولا
 * إلغاءَ ولا أثرَ يُقرأ بعد شهر.** **وهذه تحفظ الحملةَ صفّاً**: تُجدوَل،
 * وتُلغى قبل موعدها، **ويبقى بعدها ما يُقرأ: كم استُهدف وكم وصل.**
 *
 * # والعددُ يُرى قبل الإرسال
 *
 * **ومن ظنّ أنّه يخاطب عشرةً فوجد ألفاً قد وصلتهم رسالتُه لا يملك أن
 * يسحبها.**
 *
 * # ولا عددَ «وصل» ولا «قُرئ»
 *
 * **وقبولُ غوغل للرسالة ليس قراءةَ إنسان** — **ومن كتب «وصلت» بناءً
 * عليه كذب برقم.** **فيُعرض ما يُعرَف: استُهدف، وكُتب في الصندوق،
 * وأُجّل.**
 */

import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale, fmtNum } from "@rahalgo/i18n";
import { roleLabelByCode } from "@/lib/rolemeta";
import {
  Alert, Button, Input, Modal, FormSection, FormActions, IconWhatsApp,
} from "@rahalgo/ui";
import { api, ApiError } from "@/lib/api";

const m = getMessages(defaultLocale);
const C = m.admin.campaigns;

const ROLES = ["customer", "driver", "merchant", "sales"] as const;

type Campaign = {
  id: string;
  title: string;
  audience_type: string;
  audience_ref: string;
  status: string;
  scheduled_at: string | null;
  targeted: number;
  inbox_created: number;
  deferred: number;
};

/** **ولفظُ الحال عربيٌّ** — **ولا يُطبَع رمزٌ آليٌّ على شاشة.** */
function statusLabel(s: string): string {
  if (s === "draft") return C.statusDraft;
  if (s === "scheduled") return C.statusScheduled;
  if (s === "sending") return C.statusSending;
  if (s === "sent") return C.statusSent;
  if (s === "cancelled") return C.statusCancelled;
  return C.statusFailed;
}

export default function CampaignsPanel() {
  const [rows, setRows] = useState<Campaign[]>([]);
  const [audienceType, setAudienceType] = useState<"role" | "service_interest">("role");
  const [audienceRef, setAudienceRef] = useState("customer");
  const [title, setTitle] = useState("");
  const [body, setBody] = useState("");
  const [destType, setDestType] = useState<"home" | "offer">("home");
  const [destId, setDestId] = useState("");
  const [when, setWhen] = useState("");
  const [count, setCount] = useState<number | null>(null);
  const [quiet, setQuiet] = useState(false);
  const [confirming, setConfirming] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const load = useCallback(async () => {
    try {
      const res = await api<{ campaigns: Campaign[] }>("/api/v1/admin/campaigns");
      setRows(res.campaigns ?? []);
    } catch {
      setRows([]);
    }
  }, []);

  /** **والعددُ يُقرأ من الخادم** — **ولا يُحسب في المتصفّح.** */
  const loadCount = useCallback(async () => {
    if (audienceRef.trim() === "") {
      setCount(null);
      return;
    }
    try {
      const res = await api<{ count: number; quiet_now: boolean }>(
        `/api/v1/admin/campaigns/preview?audience_type=${audienceType}` +
          `&audience_ref=${encodeURIComponent(audienceRef.trim())}`,
      );
      setCount(res.count);
      setQuiet(res.quiet_now);
    } catch {
      setCount(null);
    }
  }, [audienceType, audienceRef]);

  useEffect(() => {
    void load();
  }, [load]);

  useEffect(() => {
    void loadCount();
  }, [loadCount]);

  function fail(err: unknown) {
    setError(
      err instanceof ApiError
        ? ((m.errors as Record<string, string>)[
            err.body.message_key.split(".").pop() ?? ""
          ] ?? m.errors.internal)
        : m.errors.internal,
    );
  }

  /** **يحفظ الحملةَ** — **مسوّدةً أو مجدولةً بحسب الموعد.** */
  async function create(): Promise<string> {
    const res = await api<{ id: string }>("/api/v1/admin/campaigns", {
      method: "POST",
      body: JSON.stringify({
        title: title.trim(),
        body: body.trim(),
        audience_type: audienceType,
        audience_ref: audienceRef.trim(),
        dest_type: destType,
        dest_id: destType === "home" ? "" : destId.trim(),
        // **والموعدُ يُرسَل لحظةً عالميّةً صريحة** — **والخادمُ هو
        // الحَكَم، وساعةُ المتصفّح للعرض وحدَها.**
        scheduled_at: when === "" ? null : new Date(when).toISOString(),
      }),
    });
    return res.id;
  }

  async function saveOnly() {
    setBusy(true);
    setError("");
    try {
      await create();
      setTitle("");
      setBody("");
      setWhen("");
      await load();
    } catch (err) {
      fail(err);
    } finally {
      setBusy(false);
    }
  }

  async function sendNow() {
    setBusy(true);
    setError("");
    try {
      const id = await create();
      await api(`/api/v1/admin/campaigns/${id}/send`, { method: "POST", body: "{}" });
      setTitle("");
      setBody("");
      setConfirming(false);
      await load();
    } catch (err) {
      fail(err);
      setConfirming(false);
    } finally {
      setBusy(false);
    }
  }

  async function cancel(id: string) {
    setBusy(true);
    setError("");
    try {
      await api(`/api/v1/admin/campaigns/${id}/cancel`, { method: "POST", body: "{}" });
      await load();
    } catch (err) {
      fail(err);
    } finally {
      setBusy(false);
    }
  }

  const ready = title.trim() !== "" && audienceRef.trim() !== "" && (count ?? 0) > 0;

  return (
    <FormSection title={C.title} icon={<IconWhatsApp />}>
      <p className="mb-3 text-xs text-ink-muted">{C.hint}</p>

      <div className="mb-3 flex flex-wrap gap-2">
        {/* **والجمهورُ يُختار من جدولٍ نعرفه** — **ولا شرطٌ يُكتب.** */}
        {ROLES.map((r) => (
          <Button
            key={r}
            variant={audienceType === "role" && audienceRef === r ? "primary" : "ghost"}
            onClick={() => {
              setAudienceType("role");
              setAudienceRef(r);
            }}
          >
            {roleLabelByCode(r)}
          </Button>
        ))}
        <Button
          variant={audienceType === "service_interest" ? "primary" : "ghost"}
          onClick={() => {
            setAudienceType("service_interest");
            setAudienceRef("");
          }}
        >
          {C.audienceInterest}
        </Button>
      </div>

      {audienceType === "service_interest" && (
        <Input
          id="cp-ref"
          label={C.audienceRef}
          value={audienceRef}
          onChange={(e) => setAudienceRef(e.target.value)}
        />
      )}

      <div className="space-y-3">
        <Input
          id="cp-title"
          label={C.titleField}
          required
          value={title}
          onChange={(e) => setTitle(e.target.value)}
        />
        <Input
          id="cp-body"
          label={C.bodyField}
          value={body}
          onChange={(e) => setBody(e.target.value)}
        />

        <div className="flex flex-wrap gap-2">
          {/*
            **والوجهةُ نوعٌ نعرفه** — **ولا رابطٌ حرّ.**

            **و«المتجر» رُفعت** (قرارُ المالك ٢٠٢٦-٠٩-١٥): **ولا شاشةَ
            متجرٍ عند الزبون** — **قرارُه ٢٠٢٦-٠٨-٠٥**: «المتاجرُ
            مخفيّةٌ عن الزبون بالكامل». **ووجهةٌ لا يملك التطبيقُ أن
            يفتحها وعدٌ يُرسَل في جيبه ثمّ لا يُوفى.**

            **والمنعُ في المحرّك** (`campaigns.ValidDest`) — **وهذه
            لوحةٌ لا تعرض ما يُردّ.**
          */}
          {(["home", "offer"] as const).map((d) => (
            <Button
              key={d}
              variant={destType === d ? "primary" : "ghost"}
              onClick={() => setDestType(d)}
            >
              {d === "home" ? C.destHome : C.destOffer}
            </Button>
          ))}
        </div>
        {destType !== "home" && (
          <Input
            id="cp-dest"
            label={C.destId}
            value={destId}
            onChange={(e) => setDestId(e.target.value)}
          />
        )}

        <Input
          id="cp-when"
          type="datetime-local"
          label={C.scheduledAt}
          value={when}
          onChange={(e) => setWhen(e.target.value)}
        />

        {count != null && (
          <p
            className={`rounded-control px-3 py-2 text-sm ${
              count > 100 ? "bg-warning-tint font-bold text-warning" : "bg-ink-faint text-ink-muted"
            }`}
          >
            {C.willReach.replace("{n}", fmtNum(count))}
          </p>
        )}
        {quiet && <Alert tone="warning">{C.quietNow}</Alert>}
        {error && <p className="text-sm text-danger">{error}</p>}

        <div className="flex justify-end gap-2">
          <Button variant="ghost" disabled={!ready || busy} onClick={() => void saveOnly()}>
            {C.create}
          </Button>
          <Button disabled={!ready || busy || when !== ""} onClick={() => setConfirming(true)}>
            {C.sendNow}
          </Button>
        </div>
      </div>

      {/* **ولا يُرسل بضغطةٍ واحدة** — **ما لا يُسحب يلزمه تأكيد.** */}
      <Modal open={confirming} onClose={() => setConfirming(false)} title={C.confirmTitle}>
        <p className="mb-4 text-sm">{C.confirmBody.replace("{n}", fmtNum(count ?? 0))}</p>
        <p className="mb-4 rounded-control bg-field px-3 py-2 text-sm font-medium">{title}</p>
        <FormActions
          onSave={() => void sendNow()}
          onCancel={() => setConfirming(false)}
          busy={busy}
          saveLabel={C.confirmSend}
          tone="danger"
        />
      </Modal>

      <h3 className="mb-2 mt-6 text-sm font-bold">{C.history}</h3>
      {rows.length === 0 ? (
        <p className="text-sm text-ink-muted">{C.empty}</p>
      ) : (
        <ul className="space-y-2">
          {rows.map((c) => (
            <li key={c.id} className="rounded-control bg-field px-3 py-2 text-sm">
              <div className="flex items-center justify-between gap-2">
                <span className="font-medium">{c.title}</span>
                <span className="text-xs text-ink-muted">{statusLabel(c.status)}</span>
              </div>
              <div className="mt-1 text-xs text-ink-muted">
                {C.targeted}: {fmtNum(c.targeted)} · {C.inboxCreated}: {fmtNum(c.inbox_created)}
                {c.deferred > 0 ? ` · ${C.deferred}: ${fmtNum(c.deferred)}` : ""}
              </div>
              {(c.status === "draft" || c.status === "scheduled") && (
                <div className="mt-2 flex justify-end">
                  <Button variant="ghost" disabled={busy} onClick={() => void cancel(c.id)}>
                    {C.cancel}
                  </Button>
                </div>
              )}
            </li>
          ))}
        </ul>
      )}
    </FormSection>
  );
}
