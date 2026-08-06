"use client";

/**
 * **إعلانُ المنصة — أن تخاطب أهلَها.**
 *
 * كلُّ إشعارٍ في المنصة يُولد من واقعة: طلبٌ قُبل، مالٌ دخل، شكوى فُتحت.
 * **ولا سبيلَ لقول شيءٍ لا واقعةَ له** — «مغلقون اليومَ للصيانة» · «تأخّرٌ
 * عامٌّ بسبب المطر» · «كلُّ سائقٍ يسلّم صندوقَه قبل السادسة».
 *
 * **ومنصّةُ توصيلٍ لا تملك أن تخاطب زبائنَها تُدير أزمتَها بالهاتف** — واحداً
 * واحداً.
 *
 * # والعددُ يُرى قبل الإرسال
 *
 * **ومن ظنّ أنّه يخاطب ثلاثةَ سائقين فوجد ألفَ زبونٍ قد وصلتهم رسالتُه لا يملك
 * أن يسحبها.** فالعددُ يُقرأ من الخادم مع كلّ تغيّرٍ في الاختيار، **ويُطلب
 * تأكيدٌ عليه بالرقم.**
 */

import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale, fmtNum } from "@rahalgo/i18n";
import {
  Alert, Button, Input, Modal, FormSection, Checkbox, IconWhatsApp } from "@rahalgo/ui";
import { api, ApiError } from "@/lib/api";

const m = getMessages(defaultLocale);
const B = m.admin.broadcast;
const ROLE_LABELS: Record<string, string> = m.terms.roleNames;

const ROLES = ["customer", "driver", "merchant", "sales"] as const;

export default function BroadcastPanel() {
  const [roles, setRoles] = useState<string[]>([]);
  const [title, setTitle] = useState("");
  const [body, setBody] = useState("");
  const [count, setCount] = useState<number | null>(null);
  const [confirming, setConfirming] = useState(false);
  const [busy, setBusy] = useState(false);
  const [sent, setSent] = useState("");
  const [error, setError] = useState("");

  const loadCount = useCallback(async () => {
    if (roles.length === 0) {
      setCount(null);
      return;
    }
    try {
      const res = await api<{ count: number }>(
        `/api/v1/admin/broadcast/count?roles=${roles.join(",")}`,
      );
      setCount(res.count);
    } catch {
      setCount(null);
    }
  }, [roles]);

  useEffect(() => {
    void loadCount();
  }, [loadCount]);

  function toggle(r: string) {
    setSent("");
    setRoles((cur) => (cur.includes(r) ? cur.filter((x) => x !== r) : [...cur, r]));
  }

  async function send() {
    setBusy(true);
    setError("");
    try {
      const res = await api<{ sent: number }>("/api/v1/admin/broadcast", {
        method: "POST",
        body: JSON.stringify({ roles, title: title.trim(), body: body.trim() }),
      });
      setSent(B.done.replace("{n}", fmtNum(res.sent)));
      setTitle("");
      setBody("");
      setConfirming(false);
    } catch (err) {
      setError(
        err instanceof ApiError
          ? ((m.errors as Record<string, string>)[
              err.body.message_key.split(".").pop() ?? ""
            ] ?? m.errors.internal)
          : m.errors.internal,
      );
      setConfirming(false);
    } finally {
      setBusy(false);
    }
  }

  const ready = roles.length > 0 && title.trim() !== "" && (count ?? 0) > 0;

  return (
    <FormSection title={B.title} icon={<IconWhatsApp />}>
      <p className="mb-3 text-xs text-ink-muted">{B.hint}</p>

      {/* **بالدور لا بالكلّ.** رسالةٌ عن تسليم الصناديق تصل زبوناً لا شأنَ له
          بها، **فيتعلّم أن إشعاراتِ المنصة لا تُقرأ** — ثمّ لا يقرأ ما يهمّه. */}
      <div className="mb-3 flex flex-wrap gap-3">
        {ROLES.map((r) => (
          <Checkbox
            key={r}
            checked={roles.includes(r)}
            onChange={() => toggle(r)}
            label={ROLE_LABELS[r] ?? r}
          />
        ))}
      </div>

      <div className="space-y-3">
        <Input
          id="bc-title"
          label={B.titleField}
          required
          value={title}
          onChange={(e) => {
            setTitle(e.target.value);
            setSent("");
          }}
        />
        <Input
          id="bc-body"
          label={B.bodyField}
          value={body}
          onChange={(e) => {
            setBody(e.target.value);
            setSent("");
          }}
        />

        {count != null && (
          <p
            className={`rounded-control px-3 py-2 text-sm ${
              count > 100 ? "bg-warning-tint font-bold text-warning" : "bg-ink-faint text-ink-muted"
            }`}
          >
            {B.willReach.replace("{n}", fmtNum(count))}
          </p>
        )}
        {error && <p className="text-sm text-danger">{error}</p>}
        {sent && (
          <Alert tone="success">{sent}</Alert>
        )}

        <div className="flex justify-end">
          <Button disabled={!ready || busy} onClick={() => setConfirming(true)}>
            {B.send}
          </Button>
        </div>
      </div>

      {/* **ولا يُرسل بضغطةٍ واحدة.** ما لا يُسحب يلزمه تأكيدٌ بالرقم. */}
      <Modal open={confirming} onClose={() => setConfirming(false)} title={B.confirmTitle}>
        <p className="mb-4 text-sm">
          {B.confirmBody.replace("{n}", fmtNum(count ?? 0))}
        </p>
        <p className="mb-4 rounded-control bg-field px-3 py-2 text-sm font-medium">{title}</p>
        <div className="flex justify-end gap-2">
          <Button variant="secondary" onClick={() => setConfirming(false)}>
            {m.common.cancel}
          </Button>
          <Button variant="danger" disabled={busy} onClick={() => void send()}>
            {B.confirmSend}
          </Button>
        </div>
      </Modal>
    </FormSection>
  );
}
